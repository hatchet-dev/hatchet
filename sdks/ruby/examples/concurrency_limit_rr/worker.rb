# frozen_string_literal: true

require "hatchet-sdk"

HATCHET = Hatchet::Client.new(debug: true)

# > Concurrency Strategy With Key
CONCURRENCY_LIMIT_RR_WORKFLOW = HATCHET.workflow(
  name: "ConcurrencyDemoWorkflowRR",
  concurrency: Hatchet::ConcurrencyExpression.new(
    expression: "input.group",
    max_runs: 1,
    limit_strategy: :group_round_robin
  )
)

CONCURRENCY_LIMIT_RR_WORKFLOW.task(:step1) do |input, ctx|
  puts "starting step1"
  sleep 2
  puts "finished step1"
end

# !!

def main
  worker = HATCHET.worker(
    "concurrency-demo-worker-rr",
    slots: 10,
    workflows: [CONCURRENCY_LIMIT_RR_WORKFLOW]
  )
  worker.start
end

main if __FILE__ == $PROGRAM_NAME
