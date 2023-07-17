package server

import (
	"context"
	"encoding/json"
	"os"
	"os/exec"

	"github.com/creack/pty"
	"github.com/go-zoox/fs"
	"github.com/go-zoox/logger"
	"github.com/go-zoox/zoox/components/application/websocket"
)

func connectHost(ctx context.Context, cfg *Config, client *websocket.Client) (cleanup func(), err error) {
	userShell := cfg.Shell
	userContext := fs.CurrentDir()
	if userShell == "" {
		userShell = "sh"
	}

	shell := exec.Command(userShell)
	shell.Env = append(os.Environ(), "TERM=xterm")
	shell.Dir = userContext

	terminal, err := pty.Start(shell)
	if err != nil {
		return nil, err
	}
	cleanup = func() {
		terminal.Close()
	}

	go func() {
		buf := make([]byte, 128)
		for {
			n, err := terminal.Read(buf)
			if err != nil {
				logger.Errorf("Failed to read from pty master: %s", err)
				client.WriteText([]byte(err.Error()))
				return
			}

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
				setWindowSize(terminal, resize.Columns, resize.Rows)
				return
			}
		}

		// 1. user input
		terminal.Write(msg)
	}

	return
}
