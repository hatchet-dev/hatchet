# frozen_string_literal: true

require "hatchet-sdk"

HATCHET = Hatchet::Client.new(debug: true) unless defined?(HATCHET)

SLEEP_TIME_MK = 2
DIGIT_MAX_RUNS = 8
NAME_MAX_RUNS = 3

# > Concurrency Strategy With Key
CONCURRENCY_MULTIPLE_KEYS_WORKFLOW = HATCHET.workflow(
  name: "ConcurrencyWorkflowManyKeys"
)

CONCURRENCY_MULTIPLE_KEYS_WORKFLOW.task(
  :concurrency_task,
  concurrency: [
    Hatchet::ConcurrencyExpression.new(
      expression: "input.digit",
      max_runs: DIGIT_MAX_RUNS,
      limit_strategy: :group_round_robin
    ),
    Hatchet::ConcurrencyExpression.new(
      expression: "input.name",
      max_runs: NAME_MAX_RUNS,
      limit_strategy: :group_round_robin
    )
  ]
) do |input, ctx|
  sleep SLEEP_TIME_MK
end


def main
  worker = HATCHET.worker(
    "concurrency-worker-multiple-keys",
    slots: 10,
    workflows: [CONCURRENCY_MULTIPLE_KEYS_WORKFLOW]
  )
  worker.start
end

main if __FILE__ == $PROGRAM_NAME
