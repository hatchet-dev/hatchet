package main

import (
	"github.com/hatchet-dev/hatchet/api/v1/server/run"
	"github.com/hatchet-dev/hatchet/internal/config/loader"
)

func main() {
	// init the repository
	cf := &loader.ConfigLoader{}

	sc, err := cf.LoadServerConfig()

	if err != nil {
		panic(err)
	}

	runner := run.NewAPIServer(sc)

	err = runner.Run()

	if err != nil {
		panic(err)
	}
}
