package dockerharness

import (
	"context"
	"fmt"
	"io"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	docker "github.com/docker/docker/client"
	"github.com/docker/go-connections/nat"
	"github.com/phayes/freeport"
)

type Container struct {
	client  *docker.Client
	id      string
	name    string
	ports   map[string]string
	env     map[string]string
	image   string
	tag     string
	volumes []string

	lock sync.Mutex
}

func NewContainer(name string, image string, tag string, ports map[string]string, env map[string]string) (*Container, error) {
	client, err := docker.NewClientWithOpts(docker.FromEnv)
	if err != nil {
		return nil, err
	}

	if tag == "" {
		tag = "latest"
	}

	return &Container{
		client: client,
		name:   name,
		image:  image,
		tag:    tag,
		ports:  ports,
		env:    env,
	}, nil
}

/*
IsRunning will return true if the container is running, false otherwise
*/
func (c *Container) IsRunning() (bool, error) {
	// If the id was never set, we never launched it
	if c.id == "" {
		return false, nil
	}

	// Check the container status
	container, err := c.client.ContainerInspect(context.Background(), c.id)
	if err != nil && docker.IsErrNotFound(err) {
		return false, nil
	} else if err != nil {
		return false, err
	}

	return container.State.Running, nil
}

/*
ImageExists will return true if the image/tag exists, false otherwise.
A blank tag is assumed to be "latest"
*/
func (c *Container) ImageExists() (bool, error) {
	return ImageExists(c.client, c.image, c.tag)
}

/*
DeleteImage will remove the image/tag from the local machine.
*/
func (c *Container) DeleteImage() error {
	return DeleteImage(c.client, c.image, c.tag)
}

/*
Start will attempt to pull the image and start the container. If there
are assigned port mappings, it will expose and map those ports to the
host machine as specified. If those mappings are not specified, then
at the time of container start we will attempt to assign a free port
at random
*/
func (c *Container) Start() error {
	c.lock.Lock()
	defer c.lock.Unlock()

	// If the container is already running, return
	if running, err := c.IsRunning(); err != nil {
		return err
	} else if running {
		return nil
	}

	// Determine if a container of the same name (but different
	// id) exists. If so, we need to remove it
	if c.name != "" {
		containers, err := c.client.ContainerList(context.Background(), types.ContainerListOptions{
			All: true,
		})
		if err != nil {
			return err
		}
		for _, container := range containers {
			if container.Names[0] == fmt.Sprintf("/%s", c.name) {
				err = CleanupAndKillContainer(c.client, c.name)
				if err != nil {
					return err
				}
				break
			}
		}
	}

	// Attempt to pull the container if we do not have the
	// image locally
	if exists, err := c.ImageExists(); err != nil {
		return err
	} else if !exists {
		if err := c.pullImage(); err != nil {
			return err
		}
	}

	// Convert the env map to a slice of strings
	env := []string{}
	for k, v := range c.env {
		env = append(env, fmt.Sprintf("%s=%s", k, v))
	}

	// Create the port mapping -> exposed ports
	exposedPorts := nat.PortSet{}
	for k := range c.ports {
		// If the port does not contain a protocol, assume tcp
		var port string
		if !strings.Contains(k, "/") {
			port = fmt.Sprintf("%s/tcp", k)
		} else {
			port = k
		}

		exposedPorts[nat.Port(port)] = struct{}{}
	}

	// Determine what ports to assign to the container. If the assigned port is
	// "", then assign any free port. If it is specified, then assign that port
	// to the container.
	portBindings := nat.PortMap{}
	for k, v := range c.ports {
		// If the port does not contain a protocol, assume tcp
		var port string
		if !strings.Contains(k, "/") {
			port = fmt.Sprintf("%s/tcp", k)
		} else {
			port = k
		}

		// If the port is not specified, then assign any free port
		if v == "" {
			freePort, err := freeport.GetFreePort()
			if err != nil {
				return err
			}
			v = fmt.Sprintf("%d", freePort)
			c.ports[k] = v
		}

		// Assign it to our port bindings
		portBindings[nat.Port(port)] = []nat.PortBinding{
			{
				HostIP:   "0.0.0.0",
				HostPort: v,
			},
		}
	}

	// Create our configs
	containerConfig := &container.Config{
		Image:        fmt.Sprintf("%s:%s", c.image, c.tag),
		Env:          env,
		ExposedPorts: exposedPorts,
	}
	hostConfig := &container.HostConfig{
		PortBindings: portBindings,
	}

	response, err := c.client.ContainerCreate(
		context.Background(),
		containerConfig,
		hostConfig,
		nil,
		nil,
		c.name,
	)
	if err != nil {
		return err
	}
	c.id = response.ID

	err = c.client.ContainerStart(context.Background(), response.ID, types.ContainerStartOptions{})
	if err != nil {
		return err
	}

	// Identify volumes attached to our container
	list, err := c.client.ContainerList(context.Background(), types.ContainerListOptions{})
	if err != nil {
		return err
	}
	var targetContainer *types.Container
	for _, container := range list {
		if container.ID == response.ID {
			targetContainer = &container
			break
		}
	}
	if targetContainer == nil {
		return fmt.Errorf("could not find container when it should be created")
	}
	volumes := []string{}
	for _, mount := range targetContainer.Mounts {
		volumes = append(volumes, mount.Name)
	}
	c.volumes = volumes

	return nil
}

