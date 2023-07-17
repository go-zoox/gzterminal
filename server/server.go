package server

import (
	"fmt"
	"os"
	"syscall"
	"unsafe"

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
	//
	Mode string
}

type server struct {
	cfg *Config
}

func New(cfg *Config) Server {
	if cfg.Mode == "" {
		cfg.Mode = "container"
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

	app.WebSocket("/ws", func(ctx *zoox.Context, client *websocket.Client) {
		var cleanup func()
		var err error
		client.OnDisconnect = func() {
			if cleanup != nil {
				cleanup()
			}
		}

		if cfg.Mode == "host" {
			if cleanup, err = connectHost(ctx.Context(), cfg, client); err != nil {
				ctx.Logger.Errorf("[websocket] failed to connect host: %s", err)
				client.Disconnect()
				return
			}
			return
		}

		if cfg.Mode == "container" {
			if cleanup, err = connectContainer(ctx.Context(), cfg, client); err != nil {
				ctx.Logger.Errorf("[websocket] failed to connect container: %s", err)
				client.Disconnect()
				return
			}
			return
		}

		panic(fmt.Errorf("unknown mode: %s", cfg.Mode))
	})

	app.Get("/", func(ctx *zoox.Context) {
		ctx.HTML(200, RenderXTerm(zoox.H{
			"wsPath": "/ws",
			// "welcomeMessage": "custom welcome message",
		}))
	})

	return app.Run(addr)
}

type Resize struct {
	Columns int `json:"cols"`
	Rows    int `json:"rows"`
}

func setWindowSize(f *os.File, w, h int) {
	syscall.Syscall(
		syscall.SYS_IOCTL,
		f.Fd(),
		uintptr(syscall.TIOCSWINSZ),
		uintptr(unsafe.Pointer(&struct{ h, w, x, y uint16 }{uint16(h), uint16(w), 0, 0})),
	)
}
