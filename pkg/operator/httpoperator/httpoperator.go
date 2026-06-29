package httpoperator

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/google/uuid"
	"google.golang.org/protobuf/encoding/protojson"

	"github.com/hatchet-dev/hatchet/internal/services/dispatcher/contracts"

	"github.com/hatchet-dev/hatchet/internal/signature"
	"github.com/hatchet-dev/hatchet/pkg/encryption"
	"github.com/hatchet-dev/hatchet/pkg/operator"
	"github.com/hatchet-dev/hatchet/pkg/operator/httpoperator/safeclient"
	"github.com/hatchet-dev/hatchet/pkg/repository"
	"github.com/hatchet-dev/hatchet/pkg/repository/sqlcv1"

	"github.com/rs/zerolog"
)

// SignatureHeader is the HTTP header carrying the HMAC-SHA256 (hex-encoded) signature of
// the request body. Receivers verify it by recomputing the HMAC over the raw body with
// their copy of the signing secret (see signature.Verify).
const SignatureHeader = "X-Hatchet-Signature"

// SigningSecretEncryptionDataID is the envelope-encryption data id used when encrypting and
// decrypting the operator's signing secret. The API encrypts the secret under this id
// before persisting it in the operator config, and the operator decrypts it at startup.
const SigningSecretEncryptionDataID = "v1_operator_signing_secret"

// defaultRequestTimeout bounds a single request delivery when the caller-supplied context
// has no earlier deadline. The endpoint runs the task synchronously and returns its
// result, so this is generous; it can be overridden per-operator via
// HTTPOperatorConfig.RequestTimeoutSeconds.
const defaultRequestTimeout = 60 * time.Second

const (
	// healthcheckInterval is how often the operator polls its healthcheck endpoint to
	// discover which actions the endpoint can handle.
	healthcheckInterval = 5 * time.Second

	// healthcheckTimeout bounds a single healthcheck poll.
	healthcheckTimeout = 10 * time.Second
)

// HealthcheckRequest is the body POSTed to the operator's healthcheck endpoint. It is
// currently empty (the endpoint just advertises the actions it handles) but exists as a
// typed envelope so the poll can carry additional fields in the future.
type HealthcheckRequest struct{}

// HealthcheckResponse is returned by the operator's healthcheck endpoint. Actions is the
// list of action names the endpoint handles; the operator registers these with the
// dispatcher so matching tasks are routed to it.
type HealthcheckResponse struct {
	Actions []string `json:"actions"`
}

type HTTPOperatorConfig struct {
	HealthcheckEndpoint string `json:"healthcheckEndpoint"`

	TriggerEndpoint string `json:"triggerEndpoint"`

	// SigningSecret is the encrypted (envelope-encrypted, base64) HMAC signing secret as
	// stored in the operator config. It is encrypted by the API under
	// SigningSecretEncryptionDataID and decrypted by the operator at startup; the plaintext
	// is never persisted.
	SigningSecret string `json:"signingSecret"`

	// RequestTimeoutSeconds optionally overrides defaultRequestTimeout for a single
	// delivery. The caller's context deadline still applies if it is earlier.
	RequestTimeoutSeconds int `json:"requestTimeoutSeconds"`

	// Slots is the number of concurrent task slots the operator's worker advertises.
	// Defaults to defaultOperatorSlots when unset.
	Slots int `json:"slots"`
}

// defaultOperatorSlots is the worker slot count used when an operator does not configure one.
const defaultOperatorSlots = 100

// SlotConfig returns the worker slot config (slot_type -> max units) for an HTTP operator,
// derived from its stored config. It is used to provision the operator's worker and may
// vary between operators.
func SlotConfig(op *sqlcv1.V1Operator) (map[string]int32, error) {
	var cfg HTTPOperatorConfig

	if err := json.Unmarshal(op.Config, &cfg); err != nil {
		return nil, fmt.Errorf("could not unmarshal operator config: %w", err)
	}

	slots := cfg.Slots

	if slots <= 0 {
		slots = defaultOperatorSlots
	}

	return map[string]int32{repository.SlotTypeDefault: int32(slots)}, nil
}

// requestSender is the subset of safeclient.Sender used to deliver requests. It is an
// interface so tests can inject a fake that captures the signed request.
type requestSender interface {
	Deliver(ctx context.Context, method, endpoint string, body []byte, headers http.Header) (*safeclient.DeliveryResult, error)
}

type HTTPOperator struct {
	*operator.SharedOperator[HTTPOperatorConfig]

	sender requestSender

	// signingSecret is the decrypted HMAC signing secret. The config stores it encrypted;
	// it is decrypted once at construction and held in memory for signing.
	signingSecret string

	// cancel stops the healthcheck polling goroutine on Cleanup.
	cancel context.CancelFunc

	// lastActions is the most recently registered action set, used to avoid redundant
	// dispatcher writes when the healthcheck returns an unchanged list. Only the polling
	// goroutine touches it.
	lastActions []string
}

