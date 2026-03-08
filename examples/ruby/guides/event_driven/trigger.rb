# frozen_string_literal: true

require "hatchet-sdk"

HATCHET = Hatchet::Client.new(debug: true) unless defined?(HATCHET)

# > Step 03 Push Event
# Push an event to trigger the workflow. Use the same key as on_events.
HATCHET.event.push("order:created", "message" => "Order #1234", "source" => "webhook")
