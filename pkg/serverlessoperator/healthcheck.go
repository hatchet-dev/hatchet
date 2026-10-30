package serverlessoperator

import (
	"context"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"fmt"
	"math/rand/v2"
	"net/http"
	"strconv"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	"github.com/google/uuid"
	"google.golang.org/protobuf/proto"

	"github.com/hatchet-dev/hatchet/internal/enumerables"
	v1 "github.com/hatchet-dev/hatchet/internal/services/shared/proto/v1"
	"github.com/hatchet-dev/hatchet/pkg/operator/safeclient"
	"github.com/hatchet-dev/hatchet/pkg/serverlessoperator/contract"
	"github.com/hatchet-dev/hatchet/pkg/validator"
)

// RequestSender is the outbound HTTP seam: safeclient.Sender in production, a fake in tests.
type RequestSender interface {
	Deliver(ctx context.Context, method, endpoint string, body []byte, headers http.Header) (*safeclient.DeliveryResult, error)
}

// healthcheckResult is a parsed healthcheck response. workflowHashes are the per-workflow
// canonical hashes, parallel to workflows, so a poller can put only the workflows that
// changed; hash covers the whole response.
type healthcheckResult struct {
	// runtime is nil when the endpoint did not report one.
	runtime        *v1.ServerlessRuntime
	hash           string
	workflows      []*v1.CreateWorkflowVersionRequest
	workflowHashes []string
	actions        []string
	// streamActions is the sorted subset of actions whose task asked for an invocation
	// websocket (ServerlessTaskOptions.streams).
	streamActions []string
	// durable is whether the endpoint accepts the durable websocket protocol. When it does
	// not, actions holds no task declared durable: the endpoint is never registered for one,
	// so the engine never assigns it one.
	durable bool
}

