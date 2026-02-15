# frozen_string_literal: true

require "hatchet-sdk"

HATCHET = Hatchet::Client.new(debug: true) unless defined?(HATCHET)

# > StickyWorker
STICKY_WORKFLOW = HATCHET.workflow(
  name: "StickyWorkflow",
  # Specify a sticky strategy when declaring the workflow
  sticky: :soft
)

STEP1A = STICKY_WORKFLOW.task(:step1a) do |input, ctx|
  { "worker" => ctx.worker.id }
end

STEP1B = STICKY_WORKFLOW.task(:step1b) do |input, ctx|
  { "worker" => ctx.worker.id }
end


# > StickyChild
STICKY_CHILD_WORKFLOW = HATCHET.workflow(
  name: "StickyChildWorkflow",
  sticky: :soft
)

STICKY_WORKFLOW.task(:step2, parents: [STEP1A, STEP1B]) do |input, ctx|
  ref = STICKY_CHILD_WORKFLOW.run_no_wait(
    options: Hatchet::TriggerWorkflowOptions.new(sticky: true)
  )

  ref.result

  { "worker" => ctx.worker.id }
end

STICKY_CHILD_WORKFLOW.task(:child) do |input, ctx|
  { "worker" => ctx.worker.id }
end


def main
  worker = HATCHET.worker(
    "sticky-worker", slots: 10, workflows: [STICKY_WORKFLOW, STICKY_CHILD_WORKFLOW]
  )
  worker.start
end

main if __FILE__ == $PROGRAM_NAME