func NewHTTPOperator(op *sqlcv1.V1Operator, l *zerolog.Logger, repo repository.Repository, taskEventWriter operator.TaskEventWriter, enc encryption.EncryptionService, infraBlockedCIDRs []string, workerId uuid.UUID) (*HTTPOperator, error) {
	shared, err := operator.NewSharedOperator(op, l, repo, taskEventWriter, workerId, HTTPOperatorConfig{})

	if err != nil {
		return nil, err
	}

	// Block our own infrastructure CIDRs (sourced from runtime config) on top of the
	// built-in reserved/private denylist. An empty list is permitted for now (experimental)
	// so the engine still starts when SERVER_OPERATOR_INFRA_BLOCKED_CIDRS is unset; the
	// default denylist always applies.
	sender, err := safeclient.New(safeclient.Config{
		InfraBlockedCIDRs:    infraBlockedCIDRs,
		AllowEmptyInfraCIDRs: len(infraBlockedCIDRs) == 0,
	}, l)

	if err != nil {
		return nil, fmt.Errorf("could not construct request sender: %w", err)
	}

	// The signing secret is stored encrypted; decrypt it once for use when signing.
	signingSecret := ""

	if encrypted := shared.Config().SigningSecret; encrypted != "" {
		if enc == nil {
			return nil, fmt.Errorf("encryption service is required to decrypt operator signing secret")
		}

		signingSecret, err = enc.DecryptString(encrypted, SigningSecretEncryptionDataID)

		if err != nil {
			return nil, fmt.Errorf("could not decrypt operator signing secret: %w", err)
		}
	}

	ctx, cancel := context.WithCancel(context.Background())

	h := &HTTPOperator{
		SharedOperator: shared,
		sender:         sender,
		signingSecret:  signingSecret,
		cancel:         cancel,
	}

	go h.pollHealthcheck(ctx)

	return h, nil
}

// Cleanup stops the healthcheck poller in addition to the shared operator's goroutines.
func (h *HTTPOperator) Cleanup() {
	if h.cancel != nil {
		h.cancel()
	}

	h.SharedOperator.Cleanup()
}

// Drain stops the healthcheck poller and drains in-flight tasks without pausing the worker
// (used for bulk teardown, where the caller pauses all operator workers in one query).
func (h *HTTPOperator) Drain() {
	if h.cancel != nil {
		h.cancel()
	}

	h.SharedOperator.Drain()
}

// pollHealthcheck periodically polls the healthcheck endpoint to discover the actions the
// endpoint handles, registering any changes with the dispatcher.
func (h *HTTPOperator) pollHealthcheck(ctx context.Context) {
	ticker := time.NewTicker(healthcheckInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			h.refreshActions(ctx)
		}
	}
}

func (h *HTTPOperator) refreshActions(ctx context.Context) {
	cfg := h.configWithSecret()

	if cfg.HealthcheckEndpoint == "" {
		return
	}

	pollCtx, cancel := context.WithTimeout(ctx, healthcheckTimeout)
	defer cancel()

	actions, err := pollActions(pollCtx, h.sender, h.Logger(), cfg)

	if err != nil {
		h.Logger().Error().Err(err).
			Str("healthcheck_endpoint", cfg.HealthcheckEndpoint).
			Msg("could not poll operator healthcheck")

		return
	}

	if operator.SlicesEqualUnordered(actions, h.lastActions) {
		return
	}

	if err := h.UpdateWorkerActions(ctx, actions); err != nil {
		h.Logger().Error().Err(err).Msg("could not update operator worker actions")
		return
	}

	h.lastActions = actions

	h.Logger().Debug().Strs("actions", actions).Msg("updated operator worker actions from healthcheck")
}

// pollActions issues a signed healthcheck request and returns the advertised action names.
// It is a free function so it can be unit-tested with a fake sender.
func pollActions(ctx context.Context, sender requestSender, l *zerolog.Logger, cfg HTTPOperatorConfig) ([]string, error) {
	body, err := json.Marshal(HealthcheckRequest{})

	if err != nil {
		return nil, fmt.Errorf("could not build healthcheck request: %w", err)
	}

	headers, err := signedHeaders(l, cfg.SigningSecret, body)

	if err != nil {
		return nil, err
	}

	res, err := sender.Deliver(ctx, http.MethodPost, cfg.HealthcheckEndpoint, body, headers)

	if err != nil {
		return nil, fmt.Errorf("could not reach healthcheck endpoint %q: %w", cfg.HealthcheckEndpoint, err)
	}

	if res.StatusCode < http.StatusOK || res.StatusCode >= http.StatusMultipleChoices {
		return nil, fmt.Errorf("healthcheck endpoint %q returned status %d", cfg.HealthcheckEndpoint, res.StatusCode)
	}

	var resp HealthcheckResponse

	if err := json.Unmarshal(res.BodyPrefix, &resp); err != nil {
		return nil, fmt.Errorf("could not parse healthcheck response: %w", err)
	}

	return resp.Actions, nil
}


