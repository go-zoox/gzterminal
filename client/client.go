package client

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/url"
	"os"
	"time"

	"github.com/go-zoox/gzterminal/server"
	"github.com/go-zoox/logger"
	"github.com/gorilla/websocket"
	"golang.org/x/term"
)

type Client interface {
	Connect() error
	Resize() error
	Send(key string) error
}

type Config struct {
	Server   string
	Username string
	Password string
	//
	Stdout io.Writer
	Stderr io.Writer
}

type client struct {
	cfg *Config
	//
	conn *websocket.Conn
	//
	stdout io.Writer
	stderr io.Writer
}

func New(cfg *Config) Client {
	stdout := cfg.Stdout
	if stdout == nil {
		stdout = os.Stdout
	}

	stderr := cfg.Stderr
	if stderr == nil {
		stderr = os.Stderr
	}
	return &client{
		cfg: cfg,
		//
		stdout: stdout,
		stderr: stderr,
	}
}

func (c *client) Connect() error {
	u, err := url.Parse(c.cfg.Server)
	if err != nil {
		return fmt.Errorf("invalid caas server address: %s", err)
	}
	logger.Debugf("connecting to %s", u.String())

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	conn, response, err := websocket.DefaultDialer.DialContext(ctx, u.String(), nil)
	if err != nil {
		if response == nil || response.Body == nil {
			cancel()
			return fmt.Errorf("failed to connect at %s (error: %s)", u.String(), err)
		}

		body, errB := ioutil.ReadAll(response.Body)
		if errB != nil {
			cancel()
			return fmt.Errorf("failed to connect at %s (status: %s, error: %s)", u.String(), response.Status, err)
		}

		cancel()
		return fmt.Errorf("failed to connect at %s (status: %d, response: %s, error: %v)", u.String(), response.StatusCode, string(body), err)
	}
	c.conn = conn
	cancel()

	go func() {
		for {
			_, message, err := conn.ReadMessage()
			if err != nil {
				if websocket.IsCloseError(err, websocket.CloseNormalClosure) {
					return
				}

				if websocket.IsCloseError(err, websocket.CloseAbnormalClosure) {
					return
				}

				if websocket.IsCloseError(err, websocket.CloseGoingAway) {
					return
				}

				logger.Debugf("failed to receive command response: %s", err)
				// os.Exit(1)
				return
			}

			c.stdout.Write(message)

			// switch message[0] {
			// case entities.MessageCommandStdout:
			// 	c.stdout.Write(message[1:])
			// case entities.MessageCommandStderr:
			// 	c.stderr.Write(message[1:])
			// case entities.MessageCommandExitCode:
			// 	c.exitCode <- int(message[1])
			// case entities.MessageAuthResponseFailure:
			// 	c.stderr.Write(message[1:])
			// 	// c.exitCode <- 1
			// 	errCh <- &ExitError{
			// 		ExitCode: 1,
			// 		Message:  string(message[1:]),
			// 	}
			// case entities.MessageAuthResponseSuccess:
			// 	errCh <- nil
			// }
		}
	}()

	return nil
}

func (c *client) Resize() error {
	fd := int(os.Stdin.Fd())
	columns, rows, err := term.GetSize(fd)
	if err != nil {
		return err
	}
	resizeData := &server.Resize{
		Columns: columns,
		Rows:    rows,
	}
	data, err := json.Marshal(resizeData)
	if err != nil {
		return err
	}

	return c.conn.WriteMessage(websocket.TextMessage, append([]byte{'2'}, data...))
}

func (c *client) Send(key string) error {
	return c.conn.WriteMessage(websocket.TextMessage, []byte(key))
}
