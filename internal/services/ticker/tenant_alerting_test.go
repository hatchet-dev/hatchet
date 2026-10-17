//go:build !e2e && !load && !rampup && !integration

package ticker

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/hatchet-dev/hatchet/pkg/limitalerts"
	"github.com/hatchet-dev/hatchet/pkg/repository/sqlcv1"
)

type fakeTenantAlerter struct {
	sent []uuid.UUID
	err  error
}

func (f *fakeTenantAlerter) SendExpiringTokenAlert(uuid.UUID, *sqlcv1.PollExpiringTokensRow) error {
	return nil
}

func (f *fakeTenantAlerter) SendTenantResourceLimitAlert(tenantId uuid.UUID, _ *sqlcv1.TenantResourceLimitAlert) error {
	f.sent = append(f.sent, tenantId)
	return f.err
}

func newLimitAlertRow(tenantId uuid.UUID) *sqlcv1.TenantResourceLimitAlert {
	return &sqlcv1.TenantResourceLimitAlert{
		ID:              uuid.New(),
		CreatedAt:       pgtype.Timestamp{Time: time.Date(2026, 10, 17, 12, 0, 0, 0, time.UTC), Valid: true},
		ResourceLimitId: uuid.New(),
		TenantId:        tenantId,
		Resource:        sqlcv1.LimitResourceTASKRUN,
		AlertType:       sqlcv1.TenantResourceLimitAlertTypeAlarm,
		Value:           1640,
		Limit:           2000,
	}
}

func newTestTicker(ta tenantAlerter, hook limitalerts.Hook) *TickerImpl {
	l := zerolog.Nop()
	return &TickerImpl{l: &l, ta: ta, limitAlertHook: hook}
}

func TestDispatchTenantResourceLimitAlert_CallsHookWithRowFields(t *testing.T) {
	tenantId := uuid.New()
	row := newLimitAlertRow(tenantId)

	var got []limitalerts.Alert
	hook := func(_ context.Context, a limitalerts.Alert) error {
		got = append(got, a)
		return nil
	}

	ta := &fakeTenantAlerter{}
	tk := newTestTicker(ta, hook)

	require.NoError(t, tk.dispatchTenantResourceLimitAlert(context.Background(), row))

	assert.Equal(t, []uuid.UUID{tenantId}, ta.sent)
	require.Len(t, got, 1)
	assert.Equal(t, limitalerts.Alert{
		ID:        row.ID,
		TenantID:  tenantId,
		Resource:  "TASK_RUN",
		AlertType: limitalerts.AlertTypeAlarm,
		Value:     1640,
		Limit:     2000,
		CreatedAt: row.CreatedAt.Time,
	}, got[0])
}

func TestDispatchTenantResourceLimitAlert_HookErrorDoesNotStopOtherAlerts(t *testing.T) {
	first := newLimitAlertRow(uuid.New())
	second := newLimitAlertRow(uuid.New())

	hookErr := errors.New("boom")
	calls := 0
	hook := func(_ context.Context, a limitalerts.Alert) error {
		calls++
		if a.ID == first.ID {
			return hookErr
		}
		return nil
	}

	ta := &fakeTenantAlerter{}
	tk := newTestTicker(ta, hook)

	err := tk.dispatchTenantResourceLimitAlert(context.Background(), first)
	require.ErrorIs(t, err, hookErr)

	require.NoError(t, tk.dispatchTenantResourceLimitAlert(context.Background(), second))

	// the tenant-facing alerter still ran for both, and the hook saw both
	assert.Equal(t, []uuid.UUID{first.TenantId, second.TenantId}, ta.sent)
	assert.Equal(t, 2, calls)
}

func TestDispatchTenantResourceLimitAlert_AlerterErrorStillCallsHook(t *testing.T) {
	row := newLimitAlertRow(uuid.New())

	alerterErr := errors.New("smtp down")
	hookCalled := false
	hook := func(context.Context, limitalerts.Alert) error {
		hookCalled = true
		return nil
	}

	tk := newTestTicker(&fakeTenantAlerter{err: alerterErr}, hook)

	err := tk.dispatchTenantResourceLimitAlert(context.Background(), row)
	require.ErrorIs(t, err, alerterErr)
	assert.True(t, hookCalled)
}

func TestDispatchTenantResourceLimitAlert_NilHookIsNoop(t *testing.T) {
	row := newLimitAlertRow(uuid.New())
	ta := &fakeTenantAlerter{}
	tk := newTestTicker(ta, nil)

	require.NoError(t, tk.dispatchTenantResourceLimitAlert(context.Background(), row))
	assert.Len(t, ta.sent, 1)
}
