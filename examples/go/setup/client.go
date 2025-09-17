package main

import (
	"log"

	hatchet "github.com/hatchet-dev/hatchet/sdks/go"
)

func main() {
	// > Create a Hatchet client
	client, err := hatchet.NewClient()
	if err != nil {
		log.Fatalf("failed to create hatchet client: %v", err)
	}

	_ = client
}
