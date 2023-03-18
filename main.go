package main

import (
	"github.com/go-zoox/cli"
	"github.com/go-zoox/gzterminal/commands"
)

func main() {
	app := cli.NewMultipleProgram(&cli.MultipleProgramConfig{
		Name:    "gzterminal",
		Usage:   "gzterminal is a portable terminal",
		Version: Version,
	})

	// server
	commands.RegistryServer(app)
	// client
	commands.RegistryClient(app)

	app.Run()
}
