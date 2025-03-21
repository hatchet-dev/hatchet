package v1

import (
	v0Client "github.com/hatchet-dev/hatchet/pkg/client"
	v0Config "github.com/hatchet-dev/hatchet/pkg/config/client"
)

type HatchetClient interface {
	V0() v0Client.Client
}

type v1HatchetClientImpl struct {
	v0 *v0Client.Client
}

func NewHatchetClient(config ...Config) (HatchetClient, error) {
	cf := &v0Config.ClientConfigFile{}

	if len(config) > 0 {
		opts := config[0]
		cf = mapConfigToCF(opts)
	}

	client, err := v0Client.NewFromConfigFile(cf)
	if err != nil {
		return nil, err
	}

	return &v1HatchetClientImpl{
		v0: &client,
	}, nil
}

func (c *v1HatchetClientImpl) V0() v0Client.Client {
	return *c.v0
}
