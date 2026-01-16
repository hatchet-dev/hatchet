package client

import (
	"errors"
	"os"

	hatchet "github.com/hatchet-dev/hatchet/sdks/go"
	"github.com/joho/godotenv"
)

func HatchetClient() (*hatchet.Client, error) {
	if _, err := os.Stat(".env"); os.IsExist(err) {
		err := godotenv.Load()
		if err != nil {
			return nil, err
		}
	}

	// check for HATCHET_CLIENT_TOKEN
	token := os.Getenv("HATCHET_CLIENT_TOKEN")
	if token == "" {
		return nil, errors.New("HATCHET_CLIENT_TOKEN is not set")
	}

	return hatchet.NewClient()
}
