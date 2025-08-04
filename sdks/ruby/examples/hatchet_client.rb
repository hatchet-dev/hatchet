#!/usr/bin/env ruby

require 'hatchet-sdk'

hatchet = Hatchet::Client.new()

puts "Hatchet Token: #{hatchet.config.token[0, 10]}..."
puts "Hatchet Namespace: #{hatchet.config.namespace}"
puts "Hatchet Tenant ID: #{hatchet.config.tenant_id}"


# Create a properly configured REST client
rest_client = Hatchet::Clients.rest_client(hatchet.config)

# Create an EventApi instance 
event_api = Hatchet::Clients::Rest::EventApi.new(rest_client)

# Create the event request with proper structure
event_request = HatchetSdkRest::CreateEventRequest.new(
  key: "test-event",
  data: {
    message: "test"
  }
)

# Create the event
result = event_api.event_create(hatchet.config.tenant_id, event_request)
puts "Event created: #{result.inspect}"