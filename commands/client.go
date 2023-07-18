package commands

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/eiannone/keyboard"
	"github.com/go-zoox/cli"
	"github.com/go-zoox/gzterminal/client"
	"github.com/go-zoox/logger"
)

func RegistryClient(app *cli.MultipleProgram) {
	app.Register("client", &cli.Command{
		Name:  "client",
		Usage: "terminal client",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:     "server",
				Usage:    "server url",
				Aliases:  []string{"s"},
				Required: true,
			},
			&cli.StringFlag{
				Name:    "exec",
				Usage:   "specify exec command",
				Aliases: []string{"e"},
				EnvVars: []string{"EXEC"},
			},
			&cli.StringFlag{
				Name:    "username",
				Usage:   "Username for Basic Auth",
				EnvVars: []string{"USERNAME"},
			},
			&cli.StringFlag{
				Name:    "password",
				Usage:   "Password for Basic Auth",
				EnvVars: []string{"PASSWORD"},
			},
		},
		Action: func(ctx *cli.Context) (err error) {
			c := client.New(&client.Config{
				Server:   ctx.String("server"),
				Username: ctx.String("username"),
				Password: ctx.String("password"),
			})

			if err := c.Connect(); err != nil {
				return err
			}
			go func() {
				err := <-c.OnClose()
				if err != nil {
					logger.Errorf("server disconnect by %v", err)
				} else {
					logger.Errorf("server disconnect")
				}
				os.Exit(1)
			}()

			// resize
			if err := c.Resize(); err != nil {
				return err
			}

			// 监听操作系统信号
			sigWinch := make(chan os.Signal, 1)
			signal.Notify(sigWinch, syscall.SIGWINCH)
			// 启动循环来检测终端窗口大小是否发生变化
			go func() {
				for {
					select {
					case <-sigWinch:
						c.Resize()
					default:
						time.Sleep(time.Millisecond * 100)
					}
				}
			}()

			if err := keyboard.Open(); err != nil {
				return err
			}
			defer func() {
				_ = keyboard.Close()
			}()

			for {
				char, key, err := keyboard.GetKey()
				if err != nil {
					return err
				}

				// fmt.Printf("You pressed: rune:%q, key %X\r\n", char, key)
				if key == keyboard.KeyCtrlC {
					break
				}
				if err != nil {
					fmt.Fprintln(os.Stderr, err)
				}

				if key == 0 {
					err = c.Send(string(char))
					if err != nil {
						fmt.Fprintln(os.Stderr, err)
					}
				} else {
					// if key == keyboard.KeyBackspace2 {
					// 	err = c.Send("\x7F")
					// 	if err != nil {
					// 		fmt.Fprintln(os.Stderr, err)
					// 	}
					// }
					err = c.Send(string([]byte{byte(key)}))
					if err != nil {
						fmt.Fprintln(os.Stderr, err)
					}
				}

			}

			return
		},
	})
}
