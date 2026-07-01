// Deprecated: This package is part of the legacy v0 workflow definition system.
// Use the new Go SDK at github.com/hatchet-dev/hatchet/sdks/go instead. Migration guide: https://docs.hatchet.run/home/migration-guide-go
package client

import (
	"context"
	"sync"

	"github.com/rs/zerolog"
)

type streamCoreConfig[C any] struct {
	constructor   func(context.Context) (C, error)
	closeSend     func(C) error
	replay        func(context.Context, C) error
	initialClient func() (C, bool)
}

type streamListenerRunConfig struct {
	isClosed         func() bool
	retryConnectSync func(context.Context) error
	lifecycleContext func() context.Context
	listen           func(context.Context) error
	l                *zerolog.Logger
	listenErrorMsg   string
}

type streamListenerBase[C any] struct {
	stream     *reconnectingStream[C]
	streamOnce sync.Once
	listenMu   sync.Mutex
	listening  bool
}

type streamListenerStartConfig[L any] struct {
	existing         *L
	build            func() *L
	connect          func(*L, context.Context) error
	start            func(*L) bool
	closeStream      func(*L) error
	stop             func(*L)
	listen           func(*L, context.Context) error
	lifecycleContext func(*L) context.Context
	store            func(*L)
	clear            func(*L)
	l                *zerolog.Logger
	closeErrorMsg    string
	listenErrorMsg   string
}

func (b *streamListenerBase[C]) streamCore(cfg streamCoreConfig[C]) *reconnectingStream[C] {
	b.streamOnce.Do(func() {
		b.stream = newReconnectingStream(cfg.constructor, cfg.closeSend, cfg.replay)

		if cfg.initialClient != nil {
			if client, ok := cfg.initialClient(); ok {
				b.stream.setInitialClient(client)
			}
		}
	})

	return b.stream
}

func (b *streamListenerBase[C]) isListening() bool {
	b.listenMu.Lock()
	defer b.listenMu.Unlock()
	return b.listening
}

func (b *streamListenerBase[C]) startListening(isClosed func() bool) bool {
	b.listenMu.Lock()
	defer b.listenMu.Unlock()

	if b.listening || isClosed() {
		return false
	}

	b.listening = true
	return true
}

func (b *streamListenerBase[C]) stopListening() {
	b.listenMu.Lock()
	b.listening = false
	b.listenMu.Unlock()
}

func (b *streamListenerBase[C]) ensureListening(ctx context.Context, cfg streamListenerRunConfig) error {
	if cfg.isClosed() {
		return errListenerClosed
	}

	if b.isListening() {
		return nil
	}

	if err := cfg.retryConnectSync(ctx); err != nil {
		return err
	}

	if !b.startListening(cfg.isClosed) {
		if cfg.isClosed() {
			return errListenerClosed
		}

		return nil
	}

	go func() {
		defer b.stopListening()

		if err := cfg.listen(cfg.lifecycleContext()); err != nil {
			cfg.l.Error().Err(err).Msg(cfg.listenErrorMsg)
		}
	}()

	return nil
}

func (b *streamListenerBase[C]) Listen(ctx context.Context, cfg streamListenerRunConfig) error {
	if !b.startListening(cfg.isClosed) {
		return nil
	}
	defer b.stopListening()

	return cfg.listen(ctx)
}

func startStreamListener[L any](ctx context.Context, cfg streamListenerStartConfig[L]) (*L, error) {
	if cfg.existing != nil {
		return cfg.existing, nil
	}

	listener := cfg.build()

	if err := cfg.connect(listener, ctx); err != nil {
		return nil, err
	}

	cfg.store(listener)

	if !cfg.start(listener) {
		if closeErr := cfg.closeStream(listener); closeErr != nil {
			cfg.l.Error().Err(closeErr).Msg(cfg.closeErrorMsg)
		}

		cfg.clear(listener)
		return nil, errListenerClosed
	}

	go func() {
		defer cfg.clear(listener)
		defer cfg.stop(listener)

		if err := cfg.listen(listener, cfg.lifecycleContext(listener)); err != nil {
			cfg.l.Error().Err(err).Msg(cfg.listenErrorMsg)
		}
	}()

	return listener, nil
}
