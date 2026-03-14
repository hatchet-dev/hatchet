# frozen_string_literal: true

require "hatchet-sdk"

HATCHET = Hatchet::Client.new(debug: true) unless defined?(HATCHET)

CONCURRENCY_CANCEL_NEWEST_WORKFLOW = HATCHET.workflow(
  name: "ConcurrencyCancelNewest",
  concurrency: Hatchet::ConcurrencyExpression.new(
    expression: "input.group",
    max_runs: 1,
    limit_strategy: :cancel_newest,
  ),
)

STEP1_CN = CONCURRENCY_CANCEL_NEWEST_WORKFLOW.task(:step1) do |_input, _ctx|
  50.times { sleep 0.10 }
end

CONCURRENCY_CANCEL_NEWEST_WORKFLOW.task(:step2, parents: [STEP1_CN]) do |_input, _ctx|
  50.times { sleep 0.10 }
end
