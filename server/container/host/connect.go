package host

import (
	"context"
	"os"
	"os/exec"

	"github.com/creack/pty"
	"github.com/go-zoox/gzterminal/server/session"
)

func (h *host) Connect(ctx context.Context) (session session.Session, err error) {
	args := []string{}
	if h.cfg.InitCommand != "" {
		args = append(args, "-c", h.cfg.InitCommand)
	}

	cmd := exec.Command(h.cfg.Shell, args...)
	cmd.Env = append(os.Environ(), "TERM=xterm", "HISTFILE=/dev/null")
	cmd.Dir = h.cfg.WorkDir

	terminal, err := pty.Start(cmd)
	if err != nil {
		return nil, err
	}

	return &ResizableHostTerminal{
		File: terminal,
		Cmd:  cmd,
	}, nil
}
