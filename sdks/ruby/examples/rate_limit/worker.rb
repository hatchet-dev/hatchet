# frozen_string_literal: true

require "hatchet-sdk"

HATCHET = Hatchet::Client.new(debug: true)

# > Workflow
RATE_LIMIT_WORKFLOW = HATCHET.workflow(name: "RateLimitWorkflow")

# !!

# > Static
RATE_LIMIT_KEY = "test-limit"

RATE_LIMIT_WORKFLOW.task(
  :step_1,
  rate_limits: [Hatchet::RateLimit.new(static_key: RATE_LIMIT_KEY, units: 1)]
) do |input, ctx|
  puts "executed step_1"
end

# !!

# > Dynamic
RATE_LIMIT_WORKFLOW.task(
  :step_2,
  rate_limits: [
    Hatchet::RateLimit.new(
      dynamic_key: "input.user_id",
      units: 1,
      limit: 10,
      duration: :minute
    )
  ]
) do |input, ctx|
  puts "executed step_2"
end

# !!

# > Create a rate limit
def main
  HATCHET.rate_limits.put(RATE_LIMIT_KEY, 2, :second)

  worker = HATCHET.worker(
    "rate-limit-worker", slots: 10, workflows: [RATE_LIMIT_WORKFLOW]
  )
  worker.start
end

# !!

main if __FILE__ == $PROGRAM_NAME
