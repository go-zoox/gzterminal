package server

import "io"

type Session interface {
	io.ReadWriteCloser
	//

	Resize(cols, rows int) error
}
