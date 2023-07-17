package server

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"syscall"
	"unsafe"

	"github.com/go-zoox/fs"
	"github.com/go-zoox/logger"
	"github.com/go-zoox/zoox"
	"github.com/go-zoox/zoox/components/application/websocket"
	"github.com/go-zoox/zoox/defaults"

	"github.com/creack/pty"
)

type Server interface {
	Run(cfg *Config) error
}

type Config struct {
	Port        int64
	Shell       string
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

	app.WebSocket("/ws", func(ctx *zoox.Context, client *websocket.Client) {
		userShell := cfg.Shell
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
					setWindowSize(tty, resize.Columns, resize.Rows)
					return
				}
			}

			// 1. user input
			tty.Write(msg)
		}
	})

	app.Post("/api/exec", func(ctx *zoox.Context) {
		bytes, err := ctx.BodyBytes()
		if err != nil {
			ctx.Fail(fmt.Errorf("command is illegal format: %s", err), 400001, fmt.Sprintf("command is illegal format: %s", err))
			return
		}

		command := string(bytes)
		if command == "" {
			ctx.Fail(errors.New("command is required"), 400002, "command is required")
			return
		}

		ctx.Logger.Infof("[exec] start to run command: %s", command)

		cmd := exec.Command("sh", "-c", command)
		output, err := cmd.CombinedOutput()
		if err != nil {
			ctx.Logger.Errorf("[exec] failed to run command: %s (output: %s, err: %s)", command, output, err)

			ctx.JSON(400, zoox.H{
				"code":         400003,
				"message":      fmt.Sprintf("failed to run command command: %s (err: %s)", command, err),
				"exit_code":    cmd.ProcessState.ExitCode(),
				"exit_message": string(output),
			})
			return
		}

		// output, err := cmd.CombinedOutput()
		ctx.Logger.Infof("[exec] success to run command: %s", command)
		ctx.String(200, "%s", output)
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
