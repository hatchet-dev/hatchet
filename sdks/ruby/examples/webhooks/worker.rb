# frozen_string_literal: true

# > Webhooks

require "hatchet-sdk"

HATCHET = Hatchet::Client.new(debug: true)

WEBHOOK_TASK = HATCHET.task(
  name: "webhook",
  on_events: ["webhook:test"]
) do |input, ctx|
  {
    "type" => input["type"],
    "message" => input["message"]
  }
end

def main
  worker = HATCHET.worker("webhook-worker", workflows: [WEBHOOK_TASK])
  worker.start
end

main if __FILE__ == $PROGRAM_NAME
