package redis

import (
	"context"
	"fmt"
	"time"

	harness "github.com/hlfshell/docker-harness"

	"github.com/go-redis/redis/v8"
)

type Redis struct {
	container *harness.Container
	client    *redis.Client
	port      string
	password  string
}

func NewRedis(name string) *Redis {
	container, err := harness.NewContainer(
		name,
		"redis",
		"",
		map[string]string{
			"6379": "",
		},
		map[string]string{},
	)
	if err != nil {
		panic(err)
	}
	return &Redis{
		container: container,
		password:  "",
	}
}

func (r *Redis) Create() error {
	err := r.container.Start()
	if err != nil {
		return err
	}

	// Grab the assigned port
	ports := r.container.GetPorts()
	r.port = ports["6379"]

	// Ensure that the container is running
	start := time.Now()
	timeout := 10 * time.Second
	running := false
	for !running && time.Since(start) < timeout {
		running, err = r.container.IsRunning()
		if err != nil {
			r.container.Cleanup()
			return err
		}
	}
	if !running {
		r.container.Cleanup()
		return fmt.Errorf("container failed to start within timeout")
	}
	return nil
}

func (r *Redis) Connect() (*redis.Client, error) {
	client := redis.NewClient(
		&redis.Options{
			Addr: fmt.Sprintf("0.0.0.0:%s", r.port),
			// Password: r.password,
			DB: 0,
		},
	)

	// Ping the databse to ensure we're connected
	pong, err := client.Ping(context.Background()).Result()
	if err != nil {
		return nil, err
	} else if pong != "PONG" {
		return nil, fmt.Errorf("failed to ping database")
	}

	r.client = client

	return client, nil
}

func (r *Redis) ConnectWithTimeout(timeout time.Duration) (*redis.Client, error) {
	start := time.Now()
	var client *redis.Client
	var err error
	for time.Since(start) < timeout {
		client, err = r.Connect()
		if err == nil && client != nil {
			break
		} else {
			// Introduce a small delay to prevent spamming the database
			time.Sleep(50 * time.Millisecond)
		}
	}
	if err != nil {
		return nil, err
	} else if client == nil {
		return nil, fmt.Errorf("failed to connect to database within timeout")
	}
	return client, nil
}

func (r *Redis) GetClient() *redis.Client {
	return r.client
}

func (r *Redis) Cleanup() error {
	if r.client != nil {
		r.client.Close()
	}
	return r.container.Cleanup()
}
