package commands

import (
	"github.com/go-zoox/cli"
	"github.com/go-zoox/gzterminal/server"
)

func RegistryServer(app *cli.MultipleProgram) {
	app.Register("server", &cli.Command{
		Name:  "server",
		Usage: "terminal server",
		Flags: []cli.Flag{
			&cli.IntFlag{
				Name:    "port",
				Usage:   "server port",
				Aliases: []string{"p"},
				EnvVars: []string{"PORT"},
				Value:   8080,
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

			return server.Serve(&server.Config{
				Port:     ctx.Int64("port"),
				Username: Username,
				Password: Password,
			})
		},
	})
}
