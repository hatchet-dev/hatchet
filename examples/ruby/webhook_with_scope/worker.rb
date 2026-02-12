# frozen_string_literal: true

require "hatchet-sdk"

HATCHET = Hatchet::Client.new(debug: true)

WEBHOOK_WITH_SCOPE = HATCHET.task(
  name: "webhook_with_scope",
  on_events: ["webhook-scope:test"],
  default_filters: [
    Hatchet::DefaultFilter.new(
      expression: "true",
      scope: "test-scope-value",
      payload: {}
    )
  ]
) do |input, ctx|
  input
end

WEBHOOK_WITH_STATIC_PAYLOAD = HATCHET.task(
  name: "webhook_with_static_payload",
  on_events: ["webhook-static:test"]
) do |input, ctx|
  input
end

def main
  worker = HATCHET.worker(
    "webhook-scope-worker",
    workflows: [WEBHOOK_WITH_SCOPE, WEBHOOK_WITH_STATIC_PAYLOAD]
  )
  worker.start
end

main if __FILE__ == $PROGRAM_NAME
