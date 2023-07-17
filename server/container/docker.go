package container

import (
	"context"
	"fmt"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	dockerClient "github.com/docker/docker/client"
	"github.com/go-zoox/gzterminal/server/session"
	"github.com/go-zoox/logger"
	"github.com/go-zoox/uuid"
)

func Docker(ctx context.Context) (session session.Session, err error) {
	c, err := dockerClient.NewClientWithOpts(dockerClient.FromEnv)
	if err != nil {
		return nil, err
	}

	res, err := c.ContainerCreate(ctx, &container.Config{
		Image:        "whatwewant/zmicro:v1",
		Cmd:          []string{"/bin/bash"},
		Tty:          true,
		OpenStdin:    true,
		AttachStdin:  true,
		AttachStdout: true,
		AttachStderr: true,
		StdinOnce:    true,
	}, nil, nil, nil, uuid.V4())
	if err != nil {
		return nil, err
	}
	containerID := res.ID

	err = c.ContainerStart(ctx, containerID, types.ContainerStartOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to start container: %v", err)
	}

	stream, err := c.ContainerAttach(ctx, containerID, types.ContainerAttachOptions{
		Stream: true,
		Stdin:  true,
		Stdout: true,
		Stderr: true,
		Logs:   true,
	})
	if err != nil {
		return nil, err
	}

	rct := &ResizableContainerTerminal{
		Ctx:         ctx,
		Client:      c,
		ContainerID: containerID,
		ReadCh:      make(chan []byte),
		Stream:      stream,
	}
	session = rct

	go func() {
		buf := make([]byte, 128)
		for {
			n, err := stream.Reader.Read(buf)
			if err != nil {
				// client.WriteText([]byte(err.Error()))
				return
			}

			// client.WriteBinary(buf[:n])
			rct.ReadCh <- buf[:n]
		}
	}()

	go func() {
		resultC, errC := c.ContainerWait(ctx, containerID, container.WaitConditionNotRunning)
		select {
		case err := <-errC:
			if err != nil {
				logger.Errorf("Failed to wait container: %s", err)
				return
			}

		case result := <-resultC:
			if result.StatusCode != 0 {
				logger.Errorf("Container exited with non-zero status: %d", result.StatusCode)
				return
			}
		}
	}()

	return
}

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
	fmt.Println("Resize", rows, cols)
	return rct.Client.ContainerResize(rct.Ctx, rct.ContainerID, types.ResizeOptions{
		Height: uint(rows),
		Width:  uint(cols),
	})
}
