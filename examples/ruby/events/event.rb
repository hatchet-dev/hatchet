# frozen_string_literal: true

require "hatchet-sdk"

HATCHET = Hatchet::Client.new

# > Event trigger
HATCHET.event.push("user:create", { "should_skip" => false })

# > Event trigger with metadata
HATCHET.event.push(
  "user:create",
  { "userId" => "1234", "should_skip" => false },
  additional_metadata: { "source" => "api" }
)
