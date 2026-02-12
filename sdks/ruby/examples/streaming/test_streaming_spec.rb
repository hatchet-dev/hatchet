# frozen_string_literal: true

require_relative "../spec_helper"
require_relative "worker"

RSpec.describe "StreamTask" do
  it "streams chunks in order and completely" do
    ref = STREAM_TASK.run_no_wait

    received_chunks = []
    hatchet.runs.subscribe_to_stream(ref.workflow_run_id) do |chunk|
      received_chunks << chunk
    end

    ref.result

    expect(received_chunks.length).to eq(STREAM_CHUNKS.length)

    received_chunks.each_with_index do |chunk, ix|
      expect(chunk).to eq(STREAM_CHUNKS[ix])
    end

    expect(received_chunks.join).to eq(STREAM_CHUNKS.join)
  end
end
