# frozen_string_literal: true

require 'hatchet-sdk'

HATCHET = Hatchet::Client.new(debug: true) unless defined?(HATCHET)

# > idempotency
IDEMPOTENT_TASK = HATCHET.task(
  name: 'idempotent-task',
  idempotency: { expression: 'input.id', ttl_ms: 60_000 },
  on_events: ['idempotency:example']
) do |input, _ctx|
  { 'result' => "Hello from task #{input['id']}" }
end

def main
  worker = HATCHET.worker('idempotency-worker', workflows: [IDEMPOTENT_TASK])
  worker.start
end

main if __FILE__ == $PROGRAM_NAME
