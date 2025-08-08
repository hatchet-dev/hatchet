package internal

import (
	"github.com/hatchet-dev/hatchet/cmd/hatchet-admin/cli/seed"
	"github.com/hatchet-dev/hatchet/pkg/config/loader"
)

func RunSeed(cf *loader.ConfigLoader) error {
	// load the config
	dc, err := cf.InitDataLayer()

	if err != nil {
		panic(err)
	}

	defer dc.Disconnect() // nolint: errcheck

	return seed.SeedDatabase(dc)
}
