package dockerharness

import (
	"context"
	"fmt"
	"math/rand"
	"testing"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/filters"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestImagePullExistsAndDelete(t *testing.T) {
	// First we attempt to pull a non existent image;
	// this should fail
	containerName := fmt.Sprintf("%s%s", t.Name(), "-non-existent-image")
	rand.Seed(time.Now().UnixNano())
	imageName := fmt.Sprintf("non-existent-image-%d", rand.Int())
	container, err := NewContainer(
		containerName,
		imageName,
		"",
		map[string]string{},
		map[string]string{},
	)
	require.Nil(t, err)
	require.NotNil(t, container)

	err = container.pullImage()
	require.NotNil(t, err)

	// Now we clear out the hello-world image and repull
	// for our test
	containerName = fmt.Sprintf("%s%s", t.Name(), "-successful-image-pull")
	imageName = "hello-world"

	err = DeleteImage(container.client, imageName, "latest")
	require.Nil(t, err)

	container, err = NewContainer(containerName, imageName, "", map[string]string{}, map[string]string{})
	require.Nil(t, err)
	require.NotNil(t, container)

	err = container.pullImage()
	require.Nil(t, err)

	// Ensure the image was pulled
	exists, err := ImageExists(container.client, imageName, "latest")
	require.Nil(t, err)
	assert.True(t, exists)

	// Now ensure that the DeleteImage function works
	// by removing the image and re-checking for it
	err = DeleteImage(container.client, imageName, "latest")
	require.Nil(t, err)

	exists, err = ImageExists(container.client, imageName, "latest")
	require.Nil(t, err)
	assert.False(t, exists)
}

func TestSimpleRunStop(t *testing.T) {
	containerName := t.Name()
	imageName := "postgres"
	tag := "12"

	container, err := NewContainer(
		containerName,
		imageName,
		tag,
		map[string]string{},
		map[string]string{
			"POSTGRES_USER":     "postgres",
			"POSTGRES_PASSWORD": "postgres",
		},
	)
	require.Nil(t, err)
	require.NotNil(t, container)

	err = container.Start()
	require.Nil(t, err)
	defer container.Cleanup()

	// // The image should have been created in this process
	// exists, err = ImageExists(container.client, imageName, tag)
	// require.Nil(t, err)
	// assert.True(t, exists)

	// Is the container running?
	running, err := container.IsRunning()
	require.Nil(t, err)
	assert.True(t, running)

	// Stop the container
	err = container.Stop(20)
	require.Nil(t, err)

	// Is the container running?
	running, err = container.IsRunning()
	require.Nil(t, err)
	assert.False(t, running)

	// Finally, cleanup the container
	err = container.Cleanup()
	require.Nil(t, err)
}

func TestCleanup(t *testing.T) {
	imageName := "postgres"
	tag := "latest"
	containerName := t.Name()

	// Ensure that the container is started and running
	container, err := NewContainer(
		containerName,
		imageName,
		tag,
		map[string]string{},
		map[string]string{
			"POSTGRES_USER":     "postgres",
			"POSTGRES_PASSWORD": "postgres",
		},
	)
	require.Nil(t, err)
	require.NotNil(t, container)

	err = container.Start()
	require.Nil(t, err)
	defer container.Cleanup()

	running, err := container.IsRunning()
	require.Nil(t, err)
	require.True(t, running)

	// Make note of existing resources for the container - the
	// container ID and the attached volumes.
	volumeExistsMap := map[string]bool{}
	for _, volume := range container.volumes {
		volumeExistsMap[volume] = false
	}

	volumes, err := container.client.VolumeList(context.Background(), filters.Args{})
	require.Nil(t, err)
	for _, volume := range volumes.Volumes {
		if _, ok := volumeExistsMap[volume.Name]; ok {
			volumeExistsMap[volume.Name] = true
		}
	}

	// Ensure that all volumes expected to exist exists
	for _, exists := range volumeExistsMap {
		assert.True(t, exists)
	}

	// Reset the map
	for volume := range volumeExistsMap {
		volumeExistsMap[volume] = false
	}

	// Now cleanup the container. We expect it to be stopped,
	// and all volumes to be removed.
	err = container.Cleanup()
	require.Nil(t, err)

	// Ensure that the container is stopped
	running, err = container.IsRunning()
	require.Nil(t, err)
	assert.False(t, running)

	// Ensure that the container is removed
	containers, err := container.client.ContainerList(context.Background(), types.ContainerListOptions{})
	require.Nil(t, err)
	for _, c := range containers {
		assert.NotEqual(t, container.id, c.ID)
	}

	// Ensure that all volumes expected to exist exists
	volumes, err = container.client.VolumeList(context.Background(), filters.Args{})
	require.Nil(t, err)
	for _, volume := range volumes.Volumes {
		if _, ok := volumeExistsMap[volume.Name]; ok {
			volumeExistsMap[volume.Name] = true
		}
	}
	for _, exists := range volumeExistsMap {
		assert.False(t, exists)
	}
}

