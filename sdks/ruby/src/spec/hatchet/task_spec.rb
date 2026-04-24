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
end
