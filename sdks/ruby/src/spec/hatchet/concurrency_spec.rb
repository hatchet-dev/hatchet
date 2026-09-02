# frozen_string_literal: true

require "spec_helper"

RSpec.describe Hatchet::ConcurrencyExpression do
  it "serializes a static max_runs onto the static proto field" do
    proto = described_class.new(
      expression: "input.group",
      max_runs: 5,
      limit_strategy: :group_round_robin
    ).to_proto

    expect(proto.expression).to eq("input.group")
    expect(proto.max_runs).to eq(5)
    expect(proto.limit_strategy).to eq(:GROUP_ROUND_ROBIN)
    expect(proto.has_max_runs_expression?).to be(false)
  end

  it "serializes a String max_runs as a CEL expression with a static default of 1" do
    proto = described_class.new(
      expression: "input.tier",
      max_runs: "input.tier == 'premium' ? 10 : 1",
      limit_strategy: :group_round_robin
    ).to_proto

    expect(proto.max_runs).to eq(1)
    expect(proto.max_runs_expression).to eq("input.tier == 'premium' ? 10 : 1")
  end
end

RSpec.describe Hatchet::SharedConcurrency do
  it "serializes as a tenant-scoped entry" do
    proto = described_class.new(
      name: "tenant-wide-limit",
      expression: "input.group",
      max_runs: 1,
      limit_strategy: :group_round_robin
    ).to_proto

    expect(proto.name).to eq("tenant-wide-limit")
    expect(proto.is_tenant_scoped).to be(true)
    expect(proto.max_runs).to eq(1)
    expect(proto.has_max_runs_expression?).to be(false)
  end

  it "supports a CEL max_runs expression" do
    proto = described_class.new(
      name: "tenant-wide-limit",
      expression: "input.group",
      max_runs: "input.limit"
    ).to_proto

    expect(proto.max_runs).to eq(1)
    expect(proto.max_runs_expression).to eq("input.limit")
  end

  it "rejects an empty name" do
    expect do
      described_class.new(name: "", expression: "input.group")
    end.to raise_error(ArgumentError, /name must be non-empty/)
  end
end
