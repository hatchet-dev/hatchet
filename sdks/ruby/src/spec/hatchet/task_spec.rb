# frozen_string_literal: true

require "spec_helper"

RSpec.describe Hatchet::Task do
  let(:config) { instance_double(Hatchet::Config, apply_namespace: "test-service") }

  it "marks durable tasks as durable in workflow registration protos" do
    task = described_class.new(
      name: "durable_task",
      durable: true,
    ) { |_input, _ctx| nil }

    proto = task.to_proto("test-service", config: config)

    expect(proto.is_durable).to be(true)
  end

  it "leaves non-durable tasks unset in workflow registration protos" do
    task = described_class.new(
      name: "normal_task",
      durable: false,
    ) { |_input, _ctx| nil }

    proto = task.to_proto("test-service", config: config)

    expect(proto.is_durable).to be(false)
  end

  describe "#display_name" do
    it "threads a task-level CEL expression into the task proto" do
      task = described_class.new(
        name: "enrich",
        display_name: "'enrich-' + input.name",
      ) { |_input, _ctx| nil }

      proto = task.to_proto("test-service", config: config)

      expect(proto.display_name).to eq("'enrich-' + input.name")
      expect(proto.has_display_name?).to be(true)
    end

    it "leaves display_name unset on the task proto when not provided" do
      task = described_class.new(name: "plain") { |_input, _ctx| nil }

      proto = task.to_proto("test-service", config: config)

      expect(proto.has_display_name?).to be(false)
    end
  end

  it "serializes batch config onto the registration proto" do
    batch = Hatchet::BatchTaskConfig.new(max_size: 3, max_interval_ms: 200, group_key: "input.group")
    task = described_class.new(
      name: "batch_task",
      batch: batch,
    ) { |inputs, _ctx| inputs }

    proto = task.to_proto("test-service", config: config)

    expect(proto.batch).not_to be_nil
    expect(proto.batch.batch_max_size).to eq(3)
    expect(proto.batch.batch_max_interval_ms).to eq(200)
    expect(proto.batch.batch_group_key).to eq("input.group")
  end

  it "forces retries to 0 for batch tasks even when retries is explicitly set" do
    batch = Hatchet::BatchTaskConfig.new(max_size: 3)
    task = described_class.new(
      name: "batch_task_with_retries",
      batch: batch,
      retries: 5,
    ) { |inputs, _ctx| inputs }

    proto = task.to_proto("test-service", config: config)

    expect(proto.retries).to eq(0)
  end

  it "leaves batch unset on the proto for non-batch tasks" do
    task = described_class.new(
      name: "normal_task",
    ) { |_input, _ctx| nil }

    proto = task.to_proto("test-service", config: config)

    expect(proto.batch).to be_nil
  end
end
