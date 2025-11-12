package main

import (
	"fmt"
	"time"

	harness "github.com/hlfshell/docker-harness"
)

func main() {
	// The following will create a new postgres container that maps port 3306 to
	// any open port on the host machine. Since the tag is not specified, it will
	// default to "latest".
	container, err := harness.NewContainer(
		// Container name - if left blank it'll allow docker to set it
		"TestContainer",
		// Image name
		"postgres",
		// Image tag - "" is defaulted to "latest"
		"",
		// Port mapping/exposing
		map[string]string{
			"3306": "",
		},
		// Env vars
		map[string]string{
			"POSTGRES_USER":     "postgres",
			"POSTGRES_PASSWORD": "postgres",
			"POSTGRES_DB":       "postgres",
		},
	)
	if err != nil {
		panic(err)
	}

	err = container.Start()
	if err != nil {
		panic(err)
	}
	defer container.Cleanup()

	running, err := container.IsRunning()
	if err != nil {
		panic(err)
	} else if !running {
		fmt.Println("Container did not start properly")
	} else {
		fmt.Println("Container is running!")
	}

	fmt.Println("Container is listening on ports:", container.GetPorts())

	time.Sleep(3 * time.Second)
}
