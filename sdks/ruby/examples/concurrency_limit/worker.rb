# frozen_string_literal: true

require "hatchet-sdk"

HATCHET = Hatchet::Client.new(debug: true)

# > Workflow
CONCURRENCY_LIMIT_WORKFLOW = HATCHET.workflow(
  name: "ConcurrencyDemoWorkflow",
  concurrency: Hatchet::ConcurrencyExpression.new(
    expression: "input.group_key",
    max_runs: 5,
    limit_strategy: :cancel_in_progress
  )
)

CONCURRENCY_LIMIT_WORKFLOW.task(:step1) do |input, ctx|
  sleep 3
  puts "executed step1"
  { "run" => input["run"] }
end

# !!

def main
  worker = HATCHET.worker(
    "concurrency-demo-worker", slots: 10, workflows: [CONCURRENCY_LIMIT_WORKFLOW]
  )
  worker.start
end

main if __FILE__ == $PROGRAM_NAME
