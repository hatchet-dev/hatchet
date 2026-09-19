package serverlessoperator

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"math/rand/v2"
	"net/http"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/google/uuid"
	"google.golang.org/protobuf/proto"

	v1 "github.com/hatchet-dev/hatchet/internal/services/shared/proto/v1"
	"github.com/hatchet-dev/hatchet/internal/signature"
	"github.com/hatchet-dev/hatchet/pkg/operator/safeclient"
	"github.com/hatchet-dev/hatchet/pkg/serverlessoperator/contract"
	"github.com/hatchet-dev/hatchet/pkg/validator"
)

// RequestSender is the outbound HTTP seam: safeclient.Sender in production, a fake in tests.
type RequestSender interface {
	Deliver(ctx context.Context, method, endpoint string, body []byte, headers http.Header) (*safeclient.DeliveryResult, error)
}

// healthcheckResult is a parsed, namespaced healthcheck response. workflowHashes are the
// per-workflow canonical hashes, parallel to workflows, so a poller can put only the
// workflows that changed; hash covers the whole response.
type healthcheckResult struct {
	// runtime is nil when the endpoint did not report one.
	runtime        *v1.ServerlessRuntime
	hash           string
	workflows      []*v1.CreateWorkflowVersionRequest
	workflowHashes []string
	actions        []string
	durable        bool
}

// signedHeaders returns the headers every endpoint request carries: the HMAC-SHA256 hex
// signature of the raw body, the endpoint id and the timestamp the body was built with.
func signedHeaders(secret string, endpointId uuid.UUID, timestamp int64, body []byte) (http.Header, error) {
	if secret == "" {
		return nil, fmt.Errorf("endpoint %s has no signing secret", endpointId)
	}

	sig, err := signature.Sign(string(body), secret)

	if err != nil {
		return nil, fmt.Errorf("could not sign request: %w", err)
	}

	headers := http.Header{}
	headers.Set(contract.SignatureHeader, sig)
	headers.Set(contract.EndpointIdHeader, endpointId.String())
	headers.Set(contract.TimestampHeader, strconv.FormatInt(timestamp, 10))

	return headers, nil
}

// catalogLimits caps what one healthcheck may advertise.
type catalogLimits struct {
	maxWorkflows int
	maxActions   int
}

// actionRules is the engine's own action id validation, applied to every advertised action
// before it can enter a tenant's shared union.
var actionRules = validator.NewDefaultValidator()

// maxActionIdBytes caps a namespaced action id. The engine stores ids as text with no cap of
// its own; this one keeps a catalog from carrying ids that are only there to be large.
const maxActionIdBytes = 1024

// validateActionStorage rejects action ids the engine's rules accept but its storage cannot
// hold or should not: a NUL code point (text columns refuse it, and the rejection would
// arrive only after the id had entered the session's desired set), invalid UTF-8, and ids
// over maxActionIdBytes. It runs before the union or the session sees the catalog.
func validateActionStorage(actions []string) error {
	for _, action := range actions {
		switch {
		case strings.IndexByte(action, 0) >= 0:
			return fmt.Errorf("invalid action %q: contains a NUL code point", action)
		case !utf8.ValidString(action):
			return fmt.Errorf("invalid action %q: not valid UTF-8", action)
		case len(action) > maxActionIdBytes:
			return fmt.Errorf("invalid action %q: longer than %d bytes", action[:32]+"...", maxActionIdBytes)
		}
	}

	return nil
}

type advertisedActions struct {
	Actions []string `validate:"dive,actionId"`
}

// pollHealthcheck issues one signed healthcheck and parses the response. The caller owns the
// deadline and the concurrency limits.
func pollHealthcheck(ctx context.Context, sender RequestSender, ep *cachedEndpoint, cfg *endpointConfig, limits catalogLimits) (*healthcheckResult, error) {
	if cfg.secretErr != nil {
		return nil, cfg.secretErr
	}

	now := time.Now().Unix()

	body, err := contract.Marshal(&v1.ServerlessHealthcheckRequest{
		Timestamp:  now,
		EndpointId: ep.id.String(),
		Namespace:  ep.namespace.String(),
	})

	if err != nil {
		return nil, fmt.Errorf("could not build healthcheck request: %w", err)
	}

	headers, err := signedHeaders(cfg.secret, ep.id, now, body)

	if err != nil {
		return nil, err
	}

	res, err := sender.Deliver(ctx, http.MethodPost, cfg.healthcheckUrl, body, headers)

	if err != nil {
		return nil, fmt.Errorf("healthcheck request failed: %w", err)
	}

	if res.StatusCode < http.StatusOK || res.StatusCode >= http.StatusMultipleChoices {
		return nil, fmt.Errorf("healthcheck returned status %d", res.StatusCode)
	}

	return parseHealthcheckResponse(res.BodyPrefix, ep.namespace, limits)
}

