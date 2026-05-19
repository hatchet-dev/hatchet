# frozen_string_literal: true

require "hatchet-sdk"

HATCHET = Hatchet::Client.new unless defined?(HATCHET)

# > Simple task
FIRST_TASK = HATCHET.task(name: "first-task") do |input, ctx|
  puts "first-task called"
  { "transformed_message" => input["message"].downcase }
end
