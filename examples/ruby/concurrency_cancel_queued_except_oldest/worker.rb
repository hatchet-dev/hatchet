# frozen_string_literal: true

require "hatchet-sdk"

HATCHET = Hatchet::Client.new(debug: true) unless defined?(HATCHET)

# > Cancel Queued Except Oldest
CONCURRENCY_CANCEL_QUEUED_EXCEPT_OLDEST_WORKFLOW = HATCHET.workflow(
  name: "ConcurrencyCancelQueuedExceptOldest",
  concurrency: Hatchet::ConcurrencyExpression.new(
    expression: "input.group",
    max_runs: 1,
    limit_strategy: :cancel_queued_except_oldest
  )
)

STEP1_CEO = CONCURRENCY_CANCEL_QUEUED_EXCEPT_OLDEST_WORKFLOW.task(:step1) do |input, ctx|
  50.times { sleep 0.10 }
end

CONCURRENCY_CANCEL_QUEUED_EXCEPT_OLDEST_WORKFLOW.task(:step2, parents: [STEP1_CEO]) do |input, ctx|
  50.times { sleep 0.10 }
end
