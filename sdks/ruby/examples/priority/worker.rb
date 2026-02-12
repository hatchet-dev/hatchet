# frozen_string_literal: true

require "hatchet-sdk"

HATCHET = Hatchet::Client.new(debug: true)

# > Default priority
DEFAULT_PRIORITY = 1
SLEEP_TIME = 0.25

PRIORITY_WORKFLOW = HATCHET.workflow(
  name: "PriorityWorkflow",
  default_priority: DEFAULT_PRIORITY
)

PRIORITY_WORKFLOW.task(:priority_task) do |input, ctx|
  puts "Priority: #{ctx.priority}"
  sleep SLEEP_TIME
end

# !!

def main
  worker = HATCHET.worker(
    "priority-worker",
    slots: 1,
    workflows: [PRIORITY_WORKFLOW]
  )
  worker.start
end

main if __FILE__ == $PROGRAM_NAME
