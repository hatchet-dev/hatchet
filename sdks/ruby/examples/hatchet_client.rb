#!/usr/bin/env ruby

require 'hatchet-sdk'

hatchet = Hatchet::Client.new()

puts "Hatchet Token: #{hatchet.config.token[0, 10]}..."
puts "Hatchet Namespace: #{hatchet.config.namespace}"
puts "Hatchet Tenant ID: #{hatchet.config.tenant_id}"