// signedHeaders returns the headers every endpoint request carries: the HMAC-SHA256 hex
// signature of the raw body, the endpoint id and the timestamp the body was built with.
func signedHeaders(secret string, endpointId uuid.UUID, timestamp int64, body []byte) (http.Header, error) {
	if secret == "" {
		return nil, fmt.Errorf("endpoint %s has no signing secret", endpointId)
	}

	sig, err := contract.Sign(string(body), secret)

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

// maxActionIdBytes caps an action id. The engine stores ids as text with no cap of its own;
// this one keeps a catalog from carrying ids that are only there to be large.
const maxActionIdBytes = 1024

// validateActionStorage rejects action ids the engine's rules accept but its storage cannot
// hold or should not: invalid UTF-8, a control character (a NUL code point is refused by text
// columns, and the rejection would arrive only after the id had entered the session's desired
// set; a newline or any other control character has no place in an id that is logged, shown
// and compared as a line of text), and ids over maxActionIdBytes. It runs before the union or
// the session sees the catalog.
func validateActionStorage(actions []string) error {
	for _, action := range actions {
		switch {
		case !utf8.ValidString(action):
			return fmt.Errorf("invalid action %q: not valid UTF-8", action)
		case strings.IndexFunc(action, unicode.IsControl) >= 0:
			return fmt.Errorf("invalid action %q: contains a control character", action)
		case len(action) > maxActionIdBytes:
			return fmt.Errorf("invalid action %q: longer than %d bytes", action[:32]+"...", maxActionIdBytes)
		}
	}

	return nil
}

type advertisedActions struct {
	Actions []string `validate:"dive,actionId"`
}

// healthcheckError is a failed healthcheck with how far it got. Its message is the wrapped
// error's alone: it becomes the endpoint's status error, which is compared to decide status
// writes, so the response it carries (whose body may change on every poll) never enters it.
type healthcheckError struct {
	err        error
	stage      string
	body       []byte
	statusCode int
}

func (e *healthcheckError) Error() string { return e.err.Error() }
func (e *healthcheckError) Unwrap() error { return e.err }

func stageError(stage string, err error) *healthcheckError {
	return &healthcheckError{stage: stage, err: err}
}

// healthcheckStage is the stage err failed at; an error that is not a healthcheckError
// failed before any response.
func healthcheckStage(err error) *healthcheckError {
	var hcErr *healthcheckError

	if errors.As(err, &hcErr) {
		return hcErr
	}

	return stageError(stageRequest, err)
}

// pollHealthcheck issues one signed healthcheck and parses the response. The caller owns the
// deadline and the concurrency limits. A failure is a *healthcheckError.
func pollHealthcheck(ctx context.Context, sender RequestSender, ep *cachedEndpoint, cfg *endpointConfig, limits catalogLimits) (*healthcheckResult, error) {
	if cfg.secretErr != nil {
		return nil, stageError(stageRequest, cfg.secretErr)
	}

	now := time.Now().Unix()

	body, err := contract.Marshal(&v1.ServerlessHealthcheckRequest{
		TimestampUnixSeconds: now,
		EndpointId:           ep.id.String(),
	})

	if err != nil {
		return nil, stageError(stageRequest, fmt.Errorf("could not build healthcheck request: %w", err))
	}

	headers, err := signedHeaders(cfg.secret, ep.id, now, body)

	if err != nil {
		return nil, stageError(stageRequest, err)
	}

	res, err := sender.Deliver(ctx, http.MethodPost, cfg.healthcheckUrl, body, headers)

	if err != nil {
		return nil, stageError(stageRequest, fmt.Errorf("healthcheck request failed: %w", err))
	}

	if res.StatusCode < http.StatusOK || res.StatusCode >= http.StatusMultipleChoices {
		return nil, &healthcheckError{
			stage:      stageStatus,
			statusCode: res.StatusCode,
			body:       res.BodyPrefix,
			err:        fmt.Errorf("healthcheck returned status %d", res.StatusCode),
		}
	}

	parsed, err := parseHealthcheckResponse(res.BodyPrefix, limits)

	if err != nil {
		hcErr := healthcheckStage(err)
		hcErr.statusCode = res.StatusCode
		hcErr.body = res.BodyPrefix

		return nil, hcErr
	}

	return parsed, nil
}

// parseHealthcheckResponse canonicalizes the advertised workflows, settles the served action
// set (the ids the endpoint lists in actions when there are any, otherwise every action
// derived from its workflows, less every durable task when the endpoint does not support
// durable), validates every served action with the engine's rules, enforces the catalog
// caps, and hashes the canonical (deterministic) form so an unchanged response, however the
// endpoint formats it, produces the same hash.
func parseHealthcheckResponse(body []byte, limits catalogLimits) (*healthcheckResult, error) {
	resp := &v1.ServerlessHealthcheckResponse{}

	if err := contract.Unmarshal(body, resp); err != nil {
		return nil, stageError(stageParse, fmt.Errorf("could not parse healthcheck response: %w", err))
	}

	res, err := parseCatalog(resp, limits)

	if err != nil {
		return nil, stageError(stageCatalog, err)
	}

	return res, nil
}

// parseCatalog is the part of parseHealthcheckResponse after decoding: every error it returns
// is the catalog being refused.
func parseCatalog(resp *v1.ServerlessHealthcheckResponse, limits catalogLimits) (*healthcheckResult, error) {

	if limits.maxWorkflows > 0 && len(resp.Workflows) > limits.maxWorkflows {
		return nil, fmt.Errorf("healthcheck advertises %d workflows, more than the limit of %d", len(resp.Workflows), limits.maxWorkflows)
	}

	if limits.maxActions > 0 && len(resp.Actions) > limits.maxActions {
		return nil, fmt.Errorf("healthcheck advertises %d actions, more than the limit of %d", len(resp.Actions), limits.maxActions)
	}

	workflows := make([]*v1.CreateWorkflowVersionRequest, 0, len(resp.Workflows))
	workflowHashes := make([]string, 0, len(resp.Workflows))
	workflowBytes := make([][]byte, 0, len(resp.Workflows))
	derived := make([][]string, 0, len(resp.Workflows))
	durableActions := map[string]struct{}{}

	for _, wf := range resp.Workflows {
		canonicalWf, err := canonicalWorkflow(wf)

		if err != nil {
			return nil, err
		}

		actions, err := actionsForWorkflow(canonicalWf)

		if err != nil {
			return nil, err
		}

		for _, action := range durableActionsOf(canonicalWf) {
			durableActions[action] = struct{}{}
		}

		canonical, err := proto.MarshalOptions{Deterministic: true}.Marshal(canonicalWf)

		if err != nil {
			return nil, fmt.Errorf("could not hash workflow %s: %w", wf.Name, err)
		}

		wfHash := sha256.Sum256(canonical)

		workflows = append(workflows, canonicalWf)
		workflowHashes = append(workflowHashes, hex.EncodeToString(wfHash[:]))
		workflowBytes = append(workflowBytes, canonical)
		derived = append(derived, actions)
	}

	listed := make([]string, 0, len(resp.Actions))

	for _, action := range resp.Actions {
		if strings.TrimSpace(action) == "" {
			continue
		}

		normalized, err := normalizeAction(action)

		if err != nil {
			return nil, fmt.Errorf("invalid action %q: %w", action, err)
		}

		listed = append(listed, normalized)
	}

	// A listed set is the served set as the endpoint states it, which may leave out tasks of
	// a declared workflow; without one, the endpoint serves everything it declared.
	actions := sortedUnion(listed)

	if len(actions) == 0 {
		actions = sortedUnion(derived...)
	}

	// An endpoint without durable support is not registered for its durable tasks, whichever
	// form named them, so routing never picks it for one; a later healthcheck that gains the
	// support adds them back through the usual delta.
	durable := resp.GetDurable().GetSupported()

	if !durable && len(durableActions) > 0 {
		actions = enumerables.Filter(actions, func(action string) bool {
			return !enumerables.HasKey(durableActions, action)
		})
	}

	if limits.maxActions > 0 && len(actions) > limits.maxActions {
		return nil, fmt.Errorf("healthcheck advertises %d actions, more than the limit of %d", len(actions), limits.maxActions)
	}

	if err := actionRules.Validate(advertisedActions{Actions: actions}); err != nil {
		return nil, fmt.Errorf("healthcheck advertises an invalid action: %w", err)
	}

	if err := validateActionStorage(actions); err != nil {
		return nil, fmt.Errorf("healthcheck advertises an invalid action: %w", err)
	}

	streamActions, err := streamActionsOf(resp.Tasks, actions)

	if err != nil {
		return nil, err
	}

	return &healthcheckResult{
		workflows:      workflows,
		workflowHashes: workflowHashes,
		actions:        actions,
		streamActions:  streamActions,
		hash:           hashCatalog(workflowBytes, actions, streamActions, durable),
		durable:        durable,
		runtime:        resp.Runtime,
	}, nil
}

// durableActionsOf lists the actions of the workflow's tasks declared durable, the on-failure
// task included.
func durableActionsOf(wf *v1.CreateWorkflowVersionRequest) []string {
	tasks := append([]*v1.CreateTaskOpts{}, wf.Tasks...)

	if wf.OnFailureTask != nil {
		tasks = append(tasks, wf.OnFailureTask)
	}

	actions := make([]string, 0)

	for _, task := range tasks {
		if task.GetIsDurable() {
			actions = append(actions, task.Action)
		}
	}

	return actions
}

// hashCatalog is the digest of a parsed catalog: the canonical workflow encodings, the sorted
// action set, the sorted stream-action set and the durable support flag, each list framed as
// a tagged section with its element count and every element length-prefixed. The framing
// makes the digest a function of the lists themselves: no element content, newline or NUL
// included, can make two different catalogs encode alike, so a catalog whose only change is a
// stream flag always hashes differently from the one before it. The durable flag is its own
// section, so a change of support alone is a change of catalog and reaches the row.
func hashCatalog(workflows [][]byte, actions, streamActions []string, durable bool) string {
	hasher := sha256.New()
	flag := []byte{0}

	if durable {
		flag[0] = 1
	}

	sections := []struct {
		tag      byte
		elements [][]byte
	}{
		{tag: 'w', elements: workflows},
		{tag: 'a', elements: byteSlices(actions)},
		{tag: 's', elements: byteSlices(streamActions)},
		{tag: 'd', elements: [][]byte{flag}},
	}

	var prefix []byte

	for _, section := range sections {
		prefix = binary.AppendUvarint(prefix[:0], uint64(len(section.elements)))
		hasher.Write([]byte{section.tag})
		hasher.Write(prefix)

		for _, element := range section.elements {
			prefix = binary.AppendUvarint(prefix[:0], uint64(len(element)))
			hasher.Write(prefix)
			hasher.Write(element)
		}
	}

	return hex.EncodeToString(hasher.Sum(nil))
}

func byteSlices(values []string) [][]byte {
	out := make([][]byte, 0, len(values))

	for _, value := range values {
		out = append(out, []byte(value))
	}

	return out
}

// streamActionsOf normalizes the actions the task options flag with streams and checks each
// one against the catalog's action set: an option for an action the endpoint does not serve
// is a catalog error, so a misspelt id surfaces on the endpoint instead of being ignored.
func streamActionsOf(tasks []*v1.ServerlessTaskOptions, actions []string) ([]string, error) {
	served := enumerables.ToSet(actions, func(action string) string { return action })

	flagged := make([]string, 0)

	for _, task := range tasks {
		if task == nil || !task.GetStreams() {
			continue
		}

		normalized, err := normalizeAction(task.GetAction())

		if err != nil {
			return nil, fmt.Errorf("invalid task option action %q: %w", task.GetAction(), err)
		}

		if _, ok := served[normalized]; !ok {
			return nil, fmt.Errorf("task options name action %q, which the catalog does not serve", task.GetAction())
		}

		flagged = append(flagged, normalized)
	}

	return sortedUnion(flagged), nil
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
