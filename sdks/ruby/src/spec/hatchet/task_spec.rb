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
end
