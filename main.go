package main

import (
	"github.com/go-zoox/cli"
	"github.com/go-zoox/gzterminal/server"
)

func main() {
	app := cli.NewSingleProgram(&cli.SingleProgramConfig{
		Name:    "gzssh",
		Usage:   "gzssh is a portable, containered ssh server and client, aliernative to openssh server and client",
		Version: Version,
	})

	app.Command(func(ctx *cli.Context) error {
		return server.Serve(":8080")
	})

	app.Run()
}
