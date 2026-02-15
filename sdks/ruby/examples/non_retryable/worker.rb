# frozen_string_literal: true

require "hatchet-sdk"

HATCHET = Hatchet::Client.new(debug: true) unless defined?(HATCHET)

NON_RETRYABLE_WORKFLOW = HATCHET.workflow(name: "NonRetryableWorkflow")

# > Non-retryable task
NON_RETRYABLE_WORKFLOW.task(:should_not_retry, retries: 1) do |input, ctx|
  raise Hatchet::NonRetryableError, "This task should not retry"
end

NON_RETRYABLE_WORKFLOW.task(:should_retry_wrong_exception_type, retries: 1) do |input, ctx|
  raise TypeError, "This task should retry because it's not a NonRetryableError"
end

NON_RETRYABLE_WORKFLOW.task(:should_not_retry_successful_task, retries: 1) do |input, ctx|
  # no-op
end

# !!

def main
  worker = HATCHET.worker("non-retry-worker", workflows: [NON_RETRYABLE_WORKFLOW])
  worker.start
end

main if __FILE__ == $PROGRAM_NAME
