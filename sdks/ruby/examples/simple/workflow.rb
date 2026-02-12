# frozen_string_literal: true

require "hatchet-sdk"

HATCHET = Hatchet::Client.new

# > Define a workflow
EXAMPLE_WORKFLOW = hatchet.workflow(name: "example-workflow")
