package queueutils

import (
	"github.com/hashicorp/go-multierror"
	"golang.org/x/sync/errgroup"
)

func BatchConcurrent[T any](batchSize int, things []T, fn func(group []T) error) error {
	if batchSize <= 0 {
		return nil
	}

	g := new(errgroup.Group)

	for i := 0; i < len(things); i += batchSize {
		end := i + batchSize
		if end > len(things) {
			end = len(things)
		}

		group := things[i:end]

		g.Go(func() error {
			return fn(group)
		})
	}

	return g.Wait()
}

func BatchLinear[T any](batchSize int, things []T, fn func(group []T) error) error {
	if batchSize <= 0 {
		return nil
	}

	var err error

	for i := 0; i < len(things); i += batchSize {
		end := i + batchSize
		if end > len(things) {
			end = len(things)
		}

		group := things[i:end]

		innerErr := fn(group)

		if innerErr != nil {
			err = multierror.Append(err, innerErr)
		}
	}

	return err
}
