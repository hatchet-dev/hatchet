# frozen_string_literal: true

require "hatchet-sdk"

HATCHET = Hatchet::Client.new(debug: true) unless defined?(HATCHET)

# > Step 01 Define Event Task
EVENT_WF = HATCHET.workflow(name: "EventDrivenWorkflow", on_events: ["order:created", "user:signup"])

EVENT_WF.task(:process_event) do |input, ctx|
  { "processed" => input["message"], "source" => input["source"] || "api" }
end


# > Step 02 Register Event Trigger
# Push an event from your app to trigger the workflow. Use the same key as on_events.
def push_order_event
  HATCHET.event.push("order:created", "message" => "Order #1234", "source" => "webhook")
end

def main
  # > Step 04 Run Worker
  worker = HATCHET.worker("event-driven-worker", workflows: [EVENT_WF])
  worker.start
end

main if __FILE__ == $PROGRAM_NAME
