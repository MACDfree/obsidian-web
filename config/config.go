package config

import (
	"io"
	"obsidian-web/logger"
	"os"
	"sync"

	"github.com/pkg/errors"
	"gopkg.in/yaml.v3"
)

type Config struct {
	NotePath       string   `yaml:"note_path"`
	IgnorePaths    []string `yaml:"ignore_paths"`
	AttachmentPath string   `yaml:"attachment_path"`
	Paginate       int      `yaml:"paginate"`
	Title          string   `yaml:"title"`
	Password       string   `yaml:"password"`
	BindAddr       string   `yaml:"bind_addr"`
}

var cfg Config

func load() {
	file, err := os.Open("config.yml")
	if err != nil {
		logger.Fatal(errors.WithStack(err))
	}
	bs, err := io.ReadAll(file)
	if err != nil {
		logger.Fatal(errors.WithStack(err))
	}

	err = yaml.Unmarshal(bs, &cfg)
	if err != nil {
		logger.Fatal(errors.WithStack(err))
	}
}

func Get() Config {
	sync.OnceFunc(load)()
	return cfg
}
