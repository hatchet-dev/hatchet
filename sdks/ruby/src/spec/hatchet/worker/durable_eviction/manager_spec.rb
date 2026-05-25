# frozen_string_literal: true

require "spec_helper"

RSpec.describe Hatchet::WorkerRuntime::DurableEviction::DurableEvictionManager do
  def make_manager(cancel_spy: nil, request_spy: nil)
    cancel = cancel_spy || ->(_key) {}
    request_eviction = request_spy || ->(_key, _rec) {}

    mgr = described_class.new(
      durable_slots: 10,
      cancel_local: cancel,
      request_eviction_with_ack: request_eviction,
      config: Hatchet::WorkerRuntime::DurableEviction::DurableEvictionConfig.new(
        check_interval: 3600.0,
        reserve_slots: 0,
        min_wait_for_capacity_eviction: 0.0,
      ),
    )

    [mgr, cancel, request_eviction]
  end

  it "cancels and unregisters on server eviction" do
    cancel_calls = []
    cancel = ->(key) { cancel_calls << key }

    mgr, = make_manager(cancel_spy: cancel)

    key = "run-1/0"
    mgr.register_run(
      key,
      step_run_id: "ext-1",
      invocation_count: 2,
      eviction_policy: Hatchet::EvictionPolicy.new(ttl: 30),
    )
    mgr.mark_waiting(key, wait_kind: "sleep", resource_id: "s1")

    mgr.handle_server_eviction("ext-1", 2)

    expect(cancel_calls).to eq([key])
    expect(mgr.cache.get(key)).to be_nil
  end

  it "is a no-op for unknown step_run_ids" do
    cancel_calls = []
    cancel = ->(key) { cancel_calls << key }

    mgr, = make_manager(cancel_spy: cancel)

    mgr.register_run("run-1/0", step_run_id: "ext-1", invocation_count: 1, eviction_policy: nil)

    mgr.handle_server_eviction("no-such-id", 1)

    expect(cancel_calls).to be_empty
    expect(mgr.cache.get("run-1/0")).not_to be_nil
  end

  it "only evicts the matching run" do
    cancel_calls = []
    cancel = ->(key) { cancel_calls << key }

    mgr, = make_manager(cancel_spy: cancel)

    mgr.register_run(
      "run-1/0",
      step_run_id: "ext-1",
      invocation_count: 1,
      eviction_policy: Hatchet::EvictionPolicy.new(ttl: 30),
    )
    mgr.register_run(
      "run-2/0",
      step_run_id: "ext-2",
      invocation_count: 1,
      eviction_policy: Hatchet::EvictionPolicy.new(ttl: 30),
    )
    mgr.mark_waiting("run-1/0", wait_kind: "sleep", resource_id: "s1")
    mgr.mark_waiting("run-2/0", wait_kind: "sleep", resource_id: "s2")

    mgr.handle_server_eviction("ext-1", 1)

    expect(cancel_calls).to eq(["run-1/0"])
    expect(mgr.cache.get("run-1/0")).to be_nil
    expect(mgr.cache.get("run-2/0")).not_to be_nil
  end

  it "skips newer invocation counts" do
    cancel_calls = []
    cancel = ->(key) { cancel_calls << key }

    mgr, = make_manager(cancel_spy: cancel)

    mgr.register_run(
      "run-1/0",
      step_run_id: "ext-1",
      invocation_count: 3,
      eviction_policy: Hatchet::EvictionPolicy.new(ttl: 30),
    )
    mgr.mark_waiting("run-1/0", wait_kind: "sleep", resource_id: "s1")

    mgr.handle_server_eviction("ext-1", 2)

    expect(cancel_calls).to be_empty
    expect(mgr.cache.get("run-1/0")).not_to be_nil
  end

  it "evicts on an exact invocation match" do
    cancel_calls = []
    cancel = ->(key) { cancel_calls << key }

    mgr, = make_manager(cancel_spy: cancel)

    mgr.register_run(
      "run-1/0",
      step_run_id: "ext-1",
      invocation_count: 5,
      eviction_policy: Hatchet::EvictionPolicy.new(ttl: 30),
    )
    mgr.mark_waiting("run-1/0", wait_kind: "sleep", resource_id: "s1")

    mgr.handle_server_eviction("ext-1", 5)

    expect(cancel_calls).to eq(["run-1/0"])
    expect(mgr.cache.get("run-1/0")).to be_nil
  end
end
