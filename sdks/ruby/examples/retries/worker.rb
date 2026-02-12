# frozen_string_literal: true

require "hatchet-sdk"

HATCHET = Hatchet::Client.new(debug: true)

SIMPLE_RETRY_WORKFLOW = hatchet.workflow(name: "SimpleRetryWorkflow")
BACKOFF_WORKFLOW = hatchet.workflow(name: "BackoffWorkflow")

# > Simple Step Retries
SIMPLE_RETRY_WORKFLOW.task(:always_fail, retries: 3) do |input, ctx|
  raise "simple task failed"
end

# > Retries with Count
SIMPLE_RETRY_WORKFLOW.task(:fail_twice, retries: 3) do |input, ctx|
  raise "simple task failed" if ctx.retry_count < 2

  { "status" => "success" }
end

# > Retries with Backoff
BACKOFF_WORKFLOW.task(
  :backoff_task,
  retries: 10,
  # Maximum number of seconds to wait between retries
  backoff_max_seconds: 10,
  # Factor to increase the wait time between retries.
  # This sequence will be 2s, 4s, 8s, 10s, 10s, 10s... due to the maxSeconds limit
  backoff_factor: 2.0
) do |input, ctx|
  raise "backoff task failed" if ctx.retry_count < 3

  { "status" => "success" }
end

def main
  worker = hatchet.worker("backoff-worker", slots: 4, workflows: [BACKOFF_WORKFLOW])
  worker.start
end

main if __FILE__ == $PROGRAM_NAME
