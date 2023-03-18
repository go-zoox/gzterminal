package commands

import (
	"github.com/go-zoox/cli"
	"github.com/go-zoox/gzterminal/client"
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
			Username := ctx.String("username")
			Password := ctx.String("password")

			return client.Run(&client.Config{
				Server:   ctx.String("server"),
				Exec:     ctx.String("exec"),
				Username: Username,
				Password: Password,
			})
		},
	})
}
