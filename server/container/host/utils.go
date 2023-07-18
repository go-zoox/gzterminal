package host

import (
	"os"

	"github.com/creack/pty"
)

type ResizableHostTerminal struct {
	*os.File
}

func (rt *ResizableHostTerminal) Resize(rows, cols int) error {
	return pty.Setsize(rt.File, &pty.Winsize{
		Rows: uint16(rows),
		Cols: uint16(cols),
	})

	// _, _, err := syscall.Syscall(
	// 	syscall.SYS_IOCTL,
	// 	rt.Fd(),
	// 	uintptr(syscall.TIOCSWINSZ),
	// 	uintptr(unsafe.Pointer(&struct{ h, w, x, y uint16 }{uint16(rows), uint16(cols), 0, 0})),
	// )
	// return err
}
