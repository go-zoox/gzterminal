package container

import (
	"context"
	"os"
	"os/exec"
	"syscall"
	"unsafe"

	"github.com/creack/pty"
	"github.com/go-zoox/fs"
	"github.com/go-zoox/gzterminal/server/session"
)

type HostConfig struct {
	Shell string
}

func Host(ctx context.Context, cfg *HostConfig) (session session.Session, err error) {
	userShell := "sh"
	userContext := fs.CurrentDir()
	if cfg.Shell != "" {
		userShell = cfg.Shell
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

func (rt *ResizableHostTerminal) Resize(rows, cols int) error {
	_, _, err := syscall.Syscall(
		syscall.SYS_IOCTL,
		rt.Fd(),
		uintptr(syscall.TIOCSWINSZ),
		uintptr(unsafe.Pointer(&struct{ h, w, x, y uint16 }{uint16(rows), uint16(cols), 0, 0})),
	)
	return err
}
