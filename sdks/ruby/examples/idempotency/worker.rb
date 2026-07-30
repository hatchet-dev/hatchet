# frozen_string_literal: true

require 'hatchet-sdk'

HATCHET = Hatchet::Client.new(debug: true) unless defined?(HATCHET)
EVENT_KEY = 'ruby-e2e:idempotency-example'

# > idempotency
IDEMPOTENT_TASK = HATCHET.task(
  name: 'ruby-e2e-idempotent-task',
  idempotency: Hatchet::TTLBasedIdempotencyConfig.new(expression: 'input.id', ttl_ms: 60_000),
  on_events: [EVENT_KEY]
) do |input, _ctx|
  { 'result' => "Hello from task #{input['id']}" }
end

IDEMPOTENT_TASK_SHORT_WINDOW = HATCHET.task(
  name: 'ruby-e2e-idempotent-task-short-window',
  idempotency: Hatchet::TTLBasedIdempotencyConfig.new(expression: 'input.id', ttl_ms: 2_000)
) do |input, _ctx|
  { 'result' => "Hello from task #{input['id']}" }
end
# !!

# > status-based-idempotency
IDEMPOTENT_STATUS_BASED_TASK = HATCHET.task(
  name: 'ruby-e2e-idempotent-status-based-task',
  idempotency: Hatchet::StatusBasedIdempotencyConfig.new(expression: 'input.id', fallback_ttl_ms: 10_000)
) do |input, _ctx|
  { 'result' => "Hello from task #{input['id']}" }
end
# !!

def main
  worker = HATCHET.worker(
    'idempotency-worker',
    workflows: [IDEMPOTENT_TASK, IDEMPOTENT_TASK_SHORT_WINDOW, IDEMPOTENT_STATUS_BASED_TASK]
  )
  worker.start
end

main if __FILE__ == $PROGRAM_NAME
