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

	Event       Resource = "event"
	WorkflowRun Resource = "workflow-run"
	TaskRun     Resource = "task-run"
	Worker      Resource = "worker"
	RateLimit   Resource = "rate-limit"
	Webhook     Resource = "webhook"
	Log         Resource = "log"
	StreamEvent Resource = "stream-event"
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

type Properties map[string]interface{}

type Source string

const (
	SourceUI   Source = "ui"
	SourceAPI  Source = "api"
	SourceGRPC Source = "grpc"
	SourceCLI  Source = "cli"
)

type contextKey string

const (
	APITokenIDKey     = contextKey("api_token_id")
	TenantIDKey       = contextKey("tenant_id")
	OrganizationIDKey = contextKey("organization_id")
	UserIDKey         = contextKey("user_id")
	SourceKey         = contextKey("source")

	SourceMetadataKey = "x-hatchet-source"
)

type Analytics interface {
	Enqueue(ctx context.Context, resource Resource, action Action, resourceId string, properties Properties)
	Count(ctx context.Context, resource Resource, action Action, props ...Properties)
	Identify(userId uuid.UUID, properties Properties)
	Tenant(tenantId uuid.UUID, data Properties)
	Group(groupType string, groupKey string, data Properties)
	Close() error
}

func TokenIDFromContext(ctx context.Context) *uuid.UUID {
	if id, ok := ctx.Value(APITokenIDKey).(uuid.UUID); ok && id != uuid.Nil {
		return &id
	}
	return nil
}

func TenantIDFromContext(ctx context.Context) *uuid.UUID {
	if id, ok := ctx.Value(TenantIDKey).(uuid.UUID); ok && id != uuid.Nil {
		return &id
	}
	return nil
}

func OrganizationIDFromContext(ctx context.Context) *uuid.UUID {
	if id, ok := ctx.Value(OrganizationIDKey).(uuid.UUID); ok && id != uuid.Nil {
		return &id
	}
	return nil
}

func UserIDFromContext(ctx context.Context) *uuid.UUID {
	if id, ok := ctx.Value(UserIDKey).(uuid.UUID); ok && id != uuid.Nil {
		return &id
	}
	return nil
}

func SourceFromContext(ctx context.Context) Source {
	if s, ok := ctx.Value(SourceKey).(Source); ok {
		return s
	}
	return ""
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
func Props(kvs ...interface{}) Properties {
	if len(kvs) == 0 || len(kvs)%2 != 0 {
		return nil
	}

	var m Properties
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
			m = make(Properties, len(kvs)/2)
		}
		m[k] = v
	}
	return m
}

type NoOpAnalytics struct{}

func (a NoOpAnalytics) Enqueue(ctx context.Context, resource Resource, action Action, resourceId string, properties Properties) {
}

func (a NoOpAnalytics) Count(ctx context.Context, resource Resource, action Action, props ...Properties) {
}

func (a NoOpAnalytics) Identify(userId uuid.UUID, properties Properties) {}

func (a NoOpAnalytics) Tenant(tenantId uuid.UUID, data Properties) {}

func (a NoOpAnalytics) Group(groupType string, groupKey string, data Properties) {}

func (a NoOpAnalytics) Close() error { return nil }
