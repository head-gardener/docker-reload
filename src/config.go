package main

import (
	"flag"
	"os"

	log "github.com/sirupsen/logrus"
	"gopkg.in/yaml.v3"
)

type Config struct {
	Watchers []WatcherConfig `yaml:"watchers"`
	Default  WatcherConfig   `yaml:"default"`
}

type WatcherConfig struct {
	PathSpec []PathSpec `yaml:"paths"`
	Selector Selector   `yaml:"selector"`
	Action   string     `yaml:"action"`
	Hash     string     `default:"sha256"`
}

type PathSpec struct {
	Dir   string   `yaml:"dir"`
	Globs []string `yaml:"globs"`
	File  string   `yaml:"file"`
}

type Selector struct {
	Name  string `yaml:"name,omitempty"`
	Label string `yaml:"label,omitempty"`
}

func (w *WatcherConfig) applyDefault(def *WatcherConfig) {
	if w.Hash == "" {
		w.Hash = def.Hash
	}
	if w.Action == "" {
		w.Action = def.Action
	}
	if w.Selector.Name == "" && w.Selector.Label == "" {
		w.Selector = def.Selector
	}
	if len(w.PathSpec) == 0 {
		w.PathSpec = def.PathSpec
	}
}

func (w *WatcherConfig) makeDefault() {
	w.Action = "restart"
	w.Hash = "sha256"
}

func NewConfig() *Config {
	var config Config
	config.Default.makeDefault()

	configFile := flag.String("config", "./config.yml", "config path")
	logLevel := flag.String("log-level", "info", "log level")
	flag.Parse()

	lvl, err := log.ParseLevel(*logLevel)
	if err != nil {
		log.Fatalf("Invalid log level: %v", *logLevel)
	}
	log.SetLevel(lvl)

	configData, err := os.ReadFile(*configFile)
	if err != nil {
		log.Fatalf("Error reading config file: %v", err)
	}

	if err := yaml.Unmarshal(configData, &config); err != nil {
		log.Fatalf("Error parsing config file: %v", err)
	}

	for i := range config.Watchers {
		config.Watchers[i].applyDefault(&config.Default)
	}

	return &config
}
