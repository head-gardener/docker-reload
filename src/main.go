package main

import (
	"context"

	"sync"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/client"
	log "github.com/sirupsen/logrus"
)

func main() {
	config := NewConfig()
	log.Trace("Config loaded")

	dockerClient, err := client.NewClientWithOpts(client.FromEnv)
	if err != nil {
		log.Panicf("Error creating Docker client: %v", err)
	}
	log.Trace("Client initialized")

	// check if client is working
	_, err = dockerClient.ContainerList(context.Background(), container.ListOptions{All: false})
	if err != nil {
		log.Fatalf("Can't list contianers: %v", err)
	}

	var watcherCount int
	var fileCount int
	var wg sync.WaitGroup
	for _, watcherCfg := range config.Watchers {
		w := &Watcher{
			cfg:    watcherCfg,
			client: dockerClient,
			hashes: make(map[string]string),
		}

		c, err := w.configure()
		if err != nil {
			log.Errorf("Error initializing watcher: %v", err)
			continue
		}
		log.Tracef("Watcher initialized with config: %v", watcherCfg)

		wg.Add(1)
		watcherCount++
		fileCount += c
		go func(w *Watcher) {
			defer wg.Done()
			w.run()
		}(w)
	}

	log.Infof("Started %d watchers, %d files total", watcherCount, fileCount)
	wg.Wait()
	log.Tracef("All watchers exited, stopping")
}
