package serverlessoperator

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math/rand/v2"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"

	v1 "github.com/hatchet-dev/hatchet/internal/services/shared/proto/v1"
	"github.com/hatchet-dev/hatchet/internal/signature"
	"github.com/hatchet-dev/hatchet/pkg/operator/safeclient"
	"github.com/hatchet-dev/hatchet/pkg/serverlessoperator/contract"
)

// RequestSender is the outbound HTTP seam: safeclient.Sender in production, a fake in tests.
type RequestSender interface {
	Deliver(ctx context.Context, method, endpoint string, body []byte, headers http.Header) (*safeclient.DeliveryResult, error)
}

// healthcheckResult is a parsed, namespaced healthcheck response. workflowHashes are the
// per-workflow canonical hashes, parallel to workflows, so a poller can put only the
// workflows that changed; hash covers the whole response.
type healthcheckResult struct {
	runtime        contract.HealthcheckRuntime
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

// pollHealthcheck issues one signed healthcheck and parses the response. The caller owns the
// deadline and the process-wide concurrency limit.
func pollHealthcheck(ctx context.Context, sender RequestSender, ep *cachedEndpoint, cfg *endpointConfig) (*healthcheckResult, error) {
	if cfg.secretErr != nil {
		return nil, cfg.secretErr
	}

	now := time.Now().Unix()

	body, err := json.Marshal(contract.HealthcheckRequest{
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

	return parseHealthcheckResponse(res.BodyPrefix, ep.namespace)
}

// parseHealthcheckResponse applies the namespace to the advertised workflows, derives the
// action set and hashes the canonical (namespaced, deterministic) form so an unchanged
// response, however the endpoint formats it, produces the same hash.
func parseHealthcheckResponse(body []byte, ns uuid.UUID) (*healthcheckResult, error) {
	var resp contract.HealthcheckResponse

	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("could not parse healthcheck response: %w", err)
	}

	unmarshal := protojson.UnmarshalOptions{DiscardUnknown: true}
	hasher := sha256.New()

	workflows := make([]*v1.CreateWorkflowVersionRequest, 0, len(resp.Workflows))
	workflowHashes := make([]string, 0, len(resp.Workflows))
	actionLists := make([][]string, 0, len(resp.Workflows)+1)

	for i, raw := range resp.Workflows {
		wf := &v1.CreateWorkflowVersionRequest{}

		if err := unmarshal.Unmarshal(raw, wf); err != nil {
			return nil, fmt.Errorf("could not parse workflow at index %d: %w", i, err)
		}

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

	hasher.Write([]byte(strings.Join(actions, "\n")))

	result := &healthcheckResult{
		workflows:      workflows,
		workflowHashes: workflowHashes,
		actions:        actions,
		hash:           hex.EncodeToString(hasher.Sum(nil)),
	}

	if resp.Durable != nil {
		result.durable = resp.Durable.Supported
	}

	if resp.Runtime != nil {
		result.runtime = *resp.Runtime
	}

	return result, nil
}

// pollInterval spreads polls of endpoints created together by up to 10 percent either way.
func pollInterval(seconds int32) time.Duration {
	if seconds <= 0 {
		seconds = 30
	}

	base := time.Duration(seconds) * time.Second
	jitter := 0.9 + rand.Float64()*0.2 // #nosec G404 -- jitter, not security

	return time.Duration(float64(base) * jitter)
}
