# frozen_string_literal: true

require 'hatchet-sdk'

HATCHET = Hatchet::Client.new(debug: true) unless defined?(HATCHET)

# > Step 03 Subscribe Client
# Client triggers the task and subscribes to the stream.
def run_and_subscribe
  run = HATCHET.runs.create(workflow_name: 'stream-example', input: {})
  HATCHET.runs.subscribe_to_stream(run.run_id) do |chunk|
    puts chunk
  end
end