/*
TestStartOverridesExistingContainer tests a container
being Start'ed when an equivalently named container
exists will destroy/cleanup the old container properly
and then start anew
*/
func TestStartOverridesExistingContainer(t *testing.T) {
	imageName := "postgres"
	tag := "latest"
	containerName := t.Name()

	// Ensure that the container is started and running
	container, err := NewContainer(
		containerName,
		imageName,
		tag,
		map[string]string{},
		map[string]string{
			"POSTGRES_USER":     "postgres",
			"POSTGRES_PASSWORD": "postgres",
		},
	)
	require.Nil(t, err)
	require.NotNil(t, container)

	err = container.Start()
	require.Nil(t, err)
	defer container.Cleanup()

	running, err := container.IsRunning()
	require.Nil(t, err)
	require.True(t, running)

	// Now create a new container with the same name
	// and ensure that it starts properly
	container, err = NewContainer(
		containerName,
		imageName,
		tag,
		map[string]string{},
		map[string]string{
			"POSTGRES_USER":     "postgres",
			"POSTGRES_PASSWORD": "postgres",
		},
	)
	require.Nil(t, err)
	require.NotNil(t, container)
	defer container.Cleanup()

	err = container.Start()
	require.Nil(t, err)

	running, err = container.IsRunning()
	require.Nil(t, err)
	require.True(t, running)

	// Cleanup the container
	err = container.Cleanup()
	require.Nil(t, err)
}

func TestPortMapping(t *testing.T) {
	// Create an example container that maps to
	// a particular port; then we ensure that the
	// port is both exposed and mapped to the expected
	// port. Have one port be specified, the other grab
	// random free port.
	imageName := "postgres"
	tag := "latest"
	containerName := t.Name()

	container, err := NewContainer(
		containerName,
		imageName,
		tag,
		map[string]string{
			"3306": "",
			"3307": "5432",
		},
		map[string]string{
			"POSTGRES_USER":     "postgres",
			"POSTGRES_PASSWORD": "postgres",
		},
	)
	require.Nil(t, err)
	require.NotNil(t, container)

	err = container.Start()
	require.Nil(t, err)
	defer container.Cleanup()

	// Ensure that the container is running
	running, err := container.IsRunning()
	require.Nil(t, err)
	require.True(t, running)
	defer container.Cleanup()

	// Ensure that the container is listening on the
	// expected ports
	ports := container.GetPorts()

	inspect, err := container.client.ContainerInspect(context.Background(), container.id)
	require.Nil(t, err)
	require.NotNil(t, inspect)

	// Ensure that the exposed ports are as expected
	_, ok := inspect.Config.ExposedPorts["3306/tcp"]
	assert.NotNil(t, ok)
	_, ok = inspect.Config.ExposedPorts["3307/tcp"]
	assert.NotNil(t, ok)

	assert.Equal(t, ports["3306"], inspect.NetworkSettings.Ports["3306/tcp"][0].HostPort)
	assert.Equal(t, ports["3307"], inspect.NetworkSettings.Ports["3307/tcp"][0].HostPort)
}
