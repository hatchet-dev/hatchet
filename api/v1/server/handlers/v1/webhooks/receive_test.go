package webhooksv1

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/http"
	"strconv"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"

	"github.com/hatchet-dev/hatchet/pkg/config/server"
	"github.com/hatchet-dev/hatchet/pkg/encryption"
	"github.com/hatchet-dev/hatchet/pkg/enums"
	"github.com/hatchet-dev/hatchet/pkg/repository/sqlcv1"
)

// passthroughEncryption stores secrets in plaintext so tests can sign payloads with them directly.
type passthroughEncryption struct {
	encryption.EncryptionService
}

func (passthroughEncryption) Decrypt(ciphertext []byte, _ string) ([]byte, error) {
	return ciphertext, nil
}

const testSecret = "test-signing-secret"

func hmacSHA256Hex(payload string) string {
	h := hmac.New(sha256.New, []byte(testSecret))
	h.Write([]byte(payload))
	return hex.EncodeToString(h.Sum(nil))
}

func hmacWebhook(source enums.V1IncomingWebhookSourceName, header string) sqlcv1.V1IncomingWebhook {
	algorithm := enums.V1IncomingWebhookHmacAlgorithmSHA256
	encoding := enums.V1IncomingWebhookHmacEncodingHEX

	return sqlcv1.V1IncomingWebhook{
		Name:                         "test-webhook",
		SourceName:                   source,
		AuthMethod:                   enums.V1IncomingWebhookAuthTypeHMAC,
		AuthHmacAlgorithm:            &algorithm,
		AuthHmacEncoding:             &encoding,
		AuthHmacSignatureHeaderName:  pgtype.Text{String: header, Valid: true},
		AuthHmacWebhookSigningSecret: []byte(testSecret),
	}
}

func TestValidateWebhookHMAC(t *testing.T) {
	logger := zerolog.Nop()
	svc := &V1WebhooksService{config: &server.ServerConfig{
		Encryption: passthroughEncryption{},
		Logger:     &logger,
	}}

	payload := []byte(`{"hello":"world"}`)
	now := strconv.FormatInt(time.Now().Unix(), 10)

	slackSig := "v0=" + hmacSHA256Hex(fmt.Sprintf("v0:%s:%s", now, payload))
	stripeSig := fmt.Sprintf("t=%s,v1=%s", now, hmacSHA256Hex(fmt.Sprintf("%s.%s", now, payload)))
	genericSig := hmacSHA256Hex(string(payload))

	slackHeaders := func(sig string) http.Header {
		return http.Header{"X-Slack-Request-Timestamp": {now}, "X-Slack-Signature": {sig}}
	}

	withoutAlgorithm := func(wh sqlcv1.V1IncomingWebhook) sqlcv1.V1IncomingWebhook {
		wh.AuthHmacAlgorithm = nil
		return wh
	}
	withoutEncoding := func(wh sqlcv1.V1IncomingWebhook) sqlcv1.V1IncomingWebhook {
		wh.AuthHmacEncoding = nil
		return wh
	}

	slack := hmacWebhook(enums.V1IncomingWebhookSourceNameSLACK, "X-Slack-Signature")
	stripe := hmacWebhook(enums.V1IncomingWebhookSourceNameSTRIPE, "Stripe-Signature")
	generic := hmacWebhook(enums.V1IncomingWebhookSourceNameGENERIC, "X-Signature")

	tests := []struct {
		name      string
		webhook   sqlcv1.V1IncomingWebhook
		headers   http.Header
		wantValid bool
		wantCode  int
	}{
		{"slack valid signature", slack, slackHeaders(slackSig), true, 0},
		{"slack wrong signature", slack, slackHeaders("v0=deadbeef"), false, http.StatusForbidden},
		{"slack missing algorithm", withoutAlgorithm(slack), slackHeaders(slackSig), false, http.StatusInternalServerError},
		{"slack missing encoding", withoutEncoding(slack), slackHeaders(slackSig), false, http.StatusInternalServerError},

		{"stripe valid signature", stripe, http.Header{"Stripe-Signature": {stripeSig}}, true, 0},
		{"stripe wrong signature", stripe, http.Header{"Stripe-Signature": {fmt.Sprintf("t=%s,v1=deadbeef", now)}}, false, http.StatusForbidden},
		{"stripe missing algorithm", withoutAlgorithm(stripe), http.Header{"Stripe-Signature": {stripeSig}}, false, http.StatusInternalServerError},
		{"stripe missing encoding", withoutEncoding(stripe), http.Header{"Stripe-Signature": {stripeSig}}, false, http.StatusInternalServerError},

		{"generic valid signature", generic, http.Header{"X-Signature": {genericSig}}, true, 0},
		{"generic wrong signature", generic, http.Header{"X-Signature": {"deadbeef"}}, false, http.StatusForbidden},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			valid, verr := svc.validateWebhook(payload, tt.webhook, http.Request{Header: tt.headers})

			assert.Equal(t, tt.wantValid, bool(valid))
			if tt.wantCode == 0 {
				assert.Nil(t, verr)
			} else if assert.NotNil(t, verr) {
				assert.Equal(t, tt.wantCode, verr.Code)
			}
		})
	}
}
