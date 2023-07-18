package docker

import (
	"context"

	"github.com/docker/docker/api/types"
	dockerClient "github.com/docker/docker/client"
	"github.com/go-zoox/logger"
)

type ResizableContainerTerminal struct {
	Ctx         context.Context
	Client      *dockerClient.Client
	ContainerID string
	ReadCh      chan []byte
	Stream      types.HijackedResponse
}

func (rct *ResizableContainerTerminal) Close() error {
	if err := rct.Stream.CloseWrite(); err != nil {
		return err
	}

	return rct.Client.ContainerRemove(rct.Ctx, rct.ContainerID, types.ContainerRemoveOptions{
		Force: true,
	})
}

func (rct *ResizableContainerTerminal) Read(p []byte) (n int, err error) {
	return copy(p, <-rct.ReadCh), nil
}

func (rct *ResizableContainerTerminal) Write(p []byte) (n int, err error) {
	n, err = rct.Stream.Conn.Write(p)
	if err != nil {
		logger.Errorf("Failed to write to pty master: %s", err)
		return 0, err
	}

	return
}

func (rct *ResizableContainerTerminal) Resize(rows, cols int) error {
	return rct.Client.ContainerResize(rct.Ctx, rct.ContainerID, types.ResizeOptions{
		Height: uint(rows),
		Width:  uint(cols),
	})
}
