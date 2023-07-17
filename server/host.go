package server

import (
	"context"
	"os"
	"os/exec"
	"syscall"
	"unsafe"

	"github.com/creack/pty"
	"github.com/go-zoox/fs"
)

func connectHost(ctx context.Context, cfg *Config) (session Session, err error) {
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

	return &ResizableHostTerminal{
		File: terminal,
	}, nil
}

type ResizableHostTerminal struct {
	*os.File
}

func (rt *ResizableHostTerminal) Resize(w, h int) error {
	_, _, err := syscall.Syscall(
		syscall.SYS_IOCTL,
		rt.Fd(),
		uintptr(syscall.TIOCSWINSZ),
		uintptr(unsafe.Pointer(&struct{ h, w, x, y uint16 }{uint16(h), uint16(w), 0, 0})),
	)
	return err
}
