# frozen_string_literal: true

require "hatchet-sdk"

HATCHET = Hatchet::Client.new(debug: true) unless defined?(HATCHET)

# > Step 01 Define Webhook Task
PROCESS_WEBHOOK = HATCHET.task(
  name: "process-webhook",
  on_events: ["webhook:stripe", "webhook:github"],
) do |input, _ctx|
  { "processed" => input["event_id"], "type" => input["type"] }
end

# !!

# > Step 02 Register Webhook
def forward_webhook(event_key, payload)
  HATCHET.event.push(event_key, payload)
end
# forward_webhook("webhook:stripe", { "event_id" => "evt_123", "type" => "payment", "data" => {} })
# !!

# > Step 03 Process Payload
def validate_and_process(input)
  raise "event_id required for deduplication" if input["event_id"].to_s.empty?

  { "processed" => input["event_id"], "type" => input["type"] }
end
# !!

def main
  # > Step 04 Run Worker
  worker = HATCHET.worker("webhook-worker", workflows: [PROCESS_WEBHOOK])
  worker.start
  # !!
end

main if __FILE__ == $PROGRAM_NAME
