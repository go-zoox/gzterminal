package server

import (
	"encoding/json"
	"fmt"

	"github.com/go-zoox/gzterminal/server/container"
	"github.com/go-zoox/gzterminal/server/session"
	"github.com/go-zoox/logger"
	"github.com/go-zoox/zoox"
	"github.com/go-zoox/zoox/components/application/websocket"
	"github.com/go-zoox/zoox/defaults"
)

type Server interface {
	Run() error
}

type Config struct {
	Port     int64
	Shell    string
	Username string
	Password string
	// Container is the Container runtime, options: host, docker, kubernetes, ssh, default: host
	Container string
	//
	Path string
}

type server struct {
	cfg *Config
}

func New(cfg *Config) Server {
	if cfg.Container == "" {
		cfg.Container = "host"
	}

	if cfg.Path == "" {
		cfg.Path = "/ws"
	}

	return &server{
		cfg: cfg,
	}
}

func (s *server) Run() error {
	cfg := s.cfg
	addr := fmt.Sprintf(":%d", cfg.Port)
	app := defaults.Application()

	if cfg.Username != "" && cfg.Password != "" {
		app.Use(func(ctx *zoox.Context) {
			user, pass, ok := ctx.Request.BasicAuth()
			if !ok {
				ctx.Set("WWW-Authenticate", `Basic realm="go-zoox"`)
				ctx.Status(401)
				return
			}

			if !(user == cfg.Username && pass == cfg.Password) {
				ctx.Status(401)
				return
			}

			ctx.Next()
		})
	}

	app.WebSocket(cfg.Path, func(ctx *zoox.Context, client *websocket.Client) {
		var session session.Session
		var err error
		client.OnDisconnect = func() {
			if session != nil {
				session.Close()
			}
		}

		client.OnTextMessage = func(msg []byte) {
			messageType := msg[0]
			messageData := msg[1:]

			// 2. custom command
			if len(messageData) != 0 {
				// 2.1 resize
				if messageType == '2' {
					var resize Resize
					err := json.Unmarshal(messageData, &resize)
					if err != nil {
						return
					}

					//
					session.Resize(resize.Rows, resize.Columns)
					return
				}
			}

			// 1. user input
			session.Write(msg)
		}

		if cfg.Container == "host" {
			if session, err = container.Host(ctx.Context(), &container.HostConfig{
				Shell: cfg.Shell,
			}); err != nil {
				ctx.Logger.Errorf("[websocket] failed to connect host: %s", err)
				client.Disconnect()
				return
			}
		} else if cfg.Container == "docker" {
			if session, err = container.Docker(ctx.Context()); err != nil {
				ctx.Logger.Errorf("[websocket] failed to connect container: %s", err)
				client.Disconnect()
				return
			}
		} else {
			panic(fmt.Errorf("unknown mode: %s", cfg.Container))
		}

		go func() {
			buf := make([]byte, 128)
			for {
				n, err := session.Read(buf)
				if err != nil {
					logger.Errorf("Failed to read from session: %s", err)
					client.WriteText([]byte(err.Error()))
					return
				}

				client.WriteBinary(buf[:n])
			}
		}()
	})

	app.Get("/", func(ctx *zoox.Context) {
		ctx.HTML(200, RenderXTerm(zoox.H{
			"wsPath": cfg.Path,
			// "welcomeMessage": "custom welcome message",
		}))
	})

	return app.Run(addr)
}

type Resize struct {
	Columns int `json:"cols"`
	Rows    int `json:"rows"`
}
