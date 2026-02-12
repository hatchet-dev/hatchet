#!/usr/bin/env ruby

require 'hatchet-sdk'
# require_relative '../src/lib/hatchet-sdk'

# Initialize the Hatchet client
HATCHET = Hatchet::Client.new()

result = hatchet.events.create(
  key: "test-event",
  data: {
    message: "test"
  }
)
puts "Event created: #{result.inspect}"


run = hatchet.runs.create(
  name: "simple",
  input: {
    Message: "test workflow run"
  },
)

puts "TriggeredRun ID: #{run.metadata.id}"

result = hatchet.runs.poll(run.metadata.id)

puts "Runs client initialized: #{result.inspect}"
puts "Run status: #{result.run.status}"
