package pgvector

import (
	"database/sql"

	harness "github.com/hlfshell/docker-harness"
)

const PGVECTOR_IMAGE = "ankane/pgvector"

type PGVector struct {
	container *harness.Container
	db        *sql.DB
	port      string
}

func NewPGVector(name string, tag string, username string, password string, database string) (*PGVector, error) {
	container, err := harness.NewContainer(
		name,
		PGVECTOR_IMAGE,
		tag,
		map[string]string{
			"5432": "",
		},
		map[string]string{
			"POSTGRES_USER":     username,
			"POSTGRES_PASSWORD": password,
			"POSTGRES_DB":       database,
		},
	)
	if err != nil {
		return nil, err
	}

	return &PGVector{
		container: container,
		db:        nil,
		port:      "",
	}, nil
}
