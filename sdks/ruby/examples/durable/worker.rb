# frozen_string_literal: true

require 'hatchet-sdk'

HATCHET = Hatchet::Client.new(debug: true) unless defined?(HATCHET)

# > Create a durable workflow
DURABLE_WORKFLOW = HATCHET.workflow(name: 'DurableWorkflow')
EPHEMERAL_WORKFLOW = HATCHET.workflow(name: 'EphemeralWorkflow')

# !!

# > Add durable task
DURABLE_EVENT_KEY = 'durable-example:event'
DURABLE_SLEEP_TIME = 5

DURABLE_WORKFLOW.task(:ephemeral_task) do |_input, _ctx|
  puts 'Running non-durable task'
end

DURABLE_WORKFLOW.durable_task(:durable_task, execution_timeout: 60) do |_input, ctx|
  puts 'Waiting for sleep'
  ctx.sleep_for(duration: DURABLE_SLEEP_TIME)
  puts 'Sleep finished'

  puts 'Waiting for event'
  ctx.wait_for(
    'event',
    Hatchet::UserEventCondition.new(event_key: DURABLE_EVENT_KEY, expression: 'true')
  )
  puts 'Event received'

  { 'status' => 'success' }
end

# !!

# > Add durable tasks that wait for or groups
DURABLE_WORKFLOW.durable_task(:wait_for_or_group_1, execution_timeout: 60) do |_input, ctx|
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
    'runtime' => (Time.now - start).to_i,
    'key' => key,
    'event_id' => event_id
  }
end

DURABLE_WORKFLOW.durable_task(:wait_for_or_group_2, execution_timeout: 120) do |_input, ctx|
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
    'runtime' => (Time.now - start).to_i,
    'key' => key,
    'event_id' => event_id
  }
end

DURABLE_WORKFLOW.durable_task(:wait_for_multi_sleep, execution_timeout: 120) do |_input, ctx|
  start = Time.now

  3.times do
    ctx.sleep_for(duration: DURABLE_SLEEP_TIME)
  end

  { 'runtime' => (Time.now - start).to_i }
end

EPHEMERAL_WORKFLOW.task(:ephemeral_task_2) do |_input, _ctx|
  puts 'Running non-durable task'
end

WAIT_FOR_SLEEP_TWICE = HATCHET.durable_task(name: 'wait_for_sleep_twice', execution_timeout: 60) do |_input, ctx|
  start = Time.now

  ctx.sleep_for(duration: DURABLE_SLEEP_TIME)

  { 'runtime' => (Time.now - start).to_i }
end

# !!

ERROR_RAISING_TASK = HATCHET.task(name: 'error-raising-task') do |input, _ctx|
  raise input['error_message']
end

ERROR_RAISING_DURABLE_PARENT = HATCHET.durable_task(name: 'error-raising-durable-parent',
                                                    execution_timeout: 30) do |input, ctx|
  ref = ERROR_RAISING_TASK.run_no_wait(input)

  child_raised = false
  child_error_str = nil

  begin
    ref.result
  rescue StandardError => e
    child_raised = true
    child_error_str = e.message
  end

  {
    'child_raised' => child_raised,
    'child_error_str' => child_error_str,
    'child_run_external_id' => ref.workflow_run_id,
    'parent_run_external_id' => ctx.workflow_run_id
  }
end

def main
  workflows = [DURABLE_WORKFLOW, EPHEMERAL_WORKFLOW, WAIT_FOR_SLEEP_TWICE, ERROR_RAISING_TASK,
               ERROR_RAISING_DURABLE_PARENT]
  worker = HATCHET.worker('durable-worker', workflows: workflows)
  worker.start
end

main if __FILE__ == $PROGRAM_NAME
