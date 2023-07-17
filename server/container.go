package server

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	dockerClient "github.com/docker/docker/client"
	"github.com/go-zoox/logger"
	"github.com/go-zoox/uuid"
	"github.com/go-zoox/zoox/components/application/websocket"
)

func connectContainer(ctx context.Context, cfg *Config, client *websocket.Client) (cleanup func(), err error) {
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

	cleanup = func() {
		logger.Infof("close stream")
		stream.Close()
		logger.Infof("stop container")
		c.ContainerRemove(ctx, containerID, types.ContainerRemoveOptions{
			Force: true,
		})
	}

	client.OnTextMessage = func(msg []byte) {
		messageType := msg[0]
		messageData := msg[1:]

		// 2. custom command
		if len(messageData) != 0 {
			// 2.1 resize
			if messageType == '2' {
				var resize Resize
				err := json.Unmarshal(messageData, &resize)
				if err != nil {
					return
				}

				//
				err = c.ContainerResize(ctx, containerID, types.ResizeOptions{
					Height: uint(resize.Rows),
					Width:  uint(resize.Columns),
				})
				if err != nil {
					fmt.Println("resize container error:", err)
					return
				}
				return
			}
		}

		// 1. user input
		_, err = stream.Conn.Write(msg)
		if err != nil {
			logger.Errorf("Failed to write to pty master: %s", err)
		}
	}

	go func() {
		buf := make([]byte, 128)
		for {
			n, err := stream.Reader.Read(buf)
			if err != nil {
				// client.WriteText([]byte(err.Error()))
				return
			}

			client.WriteBinary(buf[:n])
		}
	}()

	go func() {
		resultC, errC := c.ContainerWait(ctx, containerID, container.WaitConditionNotRunning)
		select {
		case err := <-errC:
			if err != nil {
				logger.Errorf("Failed to wait container: %s", err)
				client.Disconnect()
				return
			}

		case result := <-resultC:
			if result.StatusCode != 0 {
				logger.Errorf("Container exited with non-zero status: %d", result.StatusCode)
				client.Disconnect()
				return
			}
		}
	}()

	return
}
