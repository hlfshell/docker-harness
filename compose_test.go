package dockerharness

import (
	"bytes"
	"context"
	"fmt"
	"math/rand/v2"
	"path/filepath"
	"strings"
	"testing"

	networkTypes "github.com/docker/docker/api/types/network"
	volumeTypes "github.com/docker/docker/api/types/volume"
	docker "github.com/docker/docker/client"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestComposeImplementsHarness(t *testing.T) {
	requireCompose(t)

	// Ensure that a compose harness can be used through
	// the common Harness interface.
	harness, err := NewHarnessFromFiles(composeTestName(t), []string{composeFile("full.yml")})
	require.Nil(t, err)
	require.NotNil(t, harness)

	err = harness.Start()
	require.Nil(t, err)
	defer harness.Cleanup()

	running, err := harness.IsRunning()
	require.Nil(t, err)
	assert.True(t, running)

	err = harness.Stop(10)
	require.Nil(t, err)

	running, err = harness.IsRunning()
	require.Nil(t, err)
	assert.False(t, running)
}

func TestComposeStartStopCleanup(t *testing.T) {
	requireCompose(t)

	// Create a compose project with multiple services, a
	// named network, exposed ports, and named volumes.
	compose, err := NewCompose(composeTestName(t), []string{composeFile("full.yml")})
	require.Nil(t, err)
	require.NotNil(t, compose)
	defer compose.Cleanup()

	err = compose.Start()
	require.Nil(t, err)

	// Ensure that at least one container in the project
	// is running.
	running, err := compose.IsRunning()
	require.Nil(t, err)
	assert.True(t, running)

	// Ensure that docker compose reports both services
	// from the compose file.
	services, err := compose.GetServices()
	require.Nil(t, err)
	assert.ElementsMatch(t, []string{"web", "worker"}, services)

	// Ensure that docker compose reports both containers
	// and associates them with this project.
	containers, err := compose.GetContainers()
	require.Nil(t, err)
	require.Len(t, containers, 2)
	for _, container := range containers {
		assert.Equal(t, compose.GetName(), container.Project)
		assert.Contains(t, []string{"web", "worker"}, container.Service)
		assert.Equal(t, "running", strings.ToLower(container.State))
	}

	// Ensure that port mappings can be resolved back to
	// the host machine.
	port, err := compose.GetPort("web", 80, "tcp")
	require.Nil(t, err)
	assert.NotEmpty(t, port)
	assert.Contains(t, port, ":")

	// Ensure that service logs can be read.
	logs, err := compose.GetLogs("worker")
	require.Nil(t, err)
	assert.Contains(t, logs, "worker-ready")

	// Ensure that the compose network and volumes exist.
	client, err := docker.NewClientWithOpts(docker.FromEnv)
	require.Nil(t, err)
	defer client.Close()

	_, err = client.NetworkInspect(context.Background(), fmt.Sprintf("%s_app", compose.GetName()), networkTypes.InspectOptions{})
	require.Nil(t, err)

	volumes, err := client.VolumeList(context.Background(), volumeTypes.ListOptions{})
	require.Nil(t, err)
	assertComposeVolumeExists(t, volumes.Volumes, fmt.Sprintf("%s_web-cache", compose.GetName()))
	assertComposeVolumeExists(t, volumes.Volumes, fmt.Sprintf("%s_worker-data", compose.GetName()))

	// Stop the project and ensure the containers are no
	// longer running.
	err = compose.Stop(10)
	require.Nil(t, err)

	running, err = compose.IsRunning()
	require.Nil(t, err)
	assert.False(t, running)

	// Cleanup the project and ensure the compose-owned
	// resources are removed.
	err = compose.Cleanup()
	require.Nil(t, err)

	_, err = client.NetworkInspect(context.Background(), fmt.Sprintf("%s_app", compose.GetName()), networkTypes.InspectOptions{})
	assert.True(t, docker.IsErrNotFound(err), "expected compose network to be removed")

	_, err = client.VolumeInspect(context.Background(), fmt.Sprintf("%s_web-cache", compose.GetName()))
	assert.True(t, docker.IsErrNotFound(err), "expected compose volume to be removed")
}

func TestComposeCleanupCanKeepVolumes(t *testing.T) {
	requireCompose(t)

	compose, err := NewComposeWithOptions(ComposeOptions{
		Name:        composeTestName(t),
		Files:       []string{composeFile("keep-volumes.yml")},
		KeepVolumes: true,
	})
	require.Nil(t, err)
	require.NotNil(t, compose)
	defer compose.Cleanup()

	err = compose.Start()
	require.Nil(t, err)

	client, err := docker.NewClientWithOpts(docker.FromEnv)
	require.Nil(t, err)
	defer client.Close()

	volumeName := fmt.Sprintf("%s_testdata", compose.GetName())
	defer client.VolumeRemove(context.Background(), volumeName, true)

	// Ensure that the volume exists while the compose
	// project is running.
	_, err = client.VolumeInspect(context.Background(), volumeName)
	require.Nil(t, err)

	// Cleanup should remove the containers and network,
	// but leave named volumes behind when requested.
	err = compose.Cleanup()
	require.Nil(t, err)

	_, err = client.VolumeInspect(context.Background(), volumeName)
	assert.Nil(t, err)
}

func TestComposeOptions(t *testing.T) {
	requireCompose(t)

	stdout := &bytes.Buffer{}
	compose, err := NewComposeWithOptions(ComposeOptions{
		Name:    composeTestName(t),
		Files:   []string{composeFile("env.yml")},
		Env:     map[string]string{"HARNESS_VALUE": "expected"},
		NoWait:  true,
		Stdout:  stdout,
		Stderr:  &bytes.Buffer{},
		WorkDir: ".",
	})
	require.Nil(t, err)
	require.NotNil(t, compose)
	defer compose.Cleanup()

	err = compose.Start()
	require.Nil(t, err)

	// Ensure that captured helpers still work when stdout
	// is assigned for start/stop output.
	services, err := compose.GetServices()
	require.Nil(t, err)
	assert.Equal(t, []string{"envcheck"}, services)
}

func requireCompose(t *testing.T) {
	t.Helper()

	if _, err := detectComposeCommand(); err != nil {
		t.Skip("docker compose is not available")
	}
}

func composeFile(name string) string {
	return filepath.Join("testdata", "compose", name)
}

func composeTestName(t *testing.T) string {
	name := strings.ToLower(t.Name())
	name = strings.ReplaceAll(name, "_", "-")
	name = strings.ReplaceAll(name, "/", "-")
	return fmt.Sprintf("%s-%d", name, rand.IntN(1000000))
}

func assertComposeVolumeExists(t *testing.T, volumes []*volumeTypes.Volume, name string) {
	t.Helper()

	for _, volume := range volumes {
		if volume.Name == name {
			return
		}
	}

	assert.Failf(t, "expected compose volume to exist", "volume %s was not found", name)
}
