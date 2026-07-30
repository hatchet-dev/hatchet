# frozen_string_literal: true

require "spec_helper"

RSpec.describe Hatchet::BatchTaskConfig do
  it "serializes required and optional fields to proto" do
    config = described_class.new(
      max_size: 5,
      max_interval_ms: 200,
      group_key: "input.group",
      group_max_runs: 3,
      broadcast_output: true,
    )

    proto = config.to_proto

    expect(proto.batch_max_size).to eq(5)
    expect(proto.batch_max_interval_ms).to eq(200)
    expect(proto.batch_group_key).to eq("input.group")
    expect(proto.batch_group_max_runs).to eq(3)
    expect(proto.broadcast_output).to be(true)
  end

  it "omits unset optional fields" do
    config = described_class.new(max_size: 1)
    proto = config.to_proto

    expect(proto.batch_max_size).to eq(1)
    expect(proto.batch_max_interval_ms).to eq(0)
    expect(proto.batch_group_key).to eq("")
    expect(proto.batch_group_max_runs).to eq(0)
    expect(proto.broadcast_output).to be(false)
  end

  it "raises when max_size is not positive" do
    expect { described_class.new(max_size: 0) }.to raise_error(ArgumentError)
    expect { described_class.new(max_size: -1) }.to raise_error(ArgumentError)
  end

  it "raises when max_interval_ms is provided but not positive" do
    expect { described_class.new(max_size: 1, max_interval_ms: 0) }.to raise_error(ArgumentError)
  end

  it "raises when group_max_runs is provided but not positive" do
    expect { described_class.new(max_size: 1, group_max_runs: 0) }.to raise_error(ArgumentError)
  end
end
