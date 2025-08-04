#!/usr/bin/env ruby

require 'hatchet'

client = Hatchet::Client.new("test")

puts "Hello, World from Hatchet!"

puts client.api_key