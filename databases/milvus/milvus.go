package milvus

import (
	"context"
	"fmt"
	"time"

	harness "github.com/hlfshell/docker-harness"

	milvus "github.com/milvus-io/milvus-sdk-go/v2/client"
)

const MILVUS_IMAGE = "milvusdb/milvus"
const MILVUS_PORT = "19530"

type Milvus struct {
	container *harness.Container
	port      string
	client    *milvus.GrpcClient
}

func NewMilvus(name string, tag string) (*Milvus, error) {
	container, err := harness.NewContainer(
		name,
		"milvusdb/milvus",
		tag,
		map[string]string{
			MILVUS_PORT: "",
		},
		map[string]string{},
	)
	if err != nil {
		return nil, err
	}

	return &Milvus{
		container: container,
	}, nil
}

func (m *Milvus) Create() error {
	err := m.container.Start()
	if err != nil {
		return err
	}

	// Grab the assigned port
	ports := m.container.GetPorts()
	m.port = ports[MILVUS_PORT]

	// Ensure that the container is running
	running, err := m.container.IsRunning()
	if err != nil {
		return err
	} else if !running {
		return fmt.Errorf("container failed to start within timeout")
	}

	return nil
}

func (m *Milvus) Connect() (*milvus.GrpcClient, error) {
	clientInterface, err := milvus.NewGrpcClient(
		context.Background(),
		fmt.Sprintf("0.0.0.0:%s", m.port),
	)
	client, ok := clientInterface.(*milvus.GrpcClient)
	if !ok {
		return nil, fmt.Errorf("failed to cast client to grpc client")
	}

	m.client = client

	return client, err
}

func (m *Milvus) ConnectWithTimeout(timeout time.Duration) (*milvus.GrpcClient, error) {
	start := time.Now()
	var client *milvus.GrpcClient
	var err error
	for time.Since(start) < timeout {
		client, err = m.Connect()
		if err == nil && client != nil {
			break
		} else {
			// Small delay to prevent overwhelming
			// the database w/ connection attempts
			time.Sleep(50 * time.Millisecond)
		}
	}

	return client, err
}

func (m *Milvus) GetClient() *milvus.GrpcClient {
	return m.client
}

func (m *Milvus) GetContainer() *harness.Container {
	return m.container
}

func (m *Milvus) Cleanup() error {
	if m.client != nil {
		m.client.Close()
	}
	return m.container.Cleanup()
}
