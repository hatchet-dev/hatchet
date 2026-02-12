# frozen_string_literal: true

# > Create a workflow

require "hatchet-sdk"

HATCHET = Hatchet::Client.new(debug: true)

TASK_CONDITION_WORKFLOW = HATCHET.workflow(name: "TaskConditionWorkflow")

# !!

# > Add base task
COND_START = TASK_CONDITION_WORKFLOW.task(:start) do |input, ctx|
  { "random_number" => rand(1..100) }
end

# !!

# > Add wait for sleep
WAIT_FOR_SLEEP = TASK_CONDITION_WORKFLOW.task(
  :wait_for_sleep,
  parents: [COND_START],
  wait_for: [Hatchet::SleepCondition.new(10)]
) do |input, ctx|
  { "random_number" => rand(1..100) }
end

# !!

# > Add skip condition override
TASK_CONDITION_WORKFLOW.task(
  :skip_with_multiple_parents,
  parents: [COND_START, WAIT_FOR_SLEEP],
  skip_if: [Hatchet::ParentCondition.new(parent: COND_START, expression: "output.random_number > 0")]
) do |input, ctx|
  { "random_number" => rand(1..100) }
end

# !!

# > Add skip on event
SKIP_ON_EVENT = TASK_CONDITION_WORKFLOW.task(
  :skip_on_event,
  parents: [COND_START],
  wait_for: [Hatchet::SleepCondition.new(30)],
  skip_if: [Hatchet::UserEventCondition.new(event_key: "skip_on_event:skip")]
) do |input, ctx|
  { "random_number" => rand(1..100) }
end

# !!

# > Add branching
LEFT_BRANCH = TASK_CONDITION_WORKFLOW.task(
  :left_branch,
  parents: [WAIT_FOR_SLEEP],
  skip_if: [
    Hatchet::ParentCondition.new(
      parent: WAIT_FOR_SLEEP,
      expression: "output.random_number > 50"
    )
  ]
) do |input, ctx|
  { "random_number" => rand(1..100) }
end

RIGHT_BRANCH = TASK_CONDITION_WORKFLOW.task(
  :right_branch,
  parents: [WAIT_FOR_SLEEP],
  skip_if: [
    Hatchet::ParentCondition.new(
      parent: WAIT_FOR_SLEEP,
      expression: "output.random_number <= 50"
    )
  ]
) do |input, ctx|
  { "random_number" => rand(1..100) }
end

# !!

# > Add wait for event
WAIT_FOR_EVENT = TASK_CONDITION_WORKFLOW.task(
  :wait_for_event,
  parents: [COND_START],
  wait_for: [
    Hatchet.or_(
      Hatchet::SleepCondition.new(60),
      Hatchet::UserEventCondition.new(event_key: "wait_for_event:start")
    )
  ]
) do |input, ctx|
  { "random_number" => rand(1..100) }
end

# !!

# > Add sum
TASK_CONDITION_WORKFLOW.task(
  :sum,
  parents: [COND_START, WAIT_FOR_SLEEP, WAIT_FOR_EVENT, SKIP_ON_EVENT, LEFT_BRANCH, RIGHT_BRANCH]
) do |input, ctx|
  one = ctx.task_output(COND_START)["random_number"]
  two = ctx.task_output(WAIT_FOR_EVENT)["random_number"]
  three = ctx.task_output(WAIT_FOR_SLEEP)["random_number"]
  four = ctx.was_skipped?(SKIP_ON_EVENT) ? 0 : ctx.task_output(SKIP_ON_EVENT)["random_number"]
  five = ctx.was_skipped?(LEFT_BRANCH) ? 0 : ctx.task_output(LEFT_BRANCH)["random_number"]
  six = ctx.was_skipped?(RIGHT_BRANCH) ? 0 : ctx.task_output(RIGHT_BRANCH)["random_number"]

  { "sum" => one + two + three + four + five + six }
end

# !!

def main
  worker = HATCHET.worker("dag-worker", workflows: [TASK_CONDITION_WORKFLOW])
  worker.start
end

main if __FILE__ == $PROGRAM_NAME