// parseHealthcheckResponse applies the namespace to the advertised workflows, derives the
// action set (the workflows' actions plus any the endpoint lists explicitly), validates every
// action with the engine's rules, enforces the catalog caps, and hashes the canonical
// (namespaced, deterministic) form so an unchanged response, however the endpoint formats it,
// produces the same hash.
func parseHealthcheckResponse(body []byte, ns uuid.UUID, limits catalogLimits) (*healthcheckResult, error) {
	resp := &v1.ServerlessHealthcheckResponse{}

	if err := contract.Unmarshal(body, resp); err != nil {
		return nil, fmt.Errorf("could not parse healthcheck response: %w", err)
	}

	if limits.maxWorkflows > 0 && len(resp.Workflows) > limits.maxWorkflows {
		return nil, fmt.Errorf("healthcheck advertises %d workflows, more than the limit of %d", len(resp.Workflows), limits.maxWorkflows)
	}

	if limits.maxActions > 0 && len(resp.Actions) > limits.maxActions {
		return nil, fmt.Errorf("healthcheck advertises %d actions, more than the limit of %d", len(resp.Actions), limits.maxActions)
	}

	hasher := sha256.New()

	workflows := make([]*v1.CreateWorkflowVersionRequest, 0, len(resp.Workflows))
	workflowHashes := make([]string, 0, len(resp.Workflows))
	actionLists := make([][]string, 0, len(resp.Workflows)+1)

	for _, wf := range resp.Workflows {
		namespaced, err := applyNamespace(wf, ns)

		if err != nil {
			return nil, err
		}

		actions, err := actionsForWorkflow(namespaced)

		if err != nil {
			return nil, err
		}

		canonical, err := proto.MarshalOptions{Deterministic: true}.Marshal(namespaced)

		if err != nil {
			return nil, fmt.Errorf("could not hash workflow %s: %w", wf.Name, err)
		}

		hasher.Write(canonical)
		hasher.Write([]byte{0})

		wfHash := sha256.Sum256(canonical)

		workflows = append(workflows, namespaced)
		workflowHashes = append(workflowHashes, hex.EncodeToString(wfHash[:]))
		actionLists = append(actionLists, actions)
	}

	extra := make([]string, 0, len(resp.Actions))

	for _, action := range resp.Actions {
		if strings.TrimSpace(action) == "" {
			continue
		}

		prefixed, err := prefixAction(ns, action)

		if err != nil {
			return nil, fmt.Errorf("invalid action %q: %w", action, err)
		}

		extra = append(extra, prefixed)
	}

	actionLists = append(actionLists, extra)
	actions := sortedUnion(actionLists...)

	if limits.maxActions > 0 && len(actions) > limits.maxActions {
		return nil, fmt.Errorf("healthcheck advertises %d actions, more than the limit of %d", len(actions), limits.maxActions)
	}

	if err := actionRules.Validate(advertisedActions{Actions: actions}); err != nil {
		return nil, fmt.Errorf("healthcheck advertises an invalid action: %w", err)
	}

	if err := validateActionStorage(actions); err != nil {
		return nil, fmt.Errorf("healthcheck advertises an invalid action: %w", err)
	}

	hasher.Write([]byte(strings.Join(actions, "\n")))

	return &healthcheckResult{
		workflows:      workflows,
		workflowHashes: workflowHashes,
		actions:        actions,
		hash:           hex.EncodeToString(hasher.Sum(nil)),
		durable:        resp.GetDurable().GetSupported(),
		runtime:        resp.Runtime,
	}, nil
}

// pollInterval spreads polls of endpoints created together by up to 10 percent either way.
func pollInterval(seconds int32) time.Duration {
	base := pollBase(seconds)
	jitter := 0.9 + rand.Float64()*0.2 // #nosec G404 -- jitter, not security

	return time.Duration(float64(base) * jitter)
}

// firstPollDelay spreads the first polls of a gained unit over up to a tenth of the interval,
// at most one second, so registration is not delayed for long.
func firstPollDelay(seconds int32) time.Duration {
	spread := min(pollBase(seconds)/10, time.Second)

	return time.Duration(rand.Float64() * float64(spread)) // #nosec G404 -- jitter, not security
}

func pollBase(seconds int32) time.Duration {
	if seconds <= 0 {
		seconds = 30
	}

	return time.Duration(seconds) * time.Second
}
