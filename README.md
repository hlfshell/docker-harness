# docker-harness

`docker-harness` is a golang package that provides a simple interface for using docker for testing applications. It also provides several useful applications pre-built and ready-to-use for testing as either out-of-the-box utilities or examples to follow.

## Module Structure

This repository uses a multi-module structure, allowing you to install only what you need:

1. **Core Harness** (`github.com/hlfshell/docker-harness`) - The main framework for launching Docker containers
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
- **Core Harness**: Docker client, container management utilities
- **PostgreSQL**: Core harness + `github.com/lib/pq` (PostgreSQL driver)
- **MySQL**: Core harness + `github.com/go-sql-driver/mysql` (MySQL driver)
- **Redis**: Core harness + `github.com/redis/go-redis/v9` (Redis client)
- **Memcached**: Core harness + `github.com/bradfitz/gomemcache` (Memcached client)

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