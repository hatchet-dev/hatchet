package analytics

import (
	"context"

	"github.com/google/uuid"
)

type Resource string

const (
	User     Resource = "user"
	Invite   Resource = "user-invite"
	Tenant   Resource = "tenant"
	Token    Resource = "api-token"
	Workflow Resource = "workflow"

	Event         Resource = "event"
	WorkflowRun   Resource = "workflow-run"
	TaskRun       Resource = "task-run"
	Worker        Resource = "worker"
	RateLimit     Resource = "rate-limit"
	WebhookWorker Resource = "webhook-worker"
	Log           Resource = "log"
	StreamEvent   Resource = "stream-event"
)

type Action string

const (
	Create Action = "create"
	Revoke Action = "revoke"
	Accept Action = "accept"
	Reject Action = "reject"

	Login     Action = "login"
	Delete    Action = "delete"
	Cancel    Action = "cancel"
	Replay    Action = "replay"
	List      Action = "list"
	Get       Action = "get"
	Register  Action = "register"
	Subscribe Action = "subscribe"
	Listen    Action = "listen"
	Release   Action = "release"
	Refresh   Action = "refresh"
	Send      Action = "send"
)

type contextKey string

const APITokenIDKey = contextKey("api_token_id")

type Analytics interface {
	Enqueue(ctx context.Context, resource Resource, action Action, userID *uuid.UUID, tenantId *uuid.UUID, resourceId string, properties map[string]interface{})
	Count(ctx context.Context, resource Resource, action Action, tenantID uuid.UUID, props ...map[string]interface{})
	Identify(userId uuid.UUID, properties map[string]interface{})
	Tenant(tenantId uuid.UUID, data map[string]interface{})
	Close() error
}

func TokenIDFromContext(ctx context.Context) *uuid.UUID {
	if id, ok := ctx.Value(APITokenIDKey).(uuid.UUID); ok && id != uuid.Nil {
		return &id
	}
	return nil
}

func DistinctID(userID *uuid.UUID, tokenID *uuid.UUID, tenantID *uuid.UUID) string {
	if userID != nil {
		return "$user_" + userID.String()
	}
	if tokenID != nil {
		return "$token_" + tokenID.String()
	}
	if tenantID != nil {
		return "$tenant_" + tenantID.String()
	}
	return ""
}

// Props builds a property map from variadic key-value pairs, keeping all
// non-nil values regardless of type. Keys must be strings; non-string keys
// are skipped. Nil values are omitted.
func Props(kvs ...interface{}) map[string]interface{} {
	if len(kvs) == 0 || len(kvs)%2 != 0 {
		return nil
	}

	var m map[string]interface{}
	for i := 0; i < len(kvs); i += 2 {
		k, ok := kvs[i].(string)
		if !ok {
			continue
		}
		v := kvs[i+1]
		if v == nil {
			continue
		}
		if m == nil {
			m = make(map[string]interface{}, len(kvs)/2)
		}
		m[k] = v
	}
	return m
}

type NoOpAnalytics struct{}

func (a NoOpAnalytics) Enqueue(ctx context.Context, resource Resource, action Action, userID *uuid.UUID, tenantId *uuid.UUID, resourceId string, properties map[string]interface{}) {
}

func (a NoOpAnalytics) Count(ctx context.Context, resource Resource, action Action, tenantID uuid.UUID, props ...map[string]interface{}) {
}

func (a NoOpAnalytics) Identify(userId uuid.UUID, properties map[string]interface{}) {}

func (a NoOpAnalytics) Tenant(tenantId uuid.UUID, data map[string]interface{}) {}

func (a NoOpAnalytics) Close() error { return nil }
