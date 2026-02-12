# frozen_string_literal: true

require "hatchet-sdk"

HATCHET = Hatchet::Client.new(debug: true)

# > Create a durable workflow
DURABLE_WORKFLOW = HATCHET.workflow(name: "DurableWorkflow")
EPHEMERAL_WORKFLOW = HATCHET.workflow(name: "EphemeralWorkflow")

# > Add durable task
DURABLE_EVENT_KEY = "durable-example:event"
DURABLE_SLEEP_TIME = 5

DURABLE_WORKFLOW.task(:ephemeral_task) do |input, ctx|
  puts "Running non-durable task"
end

DURABLE_WORKFLOW.durable_task(:durable_task) do |input, ctx|
  puts "Waiting for sleep"
  ctx.sleep_for(duration: DURABLE_SLEEP_TIME)
  puts "Sleep finished"

  puts "Waiting for event"
  ctx.wait_for(
    "event",
    Hatchet::UserEventCondition.new(event_key: DURABLE_EVENT_KEY, expression: "true")
  )
  puts "Event received"

  { "status" => "success" }
end

# > Add durable tasks that wait for or groups
DURABLE_WORKFLOW.durable_task(:wait_for_or_group_1) do |input, ctx|
  start = Time.now
  wait_result = ctx.wait_for(
    SecureRandom.hex(16),
    Hatchet.or_(
      Hatchet::SleepCondition.new(DURABLE_SLEEP_TIME),
      Hatchet::UserEventCondition.new(event_key: DURABLE_EVENT_KEY)
    )
  )

  key = wait_result.keys.first
  event_id = wait_result[key].keys.first

  {
    "runtime" => (Time.now - start).to_i,
    "key" => key,
    "event_id" => event_id
  }
end

DURABLE_WORKFLOW.durable_task(:wait_for_or_group_2) do |input, ctx|
  start = Time.now
  wait_result = ctx.wait_for(
    SecureRandom.hex(16),
    Hatchet.or_(
      Hatchet::SleepCondition.new(6 * DURABLE_SLEEP_TIME),
      Hatchet::UserEventCondition.new(event_key: DURABLE_EVENT_KEY)
    )
  )

  key = wait_result.keys.first
  event_id = wait_result[key].keys.first

  {
    "runtime" => (Time.now - start).to_i,
    "key" => key,
    "event_id" => event_id
  }
end

DURABLE_WORKFLOW.durable_task(:wait_for_multi_sleep) do |input, ctx|
  start = Time.now

  3.times do
    ctx.sleep_for(duration: DURABLE_SLEEP_TIME)
  end

  { "runtime" => (Time.now - start).to_i }
end

EPHEMERAL_WORKFLOW.task(:ephemeral_task_2) do |input, ctx|
  puts "Running non-durable task"
end

WAIT_FOR_SLEEP_TWICE = HATCHET.durable_task(name: "wait_for_sleep_twice") do |input, ctx|
  start = Time.now

  ctx.sleep_for(duration: DURABLE_SLEEP_TIME)

  { "runtime" => (Time.now - start).to_i }
end

def main
  worker = HATCHET.worker(
    "durable-worker",
    workflows: [DURABLE_WORKFLOW, EPHEMERAL_WORKFLOW, WAIT_FOR_SLEEP_TWICE]
  )
  worker.start
end

main if __FILE__ == $PROGRAM_NAME
