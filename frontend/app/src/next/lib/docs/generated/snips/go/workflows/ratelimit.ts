import { Snippet } from '@/next/lib/docs/generated/snips/types';

const snippet: Snippet = {
  language: 'go',
  content:
    'package v1_workflows\n\nimport (\n\t"strings"\n\n\t"github.com/hatchet-dev/hatchet/pkg/client/create"\n\t"github.com/hatchet-dev/hatchet/pkg/client/types"\n\tv1 "github.com/hatchet-dev/hatchet/pkg/v1"\n\t"github.com/hatchet-dev/hatchet/pkg/v1/factory"\n\t"github.com/hatchet-dev/hatchet/pkg/v1/features"\n\t"github.com/hatchet-dev/hatchet/pkg/v1/workflow"\n\t"github.com/hatchet-dev/hatchet/pkg/worker"\n)\n\ntype RateLimitInput struct {\n\tUserId string `json:"userId"`\n}\n\ntype RateLimitOutput struct {\n\tTransformedMessage string `json:"TransformedMessage"`\n}\n\nfunc upsertRateLimit(hatchet v1.HatchetClient) {\n\t// > Upsert Rate Limit\n\thatchet.RateLimits().Upsert(\n\t\tfeatures.CreateRatelimitOpts{\n\t\t\tKey:      "api-service-rate-limit",\n\t\t\tLimit:    10,\n\t\t\tDuration: types.Second,\n\t\t},\n\t)\n}\n\n// > Static Rate Limit\nfunc StaticRateLimit(hatchet v1.HatchetClient) workflow.WorkflowDeclaration[RateLimitInput, RateLimitOutput] {\n\t// Create a standalone task that transforms a message\n\n\t// define the parameters for the rate limit\n\trateLimitKey := "api-service-rate-limit"\n\tunits := 1\n\n\trateLimitTask := factory.NewTask(\n\t\tcreate.StandaloneTask{\n\t\t\tName: "rate-limit-task",\n\t\t\t// 👀 add a static rate limit\n\t\t\tRateLimits: []*types.RateLimit{\n\t\t\t\t{\n\t\t\t\t\tKey:   rateLimitKey,\n\t\t\t\t\tUnits: &units,\n\t\t\t\t},\n\t\t\t},\n\t\t},\n\t\tfunc(ctx worker.HatchetContext, input RateLimitInput) (*RateLimitOutput, error) {\n\t\t\treturn &RateLimitOutput{\n\t\t\t\tTransformedMessage: strings.ToLower(input.UserId),\n\t\t\t}, nil\n\t\t},\n\t\thatchet,\n\t)\n\n\treturn rateLimitTask\n}\n\n\n// > Dynamic Rate Limit\nfunc RateLimit(hatchet v1.HatchetClient) workflow.WorkflowDeclaration[RateLimitInput, RateLimitOutput] {\n\t// Create a standalone task that transforms a message\n\n\t// define the parameters for the rate limit\n\texpression := "input.userId"\n\tunits := 1\n\tduration := types.Second\n\n\trateLimitTask := factory.NewTask(\n\t\tcreate.StandaloneTask{\n\t\t\tName: "rate-limit-task",\n\t\t\t// 👀 add a dynamic rate limit\n\t\t\tRateLimits: []*types.RateLimit{\n\t\t\t\t{\n\t\t\t\t\tKeyExpr:  &expression,\n\t\t\t\t\tUnits:    &units,\n\t\t\t\t\tDuration: &duration,\n\t\t\t\t},\n\t\t\t},\n\t\t},\n\t\tfunc(ctx worker.HatchetContext, input RateLimitInput) (*RateLimitOutput, error) {\n\t\t\treturn &RateLimitOutput{\n\t\t\t\tTransformedMessage: strings.ToLower(input.UserId),\n\t\t\t}, nil\n\t\t},\n\t\thatchet,\n\t)\n\n\treturn rateLimitTask\n}\n\n',
  source: 'out/go/workflows/ratelimit.go',
  blocks: {
    upsert_rate_limit: {
      start: 25,
      stop: 31,
    },
    static_rate_limit: {
      start: 35,
      stop: 63,
    },
    dynamic_rate_limit: {
      start: 66,
      stop: 96,
    },
  },
  highlights: {},
};

export default snippet;
