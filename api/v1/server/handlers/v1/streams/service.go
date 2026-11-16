package streams

import (
	"github.com/hatchet-dev/hatchet/pkg/config/server"
	"github.com/hatchet-dev/hatchet/pkg/o11yusage"
)

type V1StreamsService struct {
	config *server.ServerConfig
	usage  *o11yusage.Aggregator
}

// NewV1StreamsService meters billable payload bytes on usage; nil meters nothing.
func NewV1StreamsService(config *server.ServerConfig, usage *o11yusage.Aggregator) *V1StreamsService {
	return &V1StreamsService{
		config: config,
		usage:  usage,
	}
}
