#!/usr/bin/env ruby

require 'hatchet-sdk'

hatchet = Hatchet::Client.new()

puts "Hello, World from Hatchet!"

puts hatchet.config.token
puts hatchet.config.tls_config.strategy
puts hatchet.config.tls_config.cert_file
puts hatchet.config.tls_config.key_file
puts hatchet.config.tls_config.root_ca_file
puts hatchet.config.tls_config.server_name
puts hatchet.config.namespace
puts hatchet.config.tenant_id