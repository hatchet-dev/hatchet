package v1_workflows

import (
	"strings"

	"github.com/hatchet-dev/hatchet/pkg/client/create"
	"github.com/hatchet-dev/hatchet/pkg/client/types"
	v1 "github.com/hatchet-dev/hatchet/pkg/v1"
	"github.com/hatchet-dev/hatchet/pkg/v1/factory"
	"github.com/hatchet-dev/hatchet/pkg/v1/features"
	"github.com/hatchet-dev/hatchet/pkg/v1/workflow"
	"github.com/hatchet-dev/hatchet/pkg/worker"
)

type RateLimitInput struct {
	UserId string `json:"userId"`
}

type RateLimitOutput struct {
	TransformedMessage string `json:"TransformedMessage"`
}

func upsertRateLimit(hatchet v1.HatchetClient) {
	// > Upsert Rate Limit
	hatchet.RateLimits().Upsert(
		features.CreateRatelimitOpts{
			Key:      "api-service-rate-limit",
			Limit:    10,
			Duration: types.Second,
		},
	)
}

// > Static Rate Limit
func StaticRateLimit(hatchet v1.HatchetClient) workflow.WorkflowDeclaration[RateLimitInput, RateLimitOutput] {
	// Create a standalone task that transforms a message

	// define the parameters for the rate limit
	rateLimitKey := "api-service-rate-limit"
	units := 1

	rateLimitTask := factory.NewTask(
		create.StandaloneTask{
			Name: "rate-limit-task",
			// 👀 add a static rate limit
			RateLimits: []*types.RateLimit{
				{
					Key:   rateLimitKey,
					Units: &units,
				},
			},
		},
		func(ctx worker.HatchetContext, input RateLimitInput) (*RateLimitOutput, error) {
			return &RateLimitOutput{
				TransformedMessage: strings.ToLower(input.UserId),
			}, nil
		},
		hatchet,
	)

	return rateLimitTask
}


// > Dynamic Rate Limit
func RateLimit(hatchet v1.HatchetClient) workflow.WorkflowDeclaration[RateLimitInput, RateLimitOutput] {
	// Create a standalone task that transforms a message

	// define the parameters for the rate limit
	expression := "input.userId"
	units := 1
	duration := types.Second

	rateLimitTask := factory.NewTask(
		create.StandaloneTask{
			Name: "rate-limit-task",
			// 👀 add a dynamic rate limit
			RateLimits: []*types.RateLimit{
				{
					KeyExpr:  &expression,
					Units:    &units,
					Duration: &duration,
				},
			},
		},
		func(ctx worker.HatchetContext, input RateLimitInput) (*RateLimitOutput, error) {
			return &RateLimitOutput{
				TransformedMessage: strings.ToLower(input.UserId),
			}, nil
		},
		hatchet,
	)

	return rateLimitTask
}
