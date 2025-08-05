#!/usr/bin/env ruby

# require 'hatchet-sdk'
require_relative '../src/lib/hatchet-sdk'

# Initialize the Hatchet client
hatchet = Hatchet::Client.new()

puts "Hatchet Token: #{hatchet.config.token[0, 10]}..."
puts "Hatchet Namespace: #{hatchet.config.namespace}"
puts "Hatchet Tenant ID: #{hatchet.config.tenant_id}"

# Get the events client - this should show IDE hints
events_client = hatchet.events

# Create the event request using the reexported class
event_request = HatchetSdkRest::CreateEventRequest.new(
  key: "test-event",
  data: {
    message: "test"
  }
)

result = events_client.create(event_request)
puts "Event created: #{result.inspect}"