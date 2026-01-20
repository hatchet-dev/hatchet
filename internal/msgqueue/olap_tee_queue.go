package msgqueue

import (
	"context"
	"fmt"
)

// OLAPTeeMessageQueue wraps a primary MessageQueue and a standby MessageQueue.
// When enabled by the caller, it dual-writes messages sent to OLAP_QUEUE to both queues.
//
// All other operations (Subscribe, readiness, etc.) are delegated to the primary queue so that
// consumers and non-OLAP publishers are unaffected.
type OLAPTeeMessageQueue struct {
	primary MessageQueue
	standby MessageQueue
}

func NewOLAPTeeMessageQueue(primary MessageQueue, standby MessageQueue) *OLAPTeeMessageQueue {
	return &OLAPTeeMessageQueue{
		primary: primary,
		standby: standby,
	}
}

func (m *OLAPTeeMessageQueue) Clone() (func() error, MessageQueue, error) {
	if m.primary == nil {
		return nil, nil, fmt.Errorf("primary message queue is nil")
	}

	cleanupPrimary, primaryClone, err := m.primary.Clone()
	if err != nil {
		return nil, nil, err
	}

	var cleanupStandby func() error
	var standbyClone MessageQueue

	if m.standby != nil {
		cleanupStandby, standbyClone, err = m.standby.Clone()
		if err != nil {
			_ = cleanupPrimary()
			return nil, nil, err
		}
	}

	cleanup := func() error {
		if cleanupPrimary != nil {
			_ = cleanupPrimary()
		}
		if cleanupStandby != nil {
			_ = cleanupStandby()
		}
		return nil
	}

	return cleanup, &OLAPTeeMessageQueue{
		primary: primaryClone,
		standby: standbyClone,
	}, nil
}

func (m *OLAPTeeMessageQueue) SetQOS(prefetchCount int) {
	if m.primary != nil {
		m.primary.SetQOS(prefetchCount)
	}
	// Intentionally do not set standby QOS: standby is used only for publishing in this wrapper.
}

func (m *OLAPTeeMessageQueue) SendMessage(ctx context.Context, queue Queue, msg *Message) error {
	if m.primary == nil {
		return fmt.Errorf("primary message queue is nil")
	}

	// Always publish to the primary first.
	if err := m.primary.SendMessage(ctx, queue, msg); err != nil {
		return err
	}

	// Only OLAP queue messages are dual-written.
	if m.standby != nil && queue != nil && queue.Name() == OLAP_QUEUE.Name() {
		if err := m.standby.SendMessage(ctx, queue, msg); err != nil {
			return err
		}
	}

	return nil
}

func (m *OLAPTeeMessageQueue) Subscribe(queue Queue, preAck AckHook, postAck AckHook) (func() error, error) {
	if m.primary == nil {
		return nil, fmt.Errorf("primary message queue is nil")
	}

	return m.primary.Subscribe(queue, preAck, postAck)
}

func (m *OLAPTeeMessageQueue) RegisterTenant(ctx context.Context, tenantId string) error {
	if m.primary == nil {
		return fmt.Errorf("primary message queue is nil")
	}

	return m.primary.RegisterTenant(ctx, tenantId)
}

func (m *OLAPTeeMessageQueue) IsReady() bool {
	if m.primary == nil {
		return false
	}

	return m.primary.IsReady()
}
