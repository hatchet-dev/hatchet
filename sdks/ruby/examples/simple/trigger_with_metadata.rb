# frozen_string_literal: true

require_relative "worker"

# > Trigger with metadata
SIMPLE.run(
  {},
  options: Hatchet::TriggerWorkflowOptions.new(
    additional_metadata: { "source" => "api" }
  )
)
# !!
