package loader

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/hatchet-dev/hatchet/pkg/config/server"
)

func TestValidateDeliveryIntervals(t *testing.T) {
	valid := func() server.ConfigFileRuntime {
		rt := server.ConfigFileRuntime{
			InternalEventOutboxPollInterval: time.Second,
			DispatcherPullInterval:          2 * time.Second,
			DispatcherPullGrace:             2 * time.Second,
		}
		rt.TaskOperationLimits.ReassignUnackedTasks = true

		return rt
	}

	for name, tc := range map[string]struct {
		mutate  func(*server.ConfigFileRuntime)
		wantErr bool
	}{
		"defaults":             {mutate: func(*server.ConfigFileRuntime) {}},
		"zero grace":           {mutate: func(rt *server.ConfigFileRuntime) { rt.DispatcherPullGrace = 0 }},
		"zero outbox poll":     {mutate: func(rt *server.ConfigFileRuntime) { rt.InternalEventOutboxPollInterval = 0 }, wantErr: true},
		"negative outbox poll": {mutate: func(rt *server.ConfigFileRuntime) { rt.InternalEventOutboxPollInterval = -time.Second }, wantErr: true},
		"busy outbox poll":     {mutate: func(rt *server.ConfigFileRuntime) { rt.InternalEventOutboxPollInterval = time.Nanosecond }, wantErr: true},
		"zero pull":            {mutate: func(rt *server.ConfigFileRuntime) { rt.DispatcherPullInterval = 0 }, wantErr: true},
		"busy pull":            {mutate: func(rt *server.ConfigFileRuntime) { rt.DispatcherPullInterval = time.Nanosecond }, wantErr: true},
		"negative grace":       {mutate: func(rt *server.ConfigFileRuntime) { rt.DispatcherPullGrace = -time.Second }, wantErr: true},
		"pull past reassign":   {mutate: func(rt *server.ConfigFileRuntime) { rt.DispatcherPullGrace = time.Minute }, wantErr: true},
		"long grace without reassign": {mutate: func(rt *server.ConfigFileRuntime) {
			rt.DispatcherPullGrace = time.Minute
			rt.TaskOperationLimits.ReassignUnackedTasks = false
		}},
	} {
		t.Run(name, func(t *testing.T) {
			rt := valid()
			tc.mutate(&rt)

			err := validateDeliveryIntervals(&rt)

			if tc.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
