# frozen_string_literal: true

require "hatchet-sdk"

HATCHET = Hatchet::Client.new(debug: true) unless defined?(HATCHET)

CONCURRENCY_CANCEL_IN_PROGRESS_WORKFLOW = HATCHET.workflow(
  name: "ConcurrencyCancelInProgress",
  concurrency: Hatchet::ConcurrencyExpression.new(
    expression: "input.group",
    max_runs: 1,
    limit_strategy: :cancel_in_progress
  )
)

STEP1_CIP = CONCURRENCY_CANCEL_IN_PROGRESS_WORKFLOW.task(:step1) do |input, ctx|
  50.times { sleep 0.10 }
end

CONCURRENCY_CANCEL_IN_PROGRESS_WORKFLOW.task(:step2, parents: [STEP1_CIP]) do |input, ctx|
  50.times { sleep 0.10 }
end
