# docker-harness

`docker-harness` is a golang package that provides a simple interface for using docker for testing applications. It also provides several useful applications pre-built and ready-to-use for testing as either out-of-the-box utilities or examples to follow.

Currently, the following databases are also provided:
* PostgreSQL
* MySQL

## Example Usage
```golang
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
```

The name is assumed unique for the given container instance - thus if a container already exists with the same name, it will be destroyed when the next instance is created.

## Postgres Example
```golang

container, err := postgres.NewPostgres(
    // Container name - if left blank it'll allow docker to set it
    "TestContainer",
    // Image tag - "" is defaulted to "latest"
    "",
    // Username
    "donatello",
    // Password
    "super-secret",
    // Database
    "database",
)
require.Nil(t, err)

err = container.Create()
require.Nil(t, err)
defer container.Cleanup()

db, err := container.ConnectWithTimeout(10 * time.Second)
require.Nil(t, err)
defer db.Close()
```

## Mysql Example
```golang
container, err := mysql.NewMysql(
    // Container name - if left blank it'll allow docker to set it
    "TestContainer",
    // Image tag - "" is defaulted to "latest"
    "",
    // Username
    "donatello",
    // Password
    "super-secret",
    // Database
    "database",
)
require.nil(t, err)

err = container.Create()
require.Nil(t, err)
defer container.Cleanup()
```