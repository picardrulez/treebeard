package main

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"github.com/docker/docker/api/types/container"
	containertypes "github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/image"
	"github.com/docker/docker/api/types/registry"
	"github.com/docker/docker/client"
	"github.com/docker/go-connections/nat"
	"io"
	"net/http"
	"os"
)

func updateHandler(w http.ResponseWriter, r *http.Request) {
	var config = ReadConfig()
	ctx := context.Background()
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		panic(err)
	}
	defer cli.Close()

	username := config.RegistryUser
	password := config.RegistryPassword
	server := config.Registry
	imageName := config.ImageName

	authStr, err := dockerLogin(ctx, cli, username, password, server)
	if err != nil {
		panic(err)
	}
	err = dockerContainerRemoval(ctx, cli)
	if err != nil {
		panic(err)
	}
	err = dockerPullImage(ctx, cli, imageName, authStr)
	if err != nil {
		panic(err)
	}

	runResp, err := dockerRunContainer(ctx, cli, imageName)
	if err != nil {
		panic(err)
	}
	io.WriteString(w, "run response: "+runResp)
	io.WriteString(w, "done with docker stuff")
}

func dockerLogin(ctx context.Context, cli *client.Client, username string, password string, serverAddress string) (string, error) {
	authConfig := registry.AuthConfig{
		Username:      username,
		Password:      password,
		ServerAddress: serverAddress,
	}

	encodedJSON, err := json.Marshal(authConfig)
	if err != nil {
		return "", err
	}

	authStr := base64.URLEncoding.EncodeToString(encodedJSON)

	return authStr, nil
}

func dockerPullImage(ctx context.Context, cli *client.Client, imageName string, authStr string) error {
	out, err := cli.ImagePull(ctx, imageName, image.PullOptions{RegistryAuth: authStr})
	if err != nil {
		return err
	}
	defer out.Close()
	io.Copy(os.Stdout, out)
	return nil
}

func dockerContainerRemoval(ctx context.Context, cli *client.Client) error {
	containers, err := cli.ContainerList(ctx, containertypes.ListOptions{})
	if err != nil {
		return err
	}

	for _, container := range containers {
		fmt.Print("Stopping container ", container.ID[:10], "... ")
		noWaitTimeout := 0
		if err := cli.ContainerStop(ctx, container.ID, containertypes.StopOptions{Timeout: &noWaitTimeout}); err != nil {
			return err
		}
		if err := cli.ContainerRemove(ctx, container.ID, containertypes.RemoveOptions{Force: true}); err != nil {
			return err
		}
	}
	return nil
}

func dockerRunContainer(ctx context.Context, cli *client.Client, imageName string) (string, error) {
	resp, err := cli.ContainerCreate(ctx, &container.Config{
		Image: imageName,
		ExposedPorts: nat.PortSet{
			"80/tcp": struct{}{},
		},
	}, &container.HostConfig{
		PortBindings: map[nat.Port][]nat.PortBinding{
			"80/tcp": {{
				HostIP:   "0.0.0.0",
				HostPort: "80",
			}},
		},
	}, nil, nil, "")
	if err != nil {
		return "", err
	}

	if err := cli.ContainerStart(ctx, resp.ID, containertypes.StartOptions{}); err != nil {
		return "", err
	}

	fmt.Printf("Container started with ID: %s\n", resp.ID)
	containerJSON, err := cli.ContainerInspect(ctx, resp.ID)
	if err != nil {
		return "", err
	}

	fmt.Println("ContainerIP:", containerJSON.NetworkSettings.IPAddress)
	return resp.ID, nil
}
