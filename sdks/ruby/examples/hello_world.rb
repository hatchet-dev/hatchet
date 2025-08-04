#!/usr/bin/env ruby

require 'hatchet-sdk'

client = Hatchet::Client.new("test")

puts "Hello, World from Hatchet!"

puts client.api_key