package alerting

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/hatchet-dev/hatchet/internal/integrations/alerting/alerttypes"
	"github.com/hatchet-dev/hatchet/pkg/encryption"
	"github.com/hatchet-dev/hatchet/pkg/repository/sqlcv1"
)

const fakeWebhookSecretPath = "/services/T_TEST/B_TEST/fake-secret-marker"

func newSlackTestManager(t *testing.T, webhookURL string) (*TenantAlertManager, *sqlcv1.SlackAppWebhook) {
	t.Helper()

	masterKey, privateEc256, publicEc256, _, err := encryption.GenerateLocalKeys()
	require.NoError(t, err)

	enc, err := encryption.NewLocalEncryption(masterKey, privateEc256, publicEc256)
	require.NoError(t, err)

	encrypted, err := enc.Encrypt([]byte(webhookURL), "incoming_webhook_url")
	require.NoError(t, err)

	return &TenantAlertManager{enc: enc}, &sqlcv1.SlackAppWebhook{ID: uuid.New(), WebhookURL: encrypted}
}

func assertNoWebhookSecret(t *testing.T, err error, webhookURL string, webhookID uuid.UUID) {
	t.Helper()

	require.Error(t, err)
	assert.NotContains(t, err.Error(), webhookURL)
	assert.NotContains(t, err.Error(), fakeWebhookSecretPath)
	assert.NotContains(t, err.Error(), "fake-secret-marker")
	assert.Contains(t, err.Error(), webhookID.String())
}

func TestSlackWebhookErrors_TransportFailureDoesNotLeakURL(t *testing.T) {
	// a closed server gives a connection refused *url.Error, whose message would
	// otherwise embed the full webhook URL including its secret path
	srv := httptest.NewServer(http.NotFoundHandler())
	webhookURL := srv.URL + fakeWebhookSecretPath
	srv.Close()

	m, wh := newSlackTestManager(t, webhookURL)
	payload := &alerttypes.ResourceLimitAlert{Resource: "TASK_RUN", AlertType: "Alarm"}

	err := m.sendSlackTenantResourceLimitAlert(wh, payload)
	assertNoWebhookSecret(t, err, webhookURL, wh.ID)
	assert.True(t, strings.HasPrefix(err.Error(), "slack webhook "+wh.ID.String()+": Post: "), err.Error())

	assertNoWebhookSecret(t, m.sendSlackExpiringTokenAlert(wh, &alerttypes.ExpiringTokenItem{}), webhookURL, wh.ID)
	assertNoWebhookSecret(t, m.sendSlackWorkflowRunAlert(wh, 1, nil), webhookURL, wh.ID)
}

func TestSlackWebhookErrors_StatusFailureDoesNotLeakURL(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "no_service", http.StatusInternalServerError)
	}))
	defer srv.Close()

	webhookURL := srv.URL + fakeWebhookSecretPath
	m, wh := newSlackTestManager(t, webhookURL)

	err := m.sendSlackTenantResourceLimitAlert(wh, &alerttypes.ResourceLimitAlert{Resource: "TASK_RUN", AlertType: "Alarm"})
	assertNoWebhookSecret(t, err, webhookURL, wh.ID)
	assert.Contains(t, err.Error(), "500")
}

func TestSlackWebhookErrors_DecryptFailureDoesNotLeakURL(t *testing.T) {
	webhookURL := "https://hooks.slack.com" + fakeWebhookSecretPath
	m, wh := newSlackTestManager(t, webhookURL)

	// corrupt ciphertext cannot be decrypted
	wh.WebhookURL = []byte("not-a-valid-ciphertext")

	assertNoWebhookSecret(t, m.sendSlackTenantResourceLimitAlert(wh, &alerttypes.ResourceLimitAlert{}), webhookURL, wh.ID)
}
