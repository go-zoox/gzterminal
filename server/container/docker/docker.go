package docker

import "github.com/go-zoox/gzterminal/server/container"

type Docker interface {
	container.Container
}

type docker struct {
	cfg *Config
}

func New(cfg *Config) Docker {
	if cfg.Image == "" {
		cfg.Image = "whatwewant/zmicro:v1"
	}

	return &docker{
		cfg: cfg,
	}
}
