package main

import (
	"fmt"

	"github.com/joho/godotenv"

	"github.com/hatchet-dev/hatchet/pkg/cmdutils"
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
	ch := cmdutils.InterruptChan()
	cleanup, err := run(events)
	if err != nil {
		panic(err)
	}

	<-ch

	if err := cleanup(); err != nil {
		panic(fmt.Errorf("cleanup() error = %v", err))
	}
}
