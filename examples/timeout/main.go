package main

import (
	"fmt"

	"github.com/joho/godotenv"
)

type userCreateEvent struct {
	Username string            `json:"username"`
	UserID   string            `json:"user_id"`
	Data     map[string]string `json:"data"`
}

type stepOneOutput struct {
	Message string `json:"message"`
}

func main() {
	err := godotenv.Load()
	if err != nil {
		panic(err)
	}

	events := make(chan string, 50)
	cleanup, err := run(events)
	if err != nil {
		panic(err)
	}

	<-events

	if err := cleanup(); err != nil {
		panic(fmt.Errorf("cleanup() error = %v", err))
	}
}
