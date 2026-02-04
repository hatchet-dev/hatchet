package repository

import (
	"context"

	"github.com/hatchet-dev/hatchet/pkg/repository/sqlchelpers"
	"github.com/hatchet-dev/hatchet/pkg/repository/sqlcv1"
)

type OptimisticTx struct {
	tx         sqlcv1.DBTX
	commit     func(ctx context.Context) error
	rollback   func()
	postCommit []func()
}

func (s *sharedRepository) PrepareOptimisticTx(ctx context.Context) (*OptimisticTx, error) {
	tx, commit, rollback, err := sqlchelpers.PrepareTx(ctx, s.pool, s.l)

	if err != nil {
		return nil, err
	}

	return &OptimisticTx{
		tx:         tx,
		commit:     commit,
		rollback:   rollback,
		postCommit: make([]func(), 0),
	}, nil
}

func (o *OptimisticTx) AddPostCommit(f func()) {
	o.postCommit = append(o.postCommit, f)
}

func (o *OptimisticTx) Commit(ctx context.Context) error {
	err := o.commit(ctx)

	if err != nil {
		return err
	}

	for _, f := range o.postCommit {
		doCallback(f)
	}

	return err
}

func (o *OptimisticTx) Rollback() {
	o.rollback()
}

func doCallback(f func()) {
	go func() {
		defer func() {
			recover() // nolint: errcheck
		}()

		f()
	}()
}
