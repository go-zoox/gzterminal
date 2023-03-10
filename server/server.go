package server

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"syscall"
	"unsafe"

	"github.com/go-zoox/fs"
	"github.com/go-zoox/logger"
	"github.com/go-zoox/zoox"
	"github.com/go-zoox/zoox/components/context/websocket"
	"github.com/go-zoox/zoox/defaults"

	"github.com/creack/pty"
)

type Server interface {
	Run(cfg *Config) error
}

type Config struct {
	Port        int64
	InitCommand string
	Username    string
	Password    string
}

type server struct {
}

func NewServer() Server {
	return &server{}
}

func Serve(cfg *Config) error {
	s := NewServer()
	return s.Run(cfg)
}

func (s *server) Run(cfg *Config) error {
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

	app.WebSocket("/ws", func(ctx *zoox.Context, client *websocket.WebSocketClient) {
		userShell := os.Getenv("SHELL")
		userContext := fs.CurrentDir()
		if userShell == "" {
			userShell = "sh"
		}

		args := []string{}
		if cfg.InitCommand != "" {
			args = append(args, "-c", cfg.InitCommand)
		}

		shell := exec.Command(userShell, args...)
		shell.Env = append(os.Environ(), "TERM=xterm")
		shell.Dir = userContext

		tty, err := pty.Start(shell)
		if err != nil {
			logger.Errorf("failed to create tty: %v", err)
			client.Disconnect()
			return
		}

		go func() {
			buf := make([]byte, 128)
			for {
				n, err := tty.Read(buf)
				if err != nil {
					logger.Errorf("Failed to read from pty master: %s", err)
					client.WriteText([]byte(err.Error()))
					return
				}

				// client.WriteText(buf[:n])
				client.WriteBinary(buf[:n])
			}
		}()

		client.OnTextMessage = func(msg []byte) {
			messageType := msg[0]
			messageData := msg[1:]
			switch messageType {
			//
			case '2':
				// resize
				var resize Resize
				err := json.Unmarshal(messageData, &resize)
				if err != nil {
					return
				}
				//
				setWindowSize(tty, resize.Columns, resize.Rows)
			default:
				// command
				tty.Write(msg)
			}
		}
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
