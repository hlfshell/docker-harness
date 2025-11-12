package main

import (
	"database/sql"

	"github.com/go-redis/redis/v8"
)

type FakeRobot struct {
	Name             string
	modelsService    *ModelsService
	messagingService *MessagingService
	db               *sql.DB
	cache            *redis.Client
}

func NewFakeRobot(name string) *FakeRobot {
	return &FakeRobot{
		Name: name,
	}
}

func (f *FakeRobot) Create() error {

}
