package repository

import (
	"fmt"
)

var ErrDuplicateKey = fmt.Errorf("duplicate key error")

type WebhookWorkerEngineRepository interface {
}
