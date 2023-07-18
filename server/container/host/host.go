package host

import "github.com/go-zoox/gzterminal/server/container"

type Host interface {
	container.Container
}

type host struct {
	cfg *Config
}

func New(cfg *Config) Host {
	return &host{
		cfg: cfg,
	}
}
