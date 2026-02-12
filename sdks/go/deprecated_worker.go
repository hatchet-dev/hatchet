package hatchet

// Legacy dual-worker creation for pre-slot-config engines.
// When connected to an older Hatchet engine that does not support multiple slot types,
// this module provides the old NewWorker flow which creates separate durable and
// non-durable workers, each registered with the legacy `slots` proto field.

import (
	"context"
	"fmt"
	"time"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/hatchet-dev/hatchet/pkg/worker"
)

// legacyEngineDeprecationStart is the date when slot_config support was released.
var legacyEngineDeprecationStart = time.Date(2026, 2, 12, 0, 0, 0, 0, time.UTC)

const legacyEngineMessage = "Connected to an older Hatchet engine that does not support multiple slot types. " +
	"Falling back to legacy worker registration. " +
	"Please upgrade your Hatchet engine to the latest version."

// isLegacyEngine checks whether the engine supports the new slot_config registration.
// Returns true if the engine is legacy (does not implement GetVersion).
// Returns an error if the deprecation grace period has expired and the random
// check fires (phase 3).
func (c *Client) isLegacyEngine() (bool, error) {
	ctx := context.Background()
	_, err := c.legacyClient.Dispatcher().GetVersion(ctx)
	if err != nil {
		if status.Code(err) == codes.Unimplemented {
			l := c.legacyClient.Logger()
			if depErr := EmitDeprecationNotice("legacy-engine", legacyEngineMessage, legacyEngineDeprecationStart, l, &DeprecationOpts{
				ErrorWindow: 180 * 24 * time.Hour,
			}); depErr != nil {
				return false, fmt.Errorf("legacy engine deprecated: %w", depErr)
			}
			return true, nil
		}
		// For other errors (e.g., connectivity), assume new engine and let registration fail naturally
		return false, nil
	}
	return false, nil
}

// newLegacyWorker creates workers using the old dual-worker pattern for pre-slot-config engines.
// Uses WithLegacySlots so that registration sends the deprecated `slots` proto field
// instead of `slot_config`.
func newLegacyWorker(c *Client, name string, config *workerConfig, dumps []workflowDump) (*Worker, error) {
	workerOpts := []worker.WorkerOpt{
		worker.WithClient(c.legacyClient),
		worker.WithName(name),
		worker.WithLegacySlots(int32(config.slots)), // nolint:gosec
	}

	if config.logger != nil {
		workerOpts = append(workerOpts, worker.WithLogger(config.logger))
	}

	if config.labels != nil {
		workerOpts = append(workerOpts, worker.WithLabels(config.labels))
	}

	nonDurableWorker, err := worker.NewWorker(workerOpts...)
	if err != nil {
		return nil, err
	}

	if config.panicHandler != nil {
		nonDurableWorker.SetPanicHandler(config.panicHandler)
	}

	var durableWorker *worker.Worker

	for _, dump := range dumps {
		hasDurableTasks := len(dump.durableActions) > 0

		if hasDurableTasks {
			if durableWorker == nil {
				durableWorkerOpts := []worker.WorkerOpt{
					worker.WithClient(c.legacyClient),
					worker.WithName(name + "-durable"),
					worker.WithLegacySlots(int32(config.durableSlots)), // nolint:gosec
				}

				if config.logger != nil {
					durableWorkerOpts = append(durableWorkerOpts, worker.WithLogger(config.logger))
				}

				if config.labels != nil {
					durableWorkerOpts = append(durableWorkerOpts, worker.WithLabels(config.labels))
				}

				durableWorker, err = worker.NewWorker(durableWorkerOpts...)
				if err != nil {
					return nil, err
				}

				if config.panicHandler != nil {
					durableWorker.SetPanicHandler(config.panicHandler)
				}
			}

			err := durableWorker.RegisterWorkflowV1(dump.req)
			if err != nil {
				return nil, err
			}
		} else {
			err := nonDurableWorker.RegisterWorkflowV1(dump.req)
			if err != nil {
				return nil, err
			}
		}

		for _, namedFn := range dump.durableActions {
			err = durableWorker.RegisterAction(namedFn.ActionID, namedFn.Fn)
			if err != nil {
				return nil, err
			}
		}

		for _, namedFn := range dump.regularActions {
			err = nonDurableWorker.RegisterAction(namedFn.ActionID, namedFn.Fn)
			if err != nil {
				return nil, err
			}
		}

		// Register on failure function if exists
		if dump.req.OnFailureTask != nil && dump.onFailureFn != nil {
			actionId := dump.req.OnFailureTask.Action
			onFailure := dump.onFailureFn // capture for closure
			err = nonDurableWorker.RegisterAction(actionId, func(ctx worker.HatchetContext) (any, error) {
				return onFailure(ctx)
			})
			if err != nil {
				return nil, err
			}
		}
	}

	return &Worker{
		worker:        nonDurableWorker,
		legacyDurable: durableWorker,
		name:          name,
	}, nil
}
