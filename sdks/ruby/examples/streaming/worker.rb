# frozen_string_literal: true

require "hatchet-sdk"

HATCHET = Hatchet::Client.new(debug: false)

# > Streaming
ANNA_KARENINA = <<~TEXT
  Happy families are all alike; every unhappy family is unhappy in its own way.

  Everything was in confusion in the Oblonskys' house. The wife had discovered that the husband was carrying on an intrigue with a French girl, who had been a governess in their family, and she had announced to her husband that she could not go on living in the same house with him.
TEXT

STREAM_CHUNKS = ANNA_KARENINA.scan(/.{1,10}/)

STREAM_TASK = hatchet.task(name: "stream_task") do |input, ctx|
  # Sleeping to avoid race conditions
  sleep 2

  STREAM_CHUNKS.each do |chunk|
    ctx.put_stream(chunk)
    sleep 0.20
  end
end

def main
  worker = hatchet.worker("test-worker", workflows: [STREAM_TASK])
  worker.start
end

main if __FILE__ == $PROGRAM_NAME
