#!/usr/bin/env ruby

# require 'hatchet-sdk'
require_relative '../src/lib/hatchet-sdk'

# Initialize the Hatchet client
hatchet = Hatchet::Client.new()

# event_request = HatchetSdkRest::CreateEventRequest.new(
#   key: "test-event",
#   data: {
#     message: "test"
#   }
# )

# result = hatchet.events.create(event_request)
# puts "Event created: #{result.inspect}"


# Create a workflow run request using the reexported class
workflow_run_request = HatchetSdkRest::V1TriggerWorkflowRunRequest.new(
  workflow_name: "simple",
  input: {
    message: "test workflow run"
  }
)

run = hatchet.runs.create(workflow_run_request)

puts "Runs client initialized: #{run.inspect}"
puts "Run ID: #{run.metadata.id}"