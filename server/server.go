package server

import (
	"fmt"

	"github.com/go-zoox/gzterminal/message"
	"github.com/go-zoox/gzterminal/server/container/docker"
	"github.com/go-zoox/gzterminal/server/container/host"
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

		client.OnTextMessage = func(rawMsg []byte) {
			msg, err := message.Deserialize(rawMsg)
			if err != nil {
				logger.Errorf("Failed to deserialize message: %s", err)
				return
			}

			switch msg.Type() {
			case message.TypeKey:
				session.Write(msg.Key())
			case message.TypeResize:
				resize := msg.Resize()
				err = session.Resize(resize.Rows, resize.Columns)
				if err != nil {
					logger.Errorf("Failed to resize terminal: %s", err)
				}
			default:
				logger.Errorf("Unknown message type: %d", msg.Type())
			}
		}

		if cfg.Container == "host" {
			if session, err = host.New(&host.Config{
				Shell: cfg.Shell,
			}).Connect(ctx.Context()); err != nil {
				ctx.Logger.Errorf("[websocket] failed to connect host: %s", err)
				client.Disconnect()
				return
			}
		} else if cfg.Container == "docker" {
			if session, err = docker.New(&docker.Config{
				Shell: cfg.Shell,
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
