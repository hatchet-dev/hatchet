package queueutils

import "golang.org/x/sync/errgroup"

func MakeBatched[T any](batchSize int, things []T, fn func(group []T) error) error {
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
