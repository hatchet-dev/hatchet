# frozen_string_literal: true

# > Simple

require "hatchet-sdk"

HATCHET = Hatchet::Client.new(debug: true)

SIMPLE = HATCHET.task(name: "simple") do |input, ctx|
  { "result" => "Hello, world!" }
end

SIMPLE_DURABLE = HATCHET.durable_task(name: "simple_durable") do |input, ctx|
  result = SIMPLE.run(input)
  { "result" => result["result"] }
end


def main
  worker = HATCHET.worker("test-worker", workflows: [SIMPLE, SIMPLE_DURABLE])
  worker.start
end

main if __FILE__ == $PROGRAM_NAME
