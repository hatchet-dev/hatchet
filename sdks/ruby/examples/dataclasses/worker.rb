# frozen_string_literal: true

require "hatchet-sdk"

HATCHET = Hatchet::Client.new(debug: true)

# > Task using Struct-based input
# Ruby equivalent of Python dataclass -- use plain hashes
SAY_HELLO = hatchet.task(name: "say_hello") do |input, ctx|
  { "message" => "Hello, #{input['name']}!" }
end

def main
  worker = hatchet.worker("test-worker", workflows: [SAY_HELLO])
  worker.start
end

main if __FILE__ == $PROGRAM_NAME
