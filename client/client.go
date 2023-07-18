package client

import (
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"net/url"
	"os"
	"time"

	"github.com/go-zoox/gzterminal/message"
	"github.com/go-zoox/logger"
	"github.com/gorilla/websocket"
	"golang.org/x/term"
)

type Client interface {
	Connect() error
	Close() error
	Resize() error
	Send(key string) error
	//
	OnClose() chan error
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
	//
	closeCh chan error
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
		//
		closeCh: make(chan error),
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

	// listen
	go func() {
		for {
			_, message, err := conn.ReadMessage()
			if err != nil {
				// if websocket.IsCloseError(err, websocket.CloseNormalClosure) {
				// 	return
				// }

				// if websocket.IsCloseError(err, websocket.CloseAbnormalClosure) {
				// 	return
				// }

				// if websocket.IsCloseError(err, websocket.CloseGoingAway) {
				// 	return
				// }

				c.closeCh <- err
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

func (c *client) Close() error {
	close(c.closeCh)
	return c.conn.Close()
}

func (c *client) Resize() error {
	fd := int(os.Stdin.Fd())
	columns, rows, err := term.GetSize(fd)
	if err != nil {
		return err
	}

	msg := &message.Message{}
	msg.SetType(message.TypeResize)
	msg.SetResize(&message.Resize{
		Columns: columns,
		Rows:    rows,
	})
	if err := msg.Serialize(); err != nil {
		return err
	}

	return c.conn.WriteMessage(websocket.TextMessage, msg.Msg())
}

func (c *client) Send(key string) error {
	msg := &message.Message{}
	msg.SetType(message.TypeKey)
	msg.SetKey([]byte(key))
	if err := msg.Serialize(); err != nil {
		return err
	}

	return c.conn.WriteMessage(websocket.TextMessage, msg.Msg())
}

func (c *client) OnClose() chan error {
	return c.closeCh
}
