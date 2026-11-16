# frozen_string_literal: true

require "hatchet-sdk"

HATCHET = Hatchet::Client.new(debug: true) unless defined?(HATCHET)

# > Dynamic Max Runs
# max_runs accepts an Integer or a CEL expression String. With an expression, each
# concurrency group's limit is computed from the task's input.
CONCURRENCY_DYNAMIC_WORKFLOW = HATCHET.workflow(
  name: "ConcurrencyDynamicWorkflow",
  concurrency: Hatchet::ConcurrencyExpression.new(
    expression: "input.account",
    max_runs: "input.tier == 'premium' ? 10 : 1",
    limit_strategy: :group_round_robin
  )
)

CONCURRENCY_DYNAMIC_WORKFLOW.task(:step1) do |input, ctx|
  puts "running for account #{input["account"]}"
  sleep 2
end


def main
  worker = HATCHET.worker(
    "concurrency-dynamic-worker",
    slots: 10,
    workflows: [CONCURRENCY_DYNAMIC_WORKFLOW]
  )
  worker.start
end

main if __FILE__ == $PROGRAM_NAME
