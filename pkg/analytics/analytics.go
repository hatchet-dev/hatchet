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
)

type Action string

const (
	Create Action = "create"
	Revoke Action = "revoke"
	Accept Action = "accept"
	Reject Action = "reject"
)

type contextKey string

const APITokenIDKey = contextKey("api_token_id")

type Analytics interface {
	Enqueue(ctx context.Context, resource Resource, action Action, userID *uuid.UUID, tenantId *uuid.UUID, resourceId string, properties map[string]interface{})
	Identify(userId uuid.UUID, properties map[string]interface{})
	Tenant(tenantId uuid.UUID, data map[string]interface{})
}

func TokenIDFromContext(ctx context.Context) *uuid.UUID {
	if id, ok := ctx.Value(APITokenIDKey).(uuid.UUID); ok && id != uuid.Nil {
		return &id
	}
	return nil
}

func DistinctID(userID *uuid.UUID, tokenID *uuid.UUID) string {
	if userID != nil {
		return "$user_" + userID.String()
	}
	if tokenID != nil {
		return "$token_" + tokenID.String()
	}
	return ""
}

type NoOpAnalytics struct{}

func (a NoOpAnalytics) Enqueue(ctx context.Context, resource Resource, action Action, userID *uuid.UUID, tenantId *uuid.UUID, resourceId string, properties map[string]interface{}) {
}

func (a NoOpAnalytics) Identify(userId uuid.UUID, properties map[string]interface{}) {}

func (a NoOpAnalytics) Tenant(tenantId uuid.UUID, data map[string]interface{}) {}
