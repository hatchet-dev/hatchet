# frozen_string_literal: true

require "hatchet-sdk"

HATCHET = Hatchet::Client.new unless defined?(HATCHET)

RETURN_EXCEPTIONS_TASK = HATCHET.task(name: "return_exceptions_task") do |input, _ctx|
  raise "error in task with index #{input["index"]}" if input["index"].to_i.even?

  { "message" => "this is a successful task." }
end
