package main

import (
	"context"
	"crypto/rand"
	"fmt"
	"io/fs"

	"path/filepath"
	"sync"
	"time"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/client"
	"github.com/fsnotify/fsnotify"
	"github.com/ryanuber/go-glob"
	log "github.com/sirupsen/logrus"
)

// TODO: per-path mutexes
type Watcher struct {
	cfg      WatcherConfig
	client   *client.Client
	hashes   map[string]string
	hashesMu sync.Mutex
	dirs     []string
	salt     []byte
}

func (p *PathSpec) simplify() error {
	if p.File != "" {
		p.Dir = filepath.Dir(p.File)
		p.Globs = []string{filepath.Base(p.File)}
	}

	if p.Dir == "" {
		return fmt.Errorf("Empty directory")
	}

	return nil
}

// configure populates watcher's configuration, setting defaults and
// calculating missing values
func (w *Watcher) configure() (count int, err error) {
	w.dirs = make([]string, 0, len(w.cfg.PathSpec))

	hashFunc, err := parseHashFunction(w.cfg.Hash)
	if err != nil {
		log.Errorf("Error parsing hash function %v", err)
		return 0, nil
	}
	w.salt = make([]byte, hashFunc.BlockSize())
	_, err = rand.Reader.Read(w.salt)
	if err != nil {
		return
	}

	for _, path := range w.cfg.PathSpec {
		err := path.simplify()
		if err != nil {
			log.Errorf("Error parsing pathspec %v: %v", path, err)
			continue
		}

		var hashed = false
		log.Tracef("Walking %v", path)
		filepath.WalkDir(path.Dir, func(filePath string, info fs.DirEntry, err error) error {
			if err != nil {
				log.Warnf("Error walking path %s: %v", filePath, err)
			}

			if info.IsDir() {
				// TODO: recurse
			} else {
				var matches = len(path.Globs) == 0
				for _, g := range path.Globs {
					if glob.Glob(g, info.Name()) {
						matches = true
						log.Debugf("Path %s matches glob %s", info.Name(), g)
						break
					}
				}
				if !matches {
					log.Tracef("File %s did not match any glob", filePath)
					return nil
				}

				hash, err := fileHash(filePath, w.cfg.Hash, w.salt)
				if err != nil {
					log.Errorf("Error hashing file %s: %v", filePath, err)
					return err
				}

				p, err := filepath.Abs(filePath)
				if err != nil {
					log.Errorf("Error getting absolute path to file %s: %v", filePath, err)
					p = filePath
				}
				w.hashesMu.Lock()
				w.hashes[p] = hash
				w.hashesMu.Unlock()
				log.Debugf("Hashed file: %s", filePath)
				count++
				hashed = true
			}
			return nil
		})
		if hashed {
			d, err := filepath.Abs(path.Dir)
			if err != nil {
				d = path.Dir
			}
			w.dirs = append(w.dirs, d)
		}
	}

	return
}

// run starts the watcher and listens for file changes
func (w *Watcher) run() {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		log.Errorf("Error creating watcher: %v", err)
		return
	}
	defer watcher.Close()

	for _, path := range w.dirs {
		if err := watcher.Add(path); err != nil {
			log.Errorf("Error watching dir %s: %v", path, err)
			continue
		}
		log.Debugf("Watching dir %s", path)
	}

	for {
		select {
		case event, ok := <-watcher.Events:
			log.Tracef("Receved event: %v", event)
			if !ok {
				return
			}

			if event.Op&(fsnotify.Write|fsnotify.Create) == 0 {
				continue
			}

			w.handleEvent(event.Name)
			log.Debugf("Handled event: %v", event)

		case err, ok := <-watcher.Errors:
			if !ok {
				return
			}
			log.Errorf("Watcher error: %v", err)
		}
	}
}

// handleEvent processes a file change event
func (w *Watcher) handleEvent(filePath string) {
	// since hash entries are never deleted we don't need to lock for existence
	// check
	_, exists := w.hashes[filePath]

	if !exists {
		return
	}

	w.hashesMu.Lock()
	defer w.hashesMu.Unlock()

	currentHash, err := fileHash(filePath, w.cfg.Hash, w.salt)
	if err != nil {
		log.Errorf("Error hashing file %s: %v", filePath, err)
		return
	}

	previousHash, exists := w.hashes[filePath]

	if previousHash == currentHash {
		return
	}
	log.Tracef("Hashes differ, was: %s have: %s", previousHash, currentHash)

	w.hashes[filePath] = currentHash
	log.Infof("Detected change in %s", filePath)
	w.triggerAction()
}

func (w *Watcher) triggerAction() {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	filter := filters.NewArgs()
	if w.cfg.Selector.Name != "" {
		filter.Add("name", w.cfg.Selector.Name)
	}
	if w.cfg.Selector.Label != "" {
		filter.Add("label", w.cfg.Selector.Label)
	}

	containers, err := w.client.ContainerList(ctx, container.ListOptions{
		Filters: filter,
	})
	if err != nil {
		log.Errorf("Error listing containers: %v", err)
		return
	}

	if len(containers) == 0 {
		log.Warnf("No containers found for selector: %+v", w.cfg.Selector)
		return
	}

	for _, ctr := range containers {
		switch w.cfg.Action {
		case "restart":
			log.Infof("Restarting container %s (%s)", ctr.Names[0], ctr.ID[:12])
			if err := w.client.ContainerRestart(ctx, ctr.ID, container.StopOptions{}); err != nil {
				log.Errorf("Error restarting container %s: %v", ctr.ID[:12], err)
			}
		case "sighup":
			log.Infof("Sending SIGHUP to container %s (%s)", ctr.Names[0], ctr.ID[:12])
			if err := w.client.ContainerKill(ctx, ctr.ID, "SIGHUP"); err != nil {
				log.Errorf("Error sending SIGHUP to container %s: %v", ctr.ID[:12], err)
			}
		default:
			log.Errorf("Unknown action: %s", w.cfg.Action)
		}
	}
}