func (h *HTTPOperator) HandleAction(ctx context.Context, action *contracts.AssignedAction) error {
	// Track this task so Cleanup drains it before the operator shuts down.
	release := h.RecordTask()
	defer release()

	switch action.ActionType {
	case contracts.ActionType_START_STEP_RUN:
		return h.deliver(ctx, action)
	default:
		// TODO: support CANCEL_STEP_RUN (e.g. a configurable cancel endpoint) and
		// START_GET_GROUP_KEY. Until then, acknowledge without delivering.
		h.Logger().Warn().
			Str("action_type", action.ActionType.String()).
			Str("task_run_external_id", action.TaskRunExternalId).
			Msg("http operator received unsupported action type; skipping")

		return nil
	}
}

func (h *HTTPOperator) deliver(ctx context.Context, action *contracts.AssignedAction) error {
	// Report STARTED before delivery so the task is marked running. Best-effort: a failed
	// report shouldn't prevent the actual delivery.
	if err := h.SendStarted(action); err != nil {
		h.Logger().Error().Err(err).
			Str("task_run_external_id", action.TaskRunExternalId).
			Msg("could not report task started")
	}

	cfg := h.configWithSecret()

	res, err := deliverAction(ctx, h.sender, h.Logger(), cfg, action)

	if err != nil {
		// Delivery failed (transport error or non-2xx): report the failure.
		if reportErr := h.SendFailed(action, err.Error(), false); reportErr != nil {
			h.Logger().Error().Err(reportErr).
				Str("task_run_external_id", action.TaskRunExternalId).
				Msg("could not report task failure")
		}

		return err
	}

	// Delivery succeeded: report the response body as the task output. The dispatcher
	// requires valid JSON output, so default an empty body to an empty object.
	output := res.BodyPrefix

	if len(output) == 0 {
		output = []byte("{}")
	}

	if err := h.SendCompleted(action, output); err != nil {
		return fmt.Errorf("could not report task completion: %w", err)
	}

	return nil
}

// configWithSecret returns a copy of the operator config with the decrypted signing secret
// substituted in, so the deliver/poll helpers can sign with plaintext.
func (h *HTTPOperator) configWithSecret() HTTPOperatorConfig {
	cfg := h.Config()
	cfg.SigningSecret = h.signingSecret

	return cfg
}

// deliverAction builds the signed envelope, applies the timeout backstop, delivers via the
// SSRF-hardened sender, and surfaces non-2xx responses as failures. On success it returns
// the delivery result (including the response body prefix). It is a free function (rather
// than a method) so it can be unit-tested with a fake sender.
func deliverAction(ctx context.Context, sender requestSender, l *zerolog.Logger, cfg HTTPOperatorConfig, action *contracts.AssignedAction) (*safeclient.DeliveryResult, error) {
	if cfg.TriggerEndpoint == "" {
		return nil, fmt.Errorf("http operator has no trigger endpoint configured")
	}

	body, err := buildPayload(action)

	if err != nil {
		return nil, fmt.Errorf("could not build request payload: %w", err)
	}

	headers, err := signedHeaders(l, cfg.SigningSecret, body)

	if err != nil {
		return nil, err
	}

	// The caller owns the deadline; if it has none, apply our own backstop so a hung
	// endpoint cannot block the operator indefinitely.
	if _, ok := ctx.Deadline(); !ok {
		timeout := defaultRequestTimeout

		if cfg.RequestTimeoutSeconds > 0 {
			timeout = time.Duration(cfg.RequestTimeoutSeconds) * time.Second
		}

		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, timeout)
		defer cancel()
	}

	res, err := sender.Deliver(ctx, http.MethodPost, cfg.TriggerEndpoint, body, headers)

	if err != nil {
		return nil, fmt.Errorf("could not deliver request to %q: %w", cfg.TriggerEndpoint, err)
	}

	if res.StatusCode < http.StatusOK || res.StatusCode >= http.StatusMultipleChoices {
		return nil, fmt.Errorf("request delivery to %q returned status %d", cfg.TriggerEndpoint, res.StatusCode)
	}

	return res, nil
}

// signedHeaders returns the headers for an outbound request, signing the body with the
// HMAC secret under SignatureHeader so the receiver can verify it. A missing secret is a
// misconfiguration: the request is sent unsigned with a loud warning.
func signedHeaders(l *zerolog.Logger, secret string, body []byte) (http.Header, error) {
	headers := http.Header{}

	if secret == "" {
		if l != nil {
			l.Warn().Msg("http operator has no signing secret; sending request without signature")
		}

		return headers, nil
	}

	sig, err := signature.Sign(string(body), secret)

	if err != nil {
		return nil, fmt.Errorf("could not sign request payload: %w", err)
	}

	headers.Set(SignatureHeader, sig)

	return headers, nil
}

// buildPayload serializes the assigned action to its protobuf JSON representation. Using
// protojson (rather than a hand-maintained struct) means new fields added to the
// AssignedAction proto are included automatically, with their proto JSON names.
//
// Note: ActionPayload is a proto string field, so it is emitted as a JSON string
// containing the engine's JSON payload, not as a nested object.
func buildPayload(action *contracts.AssignedAction) ([]byte, error) {
	return protojson.Marshal(action)
}
