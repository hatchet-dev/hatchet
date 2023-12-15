package errors

import (
	"context"
)

type Alerter interface {
	SendAlert(ctx context.Context, err error, data map[string]interface{})
}

type NoOpAlerter struct{}

func (s NoOpAlerter) SendAlert(ctx context.Context, err error, data map[string]interface{}) {}
