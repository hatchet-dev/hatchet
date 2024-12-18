/*
Copyright © 2024 Hatchet Technologies Inc. <support@hatchet.run>
*/
package main

import (
	"log"

	cfg "github.com/hatchet-dev/hatchet/cli/cmd"
)

func main() {

	if err := cfg.Execute(); err != nil {
		log.Fatal(err)
	}
}