/*
Stop will send a SIGTERM signal to the container, and wait up to
`wait` seconds time for the container to stop. If the container
does not stop within that time, a SIGKILL will be sent to the
container.

If a time duration of <= 0 is passed, it is the equivalent of calling
`.Kill()`
*/
func (c *Container) Stop(wait int) error {
	c.lock.Lock()
	defer c.lock.Unlock()

	// If we're not running, we are already stopped
	if running, err := c.IsRunning(); err != nil {
		return err
	} else if !running {
		return nil
	}

	// If the wait is <= 0, then we immediately call SIGKILL
	// instead of waiting
	if wait > 0 {
		// Attempt to stop the container, but abort after a set
		// amount of time
		startedAt := time.Now()
		err := c.client.ContainerStop(context.Background(), c.id, container.StopOptions{
			Timeout: &wait,
			Signal:  "SIGTERM",
		})
		if err != nil {
			// If the time since the stop has passed, then we
			// ignore the err and assume it's a timeout
			duration := time.Since(startedAt)
			if duration < time.Duration(wait)*time.Second {
				return err
			}
		}

		// Determine if the container has stopped. If not, we continue on
		// to call SIGKILL and force the kill. Otherwise, we return
		// successfully
		if running, err := c.IsRunning(); err != nil {
			return err
		} else if !running {
			return nil
		}
	}

	// The timeout has exceeded; let's call SIGKILL and force the
	// container to die
	timeout := -1
	err := c.client.ContainerStop(context.Background(), c.id, container.StopOptions{
		Timeout: &timeout,
		Signal:  "SIGKILL",
	})
	if err != nil {
		return err
	}

	// Confirm that the image is not running anymore
	if running, err := c.IsRunning(); err != nil {
		return err
	} else if !running {
		return nil
	} else {
		return fmt.Errorf("container %s did not stop", c.id)
	}
}

/*
Kill will immediately terminate the container without waiting for
any cleanup or shutdown for applications within the container.
*/
func (c *Container) Kill() error {
	return c.Stop(-1)
}

func (c *Container) Cleanup() error {
	if running, err := c.IsRunning(); err != nil {
		return err
	} else if running {
		err := c.Kill()
		if err != nil {
			return err
		}
	}
	c.lock.Lock()
	defer c.lock.Unlock()

	// Remove the container
	err := c.client.ContainerRemove(context.Background(), c.id, types.ContainerRemoveOptions{})
	if err != nil {
		return err
	}

	// Remove attached volumes
	for _, volume := range c.volumes {
		err := c.client.VolumeRemove(context.Background(), volume, true)
		if err != nil {
			return err
		}
	}

	return nil
}

func (c *Container) GetContainerID() string {
	return c.id
}

func (c *Container) GetPorts() map[string]string {
	return c.ports
}

func (c *Container) pullImage() error {
	out, err := c.client.ImagePull(context.Background(), fmt.Sprintf("%s:%s", c.image, c.tag), types.ImagePullOptions{})
	if err != nil {
		return err
	}
	defer out.Close()

	io.Copy(os.Stdout, out)

	// Check to see if the image was successfully pulled
	if exists, err := c.ImageExists(); err != nil {
		return err
	} else if !exists {
		return fmt.Errorf("image %s:%s did not successfully pull", c.image, c.tag)
	} else {
		return nil
	}
}

func ImageExists(client *docker.Client, image string, tag string) (bool, error) {
	if tag == "" {
		tag = "latest"
	}

	_, _, err := client.ImageInspectWithRaw(context.Background(), fmt.Sprintf("%s:%s", image, tag))
	if err != nil {
		if docker.IsErrNotFound(err) {
			return false, nil
		}
		return false, err
	}

	return true, nil
}

/*
DeleteImage will remove a specific image/tag from your machine
*/
func DeleteImage(client *docker.Client, image string, tag string) error {
	// Chcek if the image exists
	exists, err := ImageExists(client, image, tag)
	if err != nil {
		return err
	} else if !exists {
		return nil
	}

	// Attempt to remove the image
	_, err = client.ImageRemove(context.Background(), fmt.Sprintf("%s:%s", image, tag), types.ImageRemoveOptions{})
	if err != nil {
		return err
	}

	return nil
}

/*
Given a container of a given name, this function will kill and cleanup
all volumes associated with that container.
*/
func CleanupAndKillContainer(client *docker.Client, name string) error {
	// Get the container's ID
	volumes := []string{}
	containers, err := client.ContainerList(context.Background(), types.ContainerListOptions{})
	if err != nil {
		return err
	}
	var containerID string
	for _, container := range containers {
		if container.Names[0] == fmt.Sprintf("/%s", name) {
			containerID = container.ID
			for _, volume := range container.Mounts {
				volumes = append(volumes, volume.Name)
			}
			break
		}
	}
	if containerID == "" {
		return nil
	}

	// Attempt to stop the container
	err = client.ContainerKill(context.Background(), name, "SIGKILL")
	if err != nil {
		return err
	}

	// Remove the container
	err = client.ContainerRemove(context.Background(), name, types.ContainerRemoveOptions{})
	if err != nil {
		return err
	}

	// Remove containers
	for _, volume := range volumes {
		err := client.VolumeRemove(context.Background(), volume, true)
		if err != nil {
			return err
		}
	}

	return nil
}
