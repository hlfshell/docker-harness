package memcached

import (
	"fmt"
	"time"

	"github.com/bradfitz/gomemcache/memcache"

	harness "github.com/hlfshell/docker-harness"
)

type Memcached struct {
	container *harness.Container
	client    *memcache.Client
	port      string
}

func NewMemcached(name string) (*Memcached, error) {
	container, err := harness.NewContainer(
		name,
		"memcached",
		"",
		map[string]string{
			"11211": "",
		},
		map[string]string{},
	)
	if err != nil {
		return nil, err
	}

	return &Memcached{
		container: container,
		port:      "",
		client:    nil,
	}, nil
}

func (m *Memcached) Create() error {
	err := m.container.Start()
	if err != nil {
		return err
	}

	// Grab the assigned port
	ports := m.container.GetPorts()
	m.port = ports["11211"]

	// Ensure that the container is running
	running, err := m.container.IsRunning()
	if err != nil {
		m.container.Cleanup()
		return err
	} else if !running {
		m.container.Cleanup()
		return fmt.Errorf("container failed to start within timeout")
	}

	return nil
}

func (m *Memcached) Connect() (*memcache.Client, error) {
	client := memcache.New(fmt.Sprintf("0.0.0.0:%s", m.port))

	// Ping the
	err := client.Ping()
	if err != nil {
		return nil, err
	}

	m.client = client

	return client, nil
}

func (m *Memcached) ConnectWithTimeout(timeout time.Duration) (*memcache.Client, error) {
	start := time.Now()
	var client *memcache.Client
	var err error
	for time.Since(start) < timeout {
		client, err = m.Connect()
		if err == nil && client != nil {
			break
		} else {
			// Small delay before retrying
			time.Sleep(50 * time.Millisecond)
		}
	}
	if err != nil {
		return nil, err
	} else if client == nil {
		return nil, fmt.Errorf("failed to connect to memcached within timeout")
	}

	return client, nil
}

func (m *Memcached) GetClient() *memcache.Client {
	return m.client
}

func (m *Memcached) Cleanup() error {
	if m.client != nil {
		m.client.Close()
	}
	return m.container.Cleanup()
}
