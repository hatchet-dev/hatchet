# frozen_string_literal: true

require "hatchet-sdk"

HATCHET = Hatchet::Client.new(debug: true)

CONCURRENCY_CANCEL_NEWEST_WORKFLOW = HATCHET.workflow(
  name: "ConcurrencyCancelNewest",
  concurrency: Hatchet::ConcurrencyExpression.new(
    expression: "input.group",
    max_runs: 1,
    limit_strategy: :cancel_newest
  )
)

STEP1_CN = CONCURRENCY_CANCEL_NEWEST_WORKFLOW.task(:step1) do |input, ctx|
  50.times { sleep 0.10 }
end

CONCURRENCY_CANCEL_NEWEST_WORKFLOW.task(:step2, parents: [STEP1_CN]) do |input, ctx|
  50.times { sleep 0.10 }
end
