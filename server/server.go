package server

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"

	"github.com/go-zoox/logger"
	"github.com/go-zoox/zoox"
	"github.com/go-zoox/zoox/defaults"
)

type Server interface {
	Run(addr string) error
}

type server struct {
}

func NewServer() Server {
	return &server{}
}

func Serve(addr string) error {
	s := NewServer()
	return s.Run(addr)
}

func (s *server) Run(addr string) error {
	app := defaults.Application()

	app.WebSocket("/ws", func(ctx *zoox.Context, client *zoox.WebSocketClient) {
		shell := exec.Command(os.Getenv("SHELL"))

		shell.Env = append(os.Environ(), "TERM=xterm")

		// shell.Stdin = client
		// shell.Stdout = client
		// shell.Stderr = client
		//
		client.OnTextMessage = func(msg []byte) {
			fmt.Println("message:", string(msg))
			messageType := msg[0]
			messageContent := msg[1:]
			switch messageType {
			//
			case '2':
				var resize Resize
				err := json.Unmarshal(messageContent, &resize)
				if err != nil {
					return
				}
				//
			case '1':
			default:
				logger.Errorf("unsupport message type: %s", messageType)
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
