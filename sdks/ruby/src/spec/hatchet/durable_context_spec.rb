# frozen_string_literal: true

require "spec_helper"

RSpec.describe Hatchet::DurableContext do
  it "tracks eviction after send_event" do
    ctx = described_class.new(
      workflow_run_id: "workflow-1",
      step_run_id: "step-1",
      client: nil,
    )

    listener = instance_double(Hatchet::WorkerRuntime::DurableEventListener)
    manager = double("eviction_manager")

    ctx.durable_event_listener = listener
    ctx.eviction_manager = manager
    ctx.action_key = "step-1/0"
    ctx.invocation_count = 1
    ctx.engine_version = Hatchet::MinEngineVersion::DURABLE_EVICTION

    expect(listener).to receive(:send_event).with("step-1", 1, kind_of(Hatchet::WorkerRuntime::DurableEventListener::WaitForEvent))
      .ordered
      .and_return(branch_id: "branch-1", node_id: "node-1")
    expect(manager).to receive(:mark_waiting).with(
      "step-1/0",
      wait_kind: "wait_for",
      resource_id: "sleep:15s-0",
    ).ordered
    expect(listener).to receive(:wait_for_callback).with("step-1", 1, "branch-1", "node-1")
      .ordered
      .and_return(payload: { "status" => "completed" })
    expect(manager).to receive(:mark_active).with("step-1/0").ordered

    expect(ctx.sleep_for(duration: 15)).to eq("status" => "completed")
  end

  it "forwards an optional wait label to the durable listener" do
    ctx = described_class.new(
      workflow_run_id: "workflow-1",
      step_run_id: "step-1",
      client: nil,
    )

    listener = instance_double(Hatchet::WorkerRuntime::DurableEventListener)

    ctx.durable_event_listener = listener
    ctx.invocation_count = 1
    ctx.engine_version = Hatchet::MinEngineVersion::DURABLE_EVICTION

    expect(listener).to receive(:send_event) do |step_run_id, invocation_count, event|
      expect(step_run_id).to eq("step-1")
      expect(invocation_count).to eq(1)
      expect(event.label).to eq("waiting for payment")
      { branch_id: "branch-1", node_id: "node-1" }
    end
    expect(listener).to receive(:wait_for_callback).with("step-1", 1, "branch-1", "node-1")
      .and_return(payload: {})

    expect(
      ctx.wait_for(
        "payment",
        Hatchet::UserEventCondition.new(event_key: "payment"),
        label: "waiting for payment",
      ),
    ).to eq({})
  end

  it "falls back to the legacy durable wait path when engine version is unknown" do
    ctx = described_class.new(
      workflow_run_id: "workflow-1",
      step_run_id: "step-1",
      client: nil,
    )

    listener = instance_double(Hatchet::WorkerRuntime::DurableEventListener)

    ctx.durable_event_listener = listener
    ctx.invocation_count = 1
    ctx.engine_version = nil

    expect(listener).not_to receive(:send_event)
    expect(listener).not_to receive(:wait_for_callback)
    expect(ctx).to receive(:legacy_wait_for).with("sleep:15s-0", kind_of(V1::DurableEventListenerConditions))
      .and_return({})

    expect(ctx.sleep_for(duration: 15)).to eq({})
  end
end
