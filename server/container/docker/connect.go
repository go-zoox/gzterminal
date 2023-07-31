package docker

import (
	"context"
	"fmt"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	dockerClient "github.com/docker/docker/client"
	"github.com/go-zoox/gzterminal/server/session"
	"github.com/go-zoox/uuid"
)

func (d *docker) Connect(ctx context.Context) (session session.Session, err error) {
	c, err := dockerClient.NewClientWithOpts(dockerClient.FromEnv)
	if err != nil {
		return nil, err
	}

	res, err := c.ContainerCreate(ctx, &container.Config{
		Image:        d.cfg.Image,
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
		// Logs:   true,
	})
	if err != nil {
		return nil, err
	}

	rct := &ResizableContainerTerminal{
		Ctx:         ctx,
		Client:      c,
		ContainerID: containerID,
		ReadCh:      make(chan []byte),
		Stream:      &stream,
	}
	session = rct

	go func() {
		for {
			buf := make([]byte, 1024)
			n, err := stream.Conn.Read(buf)
			if err != nil {
				return
			}

			rct.ReadCh <- buf[:n]
		}
	}()

	return
}
