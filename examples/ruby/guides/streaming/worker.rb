# frozen_string_literal: true

require "hatchet-sdk"

HATCHET = Hatchet::Client.new(debug: true) unless defined?(HATCHET)

# > Step 01 Define Streaming Task
STREAM_TASK = HATCHET.task(name: "stream-example") do |_input, ctx|
  5.times do |i|
    ctx.put_stream("chunk-#{i}")
    sleep 0.5
  end
  { "status" => "done" }
end


# > Step 02 Emit Chunks
def emit_chunks(ctx)
  5.times do |i|
    ctx.put_stream("chunk-#{i}")
    sleep 0.5
  end
end

def main
  # > Step 04 Run Worker
  worker = HATCHET.worker("streaming-worker", workflows: [STREAM_TASK])
  worker.start
end

main if __FILE__ == $PROGRAM_NAME
