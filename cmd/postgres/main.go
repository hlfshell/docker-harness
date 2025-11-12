package main

import (
	"fmt"
	"time"

	"github.com/hlfshell/docker-harness/databases/postgres"
)

func main() {
	// The following will create a new postgres container that maps port 3306 to
	// any open port on the host machine. Since the tag is not specified, it will
	// default to "latest".
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

	// You can now use the db connection; for example, create a table:
	_, err = db.Exec("CREATE TABLE IF NOT EXISTS test (id SERIAL PRIMARY KEY, name TEXT)")
	if err != nil {
		panic(err)
	}

	// Insert a row
	_, err = db.Exec("INSERT INTO test (name) VALUES ('test')")
	if err != nil {
		panic(err)
	}

	// ...and read it back
	rows, err := db.Query("SELECT * FROM test")
	if err != nil {
		panic(err)
	}
	defer rows.Close()

	count := 0
	for rows.Next() {
		count++
	}
	fmt.Println("Rows in table:", count)

	// The database should clean itself up in the end
}
