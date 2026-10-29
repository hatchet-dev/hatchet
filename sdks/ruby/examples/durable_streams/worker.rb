# frozen_string_literal: true

require "hatchet-sdk"
require "json"

HATCHET = Hatchet::Client.new(debug: false) unless defined?(HATCHET)

# > Publishing
DURABLE_STREAMS_PROGRESS_TASK = HATCHET.task(name: "durable-streams-progress") do |input, _ctx|
  total = 5

  (1..total).each do |step|
    sleep 0.5
    HATCHET.streams.publish(input["topic"], JSON.generate({ "step" => step, "total" => total }))
  end

  { "steps" => total }
end
# !!

# > Reset on retry
DURABLE_STREAMS_RETRYING_PROGRESS_TASK = HATCHET.task(name: "durable-streams-retrying-progress", retries: 1) do |input, ctx|
  HATCHET.streams.reset(input["topic"])

  3.times do |chunk|
    HATCHET.streams.publish(input["topic"], JSON.generate({ "attempt" => ctx.retry_count, "chunk" => chunk }))
  end

  raise "intentional failure before the final message" if ctx.retry_count.zero?

  HATCHET.streams.publish(input["topic"], JSON.generate({ "done" => true }))

  { "chunks" => 3 }
end
# !!

def main
  worker = HATCHET.worker(
    "durable-streams-worker",
    workflows: [DURABLE_STREAMS_PROGRESS_TASK, DURABLE_STREAMS_RETRYING_PROGRESS_TASK],
  )
  worker.start
end

main if __FILE__ == $PROGRAM_NAME
