package hatchet_client

import (
	v1 "github.com/hatchet-dev/hatchet/pkg/v1"
	"github.com/joho/godotenv"
)

func HatchetClient() (v1.HatchetClient, error) {
	err := godotenv.Load()
	if err != nil {
		return nil, err
	}
	return v1.NewHatchetClient()
}
