package server

import (
	"fmt"
	"io"

	"github.com/go-zoox/gzterminal/message"
	"github.com/go-zoox/gzterminal/server/container/docker"
	"github.com/go-zoox/gzterminal/server/container/host"
	"github.com/go-zoox/gzterminal/server/session"
	"github.com/go-zoox/logger"
	"github.com/go-zoox/zoox"
	"github.com/go-zoox/zoox/components/application/websocket"
)

type ConnectConfig struct {
	Container   string
	Shell       string
	Environment map[string]string
	WorkDir     string
	InitCommand string
	//
	Image string
}

func connect(ctx *zoox.Context, client *websocket.Client, cfg *ConnectConfig) (session session.Session, err error) {
	if cfg.Container == "" {
		cfg.Container = "host"
	}

	if cfg.Container == "host" {
		if session, err = host.New(&host.Config{
			Shell:       cfg.Shell,
			Environment: cfg.Environment,
			WorkDir:     cfg.WorkDir,
			InitCommand: cfg.InitCommand,
		}).Connect(ctx.Context()); err != nil {
			ctx.Logger.Errorf("[websocket] failed to connect host: %s", err)
			client.Disconnect()
			return
		}
	} else if cfg.Container == "docker" {
		if session, err = docker.New(&docker.Config{
			Shell:       cfg.Shell,
			Environment: cfg.Environment,
			WorkDir:     cfg.WorkDir,
			InitCommand: cfg.InitCommand,
			//
			Image: cfg.Image,
		}).Connect(ctx.Context()); err != nil {
			ctx.Logger.Errorf("[websocket] failed to connect container: %s", err)
			client.Disconnect()
			return
		}
	} else {
		panic(fmt.Errorf("unknown mode: %s", cfg.Container))
	}

	go func() {
		if err := session.Wait(); err != nil {
			logger.Errorf("Failed to wait session: %s", err)
		}

		client.Disconnect()
	}()

	go func() {
		buf := make([]byte, 1024)
		for {
			n, err := session.Read(buf)
			if err != nil && err != io.EOF {
				logger.Errorf("failed to read from session: %s", err)
				client.WriteMessage(websocket.BinaryMessage, []byte(err.Error()))
				return
			}

			msg := &message.Message{}
			msg.SetType(message.TypeOutput)
			msg.SetOutput(buf[:n])
			if err := msg.Serialize(); err != nil {
				logger.Errorf("failed to serialize message: %s", err)
				return
			}

			client.WriteMessage(websocket.BinaryMessage, msg.Msg())

			if err == io.EOF {
				return
			}
		}
	}()

	return
}