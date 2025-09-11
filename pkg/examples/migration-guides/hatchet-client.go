package migration_guides

import (
	v1 "github.com/hatchet-dev/hatchet/pkg/v1"
)

func HatchetClient() (v1.HatchetClient, error) {
	hatchet, err := v1.NewHatchetClient()

	if err != nil {
		return nil, err
	}

	return hatchet, nil
}
