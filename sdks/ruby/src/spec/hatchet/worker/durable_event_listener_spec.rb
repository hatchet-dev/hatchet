# frozen_string_literal: true

require "spec_helper"

RSpec.describe Hatchet::WorkerRuntime::DurableEventListener do
  let(:config) { instance_double(Hatchet::Config, auth_metadata: [], host_port: "localhost:7070") }
  let(:channel) { double("channel") }
  let(:logger) { instance_double(Logger, info: nil, warn: nil, error: nil, debug: nil) }

  def build_listener(**opts)
    described_class.new(config: config, channel: channel, logger: logger, **opts)
  end

  describe "#fail_pending_acks" do
    it "clears and fails pending event acks" do
      listener = build_listener
      queue = Queue.new
      listener.instance_variable_get(:@pending_event_acks)[["task1", 1]] = queue

      listener.send(:fail_pending_acks, Hatchet::Error.new("disconnected"))

      expect(listener.instance_variable_get(:@pending_event_acks)).to be_empty
      msg = queue.pop
      expect(msg[0]).to eq(:err)
      expect(msg[1]).to be_a(Hatchet::Error)
      expect(msg[1].message).to match(/disconnected/)
    end

    it "clears and fails pending eviction acks" do
      listener = build_listener
      queue = Queue.new
      listener.instance_variable_get(:@pending_eviction_acks)[["task1", 1]] = queue

      listener.send(:fail_pending_acks, Hatchet::Error.new("disconnected"))

      expect(listener.instance_variable_get(:@pending_eviction_acks)).to be_empty
      msg = queue.pop
      expect(msg[0]).to eq(:err)
    end

    it "does not fail pending callbacks, since they survive reconnection" do
      listener = build_listener
      queue = Queue.new
      key = ["task1", 1, 0, 1]
      listener.instance_variable_get(:@pending_callbacks)[key] = queue

      listener.send(:fail_pending_acks, Hatchet::Error.new("disconnected"))

      expect(listener.instance_variable_get(:@pending_callbacks)).to have_key(key)
      expect(queue.empty?).to be(true)
    end
  end

  describe "#cleanup_task_state" do
    it "removes pending state for invocation counts <= the given count" do
      listener = build_listener

      cb_queue_old = Queue.new
      cb_queue_new = Queue.new
      listener.instance_variable_get(:@pending_callbacks)[["task1", 1, 0, 1]] = cb_queue_old
      listener.instance_variable_get(:@pending_callbacks)[["task1", 3, 0, 1]] = cb_queue_new

      ack_queue_old = Queue.new
      ack_queue_new = Queue.new
      listener.instance_variable_get(:@pending_event_acks)[["task1", 1]] = ack_queue_old
      listener.instance_variable_get(:@pending_event_acks)[["task1", 3]] = ack_queue_new

      listener.instance_variable_get(:@buffered_completions)[["task1", 1, 0, 1]] = [Time.now, { payload: {} }]
      listener.instance_variable_get(:@buffered_completions)[["task1", 3, 0, 1]] = [Time.now, { payload: {} }]

      listener.cleanup_task_state("task1", 2)

      callbacks = listener.instance_variable_get(:@pending_callbacks)
      acks = listener.instance_variable_get(:@pending_event_acks)
      buffered = listener.instance_variable_get(:@buffered_completions)

      expect(callbacks).to have_key(["task1", 3, 0, 1])
      expect(callbacks).not_to have_key(["task1", 1, 0, 1])
      expect(acks).to have_key(["task1", 3])
      expect(acks).not_to have_key(["task1", 1])
      expect(buffered).to have_key(["task1", 3, 0, 1])
      expect(buffered).not_to have_key(["task1", 1, 0, 1])
    end
  end

  describe "#fail_all_pending" do
    it "fails pending callbacks too" do
      listener = build_listener
      cb_queue = Queue.new
      listener.instance_variable_get(:@pending_callbacks)[["task1", 1, 0, 1]] = cb_queue

      listener.send(:fail_all_pending, Hatchet::Error.new("stopped"))

      expect(listener.instance_variable_get(:@pending_callbacks)).to be_empty
      msg = cb_queue.pop
      expect(msg[0]).to eq(:err)
    end
  end

  describe "#mark_stream_unavailable" do
    it "clears the request queue and fails pending acks" do
      listener = build_listener
      request_queue = Queue.new
      eviction_queue = Queue.new

      listener.instance_variable_set(:@request_queue, request_queue)
      listener.instance_variable_set(:@stream, Object.new)
      listener.instance_variable_get(:@pending_eviction_acks)[["task1", 1]] = eviction_queue

      listener.send(:mark_stream_unavailable, Hatchet::Error.new("disconnected"))

      expect(listener.instance_variable_get(:@request_queue)).to be_nil
      expect(listener.instance_variable_get(:@stream)).to be_nil
      expect(request_queue.closed?).to be(true)

      msg = eviction_queue.pop
      expect(msg[0]).to eq(:err)
      expect(msg[1].message).to match(/disconnected/)
    end
  end

  describe "#build_event_request" do
    it "raises ArgumentError for unknown event types" do
      listener = build_listener
      expect { listener.send(:build_event_request, "t", 1, Object.new) }.to raise_error(ArgumentError, /Unknown/)
    end
  end

  describe "#parse_entry_completed" do
    it "parses frozen protobuf payload strings" do
      listener = build_listener
      completed = instance_double(
        "DurableTaskEventLogEntryCompletedResponse",
        payload: "{\"ok\":true}",
        ref: instance_double(
          "DurableEventLogEntryRef",
          durable_task_external_id: "task1",
          node_id: 1,
        ),
      )

      result = listener.send(:parse_entry_completed, completed)

      expect(result[:payload]).to eq({ "ok" => true })
    end
  end
end
