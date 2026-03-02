# frozen_string_literal: true

require "hatchet-sdk"

HATCHET = Hatchet::Client.new(debug: true) unless defined?(HATCHET)

SLEEP_TIME_WL = 2
DIGIT_MAX_RUNS_WL = 8
NAME_MAX_RUNS_WL = 3

# > Multiple Concurrency Keys
CONCURRENCY_WORKFLOW_LEVEL_WORKFLOW = HATCHET.workflow(
  name: "ConcurrencyWorkflowLevel",
  concurrency: [
    Hatchet::ConcurrencyExpression.new(
      expression: "input.digit",
      max_runs: DIGIT_MAX_RUNS_WL,
      limit_strategy: :group_round_robin
    ),
    Hatchet::ConcurrencyExpression.new(
      expression: "input.name",
      max_runs: NAME_MAX_RUNS_WL,
      limit_strategy: :group_round_robin
    )
  ]
)

CONCURRENCY_WORKFLOW_LEVEL_WORKFLOW.task(:task_1) do |input, ctx|
  sleep SLEEP_TIME_WL
end

CONCURRENCY_WORKFLOW_LEVEL_WORKFLOW.task(:task_2) do |input, ctx|
  sleep SLEEP_TIME_WL
end


def main
  worker = HATCHET.worker(
    "concurrency-worker-workflow-level",
    slots: 10,
    workflows: [CONCURRENCY_WORKFLOW_LEVEL_WORKFLOW]
  )
  worker.start
end

main if __FILE__ == $PROGRAM_NAME
