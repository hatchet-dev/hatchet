# frozen_string_literal: true

require "spec_helper"

RSpec.describe Hatchet::WorkerRuntime::DurableEventListener do
  let(:config) { instance_double(Hatchet::Config, auth_metadata: [], host_port: "localhost:7070") }
  let(:channel) { double("channel") }
  let(:logger) { instance_double(Logger, info: nil, warn: nil, error: nil, debug: nil) }
  let(:listener) { described_class.new(config: config, channel: channel, logger: logger) }

  def entry_completed(task:, invocation:, node:, branch: 0, order: nil, payload: '{"ok":true}')
    ref = V1::DurableEventLogEntryRef.new(
      durable_task_external_id: task,
      invocation_count: invocation,
      branch_id: branch,
      node_id: node,
    )
    completed = V1::DurableTaskEventLogEntryCompletedResponse.new(ref: ref, payload: payload)
    completed.satisfied_order = order if order
    V1::DurableTaskResponse.new(entry_completed: completed)
  end

  def register_waiter(task, invocation, node, branch: 0)
    queue = Queue.new
    listener.instance_variable_get(:@pending_callbacks)[[task, invocation, branch, node]] = queue
    queue
  end

  def buffered_completions
    listener.instance_variable_get(:@buffered_completions)
  end

  def gates
    listener.instance_variable_get(:@gates)
  end

  describe "ordered release gate" do
    it "holds out-of-order completions until the missing order arrives" do
      q0 = register_waiter("task1", 1, 0)
      q1 = register_waiter("task1", 1, 1)

      listener.handle_response_for_test(entry_completed(task: "task1", invocation: 1, node: 1, order: 2))
      expect(q0).to be_empty
      expect(q1).to be_empty

      listener.handle_response_for_test(entry_completed(task: "task1", invocation: 1, node: 0, order: 1))
      kind, result = q0.pop
      expect(kind).to eq(:ok)
      expect(result[:node_id]).to eq(0)

      # the woken continuation has not parked yet: order 2 stays held
      expect(q1).to be_empty

      listener.send(:notify_parked, ["task1", 1])
      kind, result = q1.pop
      expect(kind).to eq(:ok)
      expect(result[:node_id]).to eq(1)
    end

    it "keeps pumping when releases are buffered instead of waking a waiter" do
      # no waiters registered: the continuation is still running, completions
      # are buffered without closing the gate
      listener.handle_response_for_test(entry_completed(task: "task1", invocation: 1, node: 0, order: 1))
      listener.handle_response_for_test(entry_completed(task: "task1", invocation: 1, node: 1, order: 2))

      expect(buffered_completions).to have_key(["task1", 1, 0, 0])
      expect(buffered_completions).to have_key(["task1", 1, 0, 1])
      expect(gates[["task1", 1]].released).to eq(2)
    end

    it "bypasses the gate for re-delivery of already-released orders" do
      listener.handle_response_for_test(entry_completed(task: "task1", invocation: 1, node: 0, order: 1))
      expect(gates[["task1", 1]].released).to eq(1)

      q0 = register_waiter("task1", 1, 9)
      listener.handle_response_for_test(entry_completed(task: "task1", invocation: 1, node: 9, order: 1))

      kind, = q0.pop
      expect(kind).to eq(:ok)
      expect(gates[["task1", 1]].released).to eq(1)
    end

    it "delivers legacy completions with no satisfied order immediately" do
      register_waiter("task1", 1, 0)
      q5 = register_waiter("task1", 1, 5)

      # gate is blocked on a hole (order 1 missing)
      listener.handle_response_for_test(entry_completed(task: "task1", invocation: 1, node: 0, order: 2))

      listener.handle_response_for_test(entry_completed(task: "task1", invocation: 1, node: 5))
      kind, result = q5.pop
      expect(kind).to eq(:ok)
      expect(result[:node_id]).to eq(5)
    end

    it "scopes gates per task invocation" do
      q_inv1 = register_waiter("task1", 1, 0)
      q_inv2 = register_waiter("task1", 2, 0)

      listener.handle_response_for_test(entry_completed(task: "task1", invocation: 1, node: 0, order: 2))
      listener.handle_response_for_test(entry_completed(task: "task1", invocation: 2, node: 0, order: 1))

      expect(q_inv1).to be_empty
      kind, = q_inv2.pop
      expect(kind).to eq(:ok)
    end

    it "fails the invocation's waiters with a NonDeterminismError on gap timeout" do
      listener.instance_variable_set(:@gap_timeout_s, -1)

      q1 = register_waiter("task1", 1, 1)
      listener.handle_response_for_test(entry_completed(task: "task1", invocation: 1, node: 1, order: 2))
      expect(q1).to be_empty

      listener.send(:sweep_gates)

      kind, exc = q1.pop
      expect(kind).to eq(:err)
      expect(exc).to be_a(Hatchet::NonDeterminismError)
      expect(exc.message).to match(/satisfied order 1 was never delivered/)
      expect(gates).not_to have_key(["task1", 1])
    end

    it "forces the gate open on park timeout" do
      listener.instance_variable_set(:@park_timeout_s, -1)

      q0 = register_waiter("task1", 1, 0)
      q1 = register_waiter("task1", 1, 1)

      listener.handle_response_for_test(entry_completed(task: "task1", invocation: 1, node: 0, order: 1))
      kind, = q0.pop
      expect(kind).to eq(:ok)

      listener.handle_response_for_test(entry_completed(task: "task1", invocation: 1, node: 1, order: 2))
      expect(q1).to be_empty

      # the woken continuation never parks: park timeout forces the gate open
      expect(logger).to receive(:warn).with(/did not park/)
      listener.send(:sweep_gates)

      kind, = q1.pop
      expect(kind).to eq(:ok)
    end

    it "opens the gate on notify_invocation_quiesced" do
      q0 = register_waiter("task1", 1, 0)
      q1 = register_waiter("task1", 1, 1)

      listener.handle_response_for_test(entry_completed(task: "task1", invocation: 1, node: 0, order: 1))
      kind, = q0.pop
      expect(kind).to eq(:ok)

      listener.handle_response_for_test(entry_completed(task: "task1", invocation: 1, node: 1, order: 2))
      expect(q1).to be_empty

      # the task fn returned without parking again: quiesce resets the wake
      # count and pumps the held completion
      listener.notify_invocation_quiesced("task1", 1)
      kind, = q1.pop
      expect(kind).to eq(:ok)
    end

    it "drops gate state on cleanup_task_state" do
      listener.handle_response_for_test(entry_completed(task: "task1", invocation: 1, node: 0, order: 1))
      listener.handle_response_for_test(entry_completed(task: "task1", invocation: 3, node: 0, order: 1))
      expect(gates).to have_key(["task1", 1])
      expect(gates).to have_key(["task1", 3])

      listener.cleanup_task_state("task1", 2)

      expect(gates).not_to have_key(["task1", 1])
      expect(gates).to have_key(["task1", 3])
    end

    it "clears gate state on fail_all_pending" do
      listener.handle_response_for_test(entry_completed(task: "task1", invocation: 1, node: 0, order: 1))
      expect(gates).not_to be_empty

      listener.send(:fail_all_pending, Hatchet::Error.new("stopped"))

      expect(gates).to be_empty
    end

    it "opens the gate when a continuation parks via wait_for_callback registration" do
      q0 = register_waiter("task1", 1, 0)

      listener.handle_response_for_test(entry_completed(task: "task1", invocation: 1, node: 0, order: 1))
      kind, = q0.pop
      expect(kind).to eq(:ok)

      listener.handle_response_for_test(entry_completed(task: "task1", invocation: 1, node: 1, order: 2))
      expect(buffered_completions).not_to have_key(["task1", 1, 0, 1])

      # the continuation parks by awaiting its next entry: the buffered hit is
      # empty so wait_for_callback registers a queue and signals the park,
      # which pumps order 2 straight into the new queue
      result = listener.wait_for_callback("task1", 1, 0, 1)
      expect(result[:node_id]).to eq(1)
    end
  end
end
