# docker-harness

`docker-harness` is a golang package that provides a simple interface for using docker for testing applications. It can start either a single container or a Docker Compose project, and it also provides several useful applications pre-built and ready-to-use for testing as either out-of-the-box utilities or examples to follow.

## Module Structure

This repository uses a multi-module structure, allowing you to install only what you need:

1. **Core Harness** (`github.com/hlfshell/docker-harness`) - The main framework for launching Docker containers and Docker Compose projects
2. **Individual Database Modules** - Each database is its own installable module:
   - PostgreSQL: `github.com/hlfshell/docker-harness/databases/postgres`
   - MySQL: `github.com/hlfshell/docker-harness/databases/mysql`
   - Redis: `github.com/hlfshell/docker-harness/databases/redis`
   - Memcached: `github.com/hlfshell/docker-harness/databases/memcached`

### Installation

**For just the core harness:**
```bash
go get github.com/hlfshell/docker-harness
```

**For specific databases (each installs only its own dependencies):**
```bash
# PostgreSQL only
go get github.com/hlfshell/docker-harness/databases/postgres

# MySQL only
go get github.com/hlfshell/docker-harness/databases/mysql

# Redis only
go get github.com/hlfshell/docker-harness/databases/redis

# Memcached only
go get github.com/hlfshell/docker-harness/databases/memcached
```

This granular separation ensures you only pull the dependencies for the databases you actually use. Each database module automatically includes the core harness as a dependency.

**What each module includes:**
- **Core Harness**: Docker client, container management utilities, Docker Compose runner
- **PostgreSQL**: Core harness + `github.com/lib/pq` (PostgreSQL driver)
- **MySQL**: Core harness + `github.com/go-sql-driver/mysql` (MySQL driver)
- **Redis**: Core harness + `github.com/redis/go-redis/v9` (Redis client)
- **Memcached**: Core harness + `github.com/bradfitz/gomemcache` (Memcached client)

## Development

This project uses [Just](https://just.systems/) as a command runner for common development tasks.

### Available Commands

- 🧪 `just test` - Run all tests (core + all databases)
- 📊 `just coverage` - Generate test coverage report
- 🧹 `just clean` - Clean test cache and artifacts
- 📦 `just build` - Build all modules
- 🎨 `just format` - Format all Go code
- 🔍 `just vet` - Run `go vet` on all modules
- 📝 `just version` - Show the current version tag
- 🚀 `just release <version>` - Tag and push a new version (e.g., `just release 1.2.3`)

### Quick Start

```bash
# Run all tests
just test

# Format code
just format

# Release a new version
just release 1.2.3
```

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

## Docker Compose Example

`docker-harness` can also run a Docker Compose project for integration tests that need multiple services. Compose support uses Docker Compose v2 (`docker compose`) when available and falls back to `docker-compose`. Compose uses the same harness methods as a single container: `Start`, `Stop`, `Cleanup`, and `IsRunning`.

```golang
package main

import (
	"fmt"

	harness "github.com/hlfshell/docker-harness"
)

func main() {
	compose, err := harness.NewCompose(
		// Compose project name - if left blank, docker-harness generates one
		"my-test-stack",
		// Docker Compose files
		[]string{"compose.yml"},
	)
	if err != nil {
		panic(err)
	}

	if err := compose.Start(); err != nil {
		panic(err)
	}
	defer compose.Cleanup()

	running, err := compose.IsRunning()
	if err != nil {
		panic(err)
	}
	fmt.Println("compose project running:", running)

	addr, err := compose.GetPort("web", 80, "tcp")
	if err != nil {
		panic(err)
	}
	fmt.Println("web service is listening at:", addr)

	services, err := compose.GetServices()
	if err != nil {
		panic(err)
	}
	fmt.Println("compose services:", services)
}
```

The name is used as the Docker Compose project name. This isolates containers, networks, and volumes so cleanup removes only resources for that compose project. If the name is left blank, `docker-harness` generates a unique project name.

If you need additional Docker Compose options, use `NewComposeWithOptions`:

```golang
compose, err := harness.NewComposeWithOptions(harness.ComposeOptions{
	Name:        "my-test-stack",
	Files:       []string{"compose.yml"},
	Profiles:    []string{"integration"},
	Services:    []string{"web", "worker"},
	Env:         map[string]string{"APP_ENV": "test"},
	KeepVolumes: true,
})
```

By default, `Start` waits for services to be running or healthy, and `Cleanup` runs `docker compose down --remove-orphans --volumes`. Set `NoWait: true` to return immediately from `Start`, `KeepOrphans: true` to leave orphaned containers alone, or `KeepVolumes: true` to preserve Compose-created volumes after cleanup.

Compose harnesses also expose helpers for the docker compose information you typically need in tests:

- `GetServices()` returns service names in the project.
- `GetContainers()` returns the compose containers with their project, service, state, and health information.
- `GetPort(service, privatePort, protocol)` returns the host address for a published service port.
- `GetLogs(services...)` returns compose logs for the whole project or for specific services.
- `GetName()` returns the compose project name, and `GetFiles()` returns the compose files used to create the harness.

## Postgres Example
```golang
package main

import (
	"database/sql"
	"time"

	"github.com/hlfshell/docker-harness/databases/postgres"
)

func main() {
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
	if err != nil {
		panic(err)
	}

	err = container.Create()
	if err != nil {
		panic(err)
	}
	defer container.Cleanup()

	db, err := container.ConnectWithTimeout(10 * time.Second)
	if err != nil {
		panic(err)
	}
	defer db.Close()
	
	// Use db (*sql.DB) here...
	_ = db
}
```

## Mysql Example
```golang
package main

import (
	"database/sql"
	"time"

	"github.com/hlfshell/docker-harness/databases/mysql"
)

func main() {
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
	if err != nil {
		panic(err)
	}

	err = container.Create()
	if err != nil {
		panic(err)
	}
	defer container.Cleanup()

	db, err := container.ConnectWithTimeout(10 * time.Second)
	if err != nil {
		panic(err)
	}
	defer db.Close()
	
	// Use db (*sql.DB) here...
	_ = db
}
```

## Redis Example

```golang
package main

import (
	"context"
	"time"

	"github.com/hlfshell/docker-harness/databases/redis"
	"github.com/redis/go-redis/v9"
)

func main() {
	r, err := redis.NewRedis("TestContainer")
	if err != nil {
		panic(err)
	}

	err = r.Create()
	if err != nil {
		panic(err)
	}
	defer r.Cleanup()

	client, err := r.ConnectWithTimeout(10 * time.Second)
	if err != nil {
		panic(err)
	}
	defer client.Close()
	
	// Use client (*redis.Client) here...
	ctx := context.Background()
	_ = client.Ping(ctx)
}
```

## Memcached Example

```golang
package main

import (
	"time"

	"github.com/bradfitz/gomemcache/memcache"
	"github.com/hlfshell/docker-harness/databases/memcached"
)

func main() {
	m, err := memcached.NewMemcached("TestContainer")
	if err != nil {
		panic(err)
	}

	// Create the container
	err = m.Create()
	if err != nil {
		panic(err)
	}
	defer m.Cleanup()

	// Connect to memcached
	client, err := m.ConnectWithTimeout(10 * time.Second)
	if err != nil {
		panic(err)
	}
	
	// Use client (*memcache.Client) here...
	_ = client.Set(&memcache.Item{
		Key:   "test",
		Value: []byte("value"),
	})
}
```
