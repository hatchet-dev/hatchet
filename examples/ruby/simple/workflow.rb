# frozen_string_literal: true

require "hatchet-sdk"

HATCHET = Hatchet::Client.new unless defined?(HATCHET)

# > Define a workflow
EXAMPLE_WORKFLOW = HATCHET.workflow(name: "example-workflow")

