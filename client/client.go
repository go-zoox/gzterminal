package client

import (
	"fmt"
	"os"

	"github.com/go-zoox/fetch"
)

type Client interface {
	Run(cfg *Config) error
}

type Config struct {
	Server   string
	Exec     string
	Username string
	Password string
}

func Run(cfg *Config) error {
	response, err := fetch.Post("/api/exec", &fetch.Config{
		BaseURL: cfg.Server,
		Body:    cfg.Exec,
	})
	if err != nil {
		return fmt.Errorf("failed to exec with terminal client: %s", err)
	}

	if ok := response.Ok(); !ok {
		exitCode := response.Get("exit_code").Int()
		exitMessage := response.Get("exit_message").String()
		os.Stderr.Write([]byte(exitMessage))
		os.Exit(int(exitCode))
		return nil
	}

	_, err = os.Stdout.Write([]byte(response.String()))
	return err
}
