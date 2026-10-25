package alerting

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/hatchet-dev/hatchet/internal/integrations/alerting/alerttypes"
	"github.com/hatchet-dev/hatchet/pkg/integrations/email"
	v1 "github.com/hatchet-dev/hatchet/pkg/repository"
	"github.com/hatchet-dev/hatchet/pkg/repository/sqlcv1"
)

type fakeEmailService struct {
	*email.NoOpService
	resourceLimitSends int
	err                error
}

func (f *fakeEmailService) SendTenantResourceLimitAlert(context.Context, []string, email.ResourceLimitAlertData) error {
	f.resourceLimitSends++
	return f.err
}

func newResourceLimitSettings(enableResourceLimit, enableExpiringToken bool) *v1.GetTenantAlertingSettingsResponse {
	return &v1.GetTenantAlertingSettingsResponse{
		Settings: &sqlcv1.TenantAlertingSettings{
			EnableTenantResourceLimitAlerts: enableResourceLimit,
			EnableExpiringTokenAlerts:       enableExpiringToken,
		},
		EmailGroups: []*v1.TenantAlertEmailGroupForSend{{Emails: []string{"ops@example.com"}}},
		Tenant:      &sqlcv1.Tenant{ID: uuid.New(), Name: "acme"},
	}
}

func TestSendTenantResourceLimitAlert_GatesOnResourceLimitSetting(t *testing.T) {
	payload := &alerttypes.ResourceLimitAlert{Resource: "TASK_RUN", AlertType: "Alarm"}

	// resource limit alerts disabled, expiring token alerts enabled: nothing is sent
	fake := &fakeEmailService{}
	m := &TenantAlertManager{email: fake}
	require.NoError(t, m.sendTenantResourceLimitAlert(context.Background(), newResourceLimitSettings(false, true), payload))
	assert.Equal(t, 0, fake.resourceLimitSends)

	// resource limit alerts enabled, expiring token alerts disabled: sent once
	fake = &fakeEmailService{}
	m = &TenantAlertManager{email: fake}
	require.NoError(t, m.sendTenantResourceLimitAlert(context.Background(), newResourceLimitSettings(true, false), payload))
	assert.Equal(t, 1, fake.resourceLimitSends)
}

func TestSendTenantResourceLimitAlert_ReturnsSendErrors(t *testing.T) {
	sendErr := errors.New("postmark unavailable")
	fake := &fakeEmailService{err: sendErr}
	m := &TenantAlertManager{email: fake}

	err := m.sendTenantResourceLimitAlert(context.Background(), newResourceLimitSettings(true, false), &alerttypes.ResourceLimitAlert{})
	require.ErrorIs(t, err, sendErr)
}
