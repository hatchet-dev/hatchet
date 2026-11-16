# frozen_string_literal: true

require "hatchet-sdk"

HATCHET = Hatchet::Client.new(debug: true) unless defined?(HATCHET)

# > Cancel Queued Except Newest
CONCURRENCY_CANCEL_QUEUED_EXCEPT_NEWEST_WORKFLOW = HATCHET.workflow(
  name: "ConcurrencyCancelQueuedExceptNewest",
  concurrency: Hatchet::ConcurrencyExpression.new(
    expression: "input.group",
    max_runs: 1,
    limit_strategy: :cancel_queued_except_newest
  )
)
# !!

STEP1_CEN = CONCURRENCY_CANCEL_QUEUED_EXCEPT_NEWEST_WORKFLOW.task(:step1) do |input, ctx|
  50.times { sleep 0.10 }
end

CONCURRENCY_CANCEL_QUEUED_EXCEPT_NEWEST_WORKFLOW.task(:step2, parents: [STEP1_CEN]) do |input, ctx|
  50.times { sleep 0.10 }
end
