# frozen_string_literal: true

require_relative "worker"

# > Consume
ref = STREAM_TASK.run_no_wait

HATCHET.runs.subscribe_to_stream(ref.workflow_run_id) do |chunk|
  print chunk
end
