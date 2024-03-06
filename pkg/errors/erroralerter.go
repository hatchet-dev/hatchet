package errors

import (
	"context"

	"github.com/hatchet-dev/hatchet/internal/datautils/merge"
)

type Alerter interface {
	SendAlert(ctx context.Context, err error, data map[string]interface{})
}

type NoOpAlerter struct{}

func (s NoOpAlerter) SendAlert(ctx context.Context, err error, data map[string]interface{}) {}

type Wrapped struct {
	a    Alerter
	data map[string]interface{}
}

func NewWrapped(a Alerter) *Wrapped {
	return &Wrapped{
		a: a,
	}
}

func (w *Wrapped) WithData(data map[string]interface{}) {
	w.data = data
}

func (w *Wrapped) WrapErr(err error, data map[string]interface{}) error {
	if err == nil {
		return nil
	}

	w.a.SendAlert(context.Background(), err, merge.MergeMaps(w.data, data))
	return err
}
