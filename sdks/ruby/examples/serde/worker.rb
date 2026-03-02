# frozen_string_literal: true

# > Custom Serialization/Deserialization

require "hatchet-sdk"
require "base64"
require "zlib"

HATCHET = Hatchet::Client.new(debug: true) unless defined?(HATCHET)

SERDE_WORKFLOW = HATCHET.workflow(name: "serde-example-workflow")

GENERATE_RESULT = SERDE_WORKFLOW.task(:generate_result) do |input, ctx|
  compressed = Base64.strict_encode64(Zlib::Deflate.deflate("my_result"))
  { "result" => compressed }
end

SERDE_WORKFLOW.task(:read_result, parents: [GENERATE_RESULT]) do |input, ctx|
  encoded = ctx.task_output(GENERATE_RESULT)["result"]
  decoded = Zlib::Inflate.inflate(Base64.strict_decode64(encoded))
  { "final_result" => decoded }
end

# !!

def main
  worker = HATCHET.worker("test-worker", workflows: [SERDE_WORKFLOW])
  worker.start
end

main if __FILE__ == $PROGRAM_NAME
