# frozen_string_literal: true

require "spec_helper"

RSpec.describe Hatchet::WorkerRuntime::DurableEviction::DurableEvictionCache do
  def dt(seconds)
    Time.utc(2026, 1, 1, 0, 0, 0) + seconds
  end

  it "prefers oldest waiting and priority during TTL eviction" do
    cache = described_class.new

    key1 = "run-1/0"
    key2 = "run-2/0"

    high_prio = Hatchet::EvictionPolicy.new(ttl: 10, priority: 10)
    low_prio = Hatchet::EvictionPolicy.new(ttl: 10, priority: 0)

    cache.register_run(key1, step_run_id: "run-1", invocation_count: 1, now: dt(0), eviction_policy: high_prio)
    cache.register_run(key2, step_run_id: "run-2", invocation_count: 1, now: dt(0), eviction_policy: low_prio)

    cache.mark_waiting(key1, now: dt(0), wait_kind: "workflow_run_result", resource_id: "wf1")
    cache.mark_waiting(key2, now: dt(5), wait_kind: "workflow_run_result", resource_id: "wf2")

    chosen = cache.select_eviction_candidate(
      now: dt(20),
      durable_slots: 100,
      reserve_slots: 0,
      min_wait_for_capacity_eviction: 0,
    )
    expect(chosen).to eq(key2)
  end

  it "never selects runs with no eviction policy" do
    cache = described_class.new

    key_no = "run-no/0"
    key_yes = "run-yes/0"

    cache.register_run(key_no, step_run_id: "run-no", invocation_count: 1, now: dt(0), eviction_policy: nil)
    cache.register_run(
      key_yes,
      step_run_id: "run-yes",
      invocation_count: 1,
      now: dt(0),
      eviction_policy: Hatchet::EvictionPolicy.new(ttl: 1),
    )

    cache.mark_waiting(key_no, now: dt(0), wait_kind: "durable_event", resource_id: "x")
    cache.mark_waiting(key_yes, now: dt(0), wait_kind: "durable_event", resource_id: "y")

    chosen = cache.select_eviction_candidate(
      now: dt(10),
      durable_slots: 100,
      reserve_slots: 0,
      min_wait_for_capacity_eviction: 0,
    )
    expect(chosen).to eq(key_yes)
  end

  it "respects allow_capacity_eviction and the min-wait threshold" do
    cache = described_class.new

    key_blocked = "run-blocked/0"
    key_ok = "run-ok/0"

    cache.register_run(
      key_blocked,
      step_run_id: "run-blocked",
      invocation_count: 1,
      now: dt(0),
      eviction_policy: Hatchet::EvictionPolicy.new(ttl: 3600, allow_capacity_eviction: false, priority: 0),
    )
    cache.register_run(
      key_ok,
      step_run_id: "run-ok",
      invocation_count: 1,
      now: dt(0),
      eviction_policy: Hatchet::EvictionPolicy.new(ttl: 3600, allow_capacity_eviction: true, priority: 0),
    )

    cache.mark_waiting(key_blocked, now: dt(0), wait_kind: "durable_event", resource_id: "x")
    cache.mark_waiting(key_ok, now: dt(0), wait_kind: "durable_event", resource_id: "y")

    too_soon = cache.select_eviction_candidate(
      now: dt(5),
      durable_slots: 2,
      reserve_slots: 0,
      min_wait_for_capacity_eviction: 10,
    )
    expect(too_soon).to be_nil

    chosen = cache.select_eviction_candidate(
      now: dt(15),
      durable_slots: 2,
      reserve_slots: 0,
      min_wait_for_capacity_eviction: 10,
    )
    expect(chosen).to eq(key_ok)
  end

  it "keeps the run in waiting state until all concurrent waits resolve" do
    cache = described_class.new
    key = "run-bulk/0"
    policy = Hatchet::EvictionPolicy.new(ttl: 5, priority: 0)

    cache.register_run(key, step_run_id: "run-bulk", invocation_count: 1, now: dt(0), eviction_policy: policy)

    cache.mark_waiting(key, now: dt(1), wait_kind: "spawn_child", resource_id: "child0")
    cache.mark_waiting(key, now: dt(1), wait_kind: "spawn_child", resource_id: "child1")
    cache.mark_waiting(key, now: dt(1), wait_kind: "spawn_child", resource_id: "child2")

    rec = cache.get(key)
    expect(rec).not_to be_nil
    expect(rec.waiting?).to be(true)
    expect(rec.wait_count).to eq(3)

    cache.mark_active(key, now: dt(2))
    expect(rec.waiting?).to be(true)
    expect(rec.wait_count).to eq(2)
    expect(rec.waiting_since).to eq(dt(1))

    chosen = cache.select_eviction_candidate(
      now: dt(10),
      durable_slots: 100,
      reserve_slots: 0,
      min_wait_for_capacity_eviction: 0,
    )
    expect(chosen).to eq(key)

    cache.mark_active(key, now: dt(11))
    expect(rec.waiting?).to be(true)
    expect(rec.wait_count).to eq(1)

    cache.mark_active(key, now: dt(12))
    expect(rec.waiting?).to be(false)
    expect(rec.wait_count).to eq(0)
    expect(rec.waiting_since).to be_nil
  end

  describe "#find_key_by_step_run_id" do
    it "returns the matching key" do
      cache = described_class.new
      cache.register_run("run-a/0", step_run_id: "ext-a", invocation_count: 1, now: dt(0), eviction_policy: nil)
      cache.register_run("run-b/0", step_run_id: "ext-b", invocation_count: 1, now: dt(0), eviction_policy: nil)

      expect(cache.find_key_by_step_run_id("ext-a")).to eq("run-a/0")
      expect(cache.find_key_by_step_run_id("ext-b")).to eq("run-b/0")
    end

    it "returns nil for unknown ids" do
      cache = described_class.new
      cache.register_run("run-a/0", step_run_id: "ext-a", invocation_count: 1, now: dt(0), eviction_policy: nil)

      expect(cache.find_key_by_step_run_id("no-such-id")).to be_nil
    end

    it "returns nil after unregister" do
      cache = described_class.new
      cache.register_run("run-a/0", step_run_id: "ext-a", invocation_count: 1, now: dt(0), eviction_policy: nil)

      expect(cache.find_key_by_step_run_id("ext-a")).to eq("run-a/0")
      cache.unregister_run("run-a/0")
      expect(cache.find_key_by_step_run_id("ext-a")).to be_nil
    end
  end

  it "floors wait_count at zero under extra mark_active calls" do
    cache = described_class.new
    key = "run-extra/0"
    policy = Hatchet::EvictionPolicy.new(ttl: 5, priority: 0)

    cache.register_run(key, step_run_id: "run-extra", invocation_count: 1, now: dt(0), eviction_policy: policy)
    cache.mark_waiting(key, now: dt(0), wait_kind: "sleep", resource_id: "s")

    cache.mark_active(key, now: dt(1))
    cache.mark_active(key, now: dt(2))

    rec = cache.get(key)
    expect(rec).not_to be_nil
    expect(rec.wait_count).to eq(0)
    expect(rec.waiting?).to be(false)
  end
end
