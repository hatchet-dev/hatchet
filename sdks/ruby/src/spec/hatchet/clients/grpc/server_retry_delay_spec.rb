# frozen_string_literal: true

require "json"
require "spec_helper"

RSpec.describe Hatchet::Clients::Grpc::ServerRetryDelay do
  def throttled(seconds: 0, nanos: 0)
    retry_info = Google::Rpc::RetryInfo.new(retry_delay: Google::Protobuf::Duration.new(seconds: seconds, nanos: nanos))
    status = Google::Rpc::Status.new(
      code: GRPC::Core::StatusCodes::RESOURCE_EXHAUSTED,
      message: "throttled",
      details: [Google::Protobuf::Any.pack(retry_info)],
    )
    GRPC::ResourceExhausted.new("throttled", { "grpc-status-details-bin" => Google::Rpc::Status.encode(status) })
  end

  # Runs ServerRetryDelay.call over a block that raises the given errors in order, then returns :ok.
  # Returns what the call returned or raised, and how many times the block ran.
  def call_with(*errors)
    calls = 0
    result = described_class.call do
      calls += 1
      raise errors[calls - 1] if calls <= errors.length

      :ok
    end
    [result, calls]
  rescue GRPC::BadStatus => e
    [e, calls]
  end

  let(:waits) { [] }

  before { allow(described_class).to receive(:sleep) { |seconds| waits << seconds } }

  # The rule every Hatchet SDK follows for a throttled call, shared with the other SDKs' tests.
  scenarios_path = File.expand_path(
    "../../../../../../../api-contracts/sdk-conformance/throttled_calls.json",
    __dir__,
  )
  JSON.parse(File.read(scenarios_path)).fetch("scenarios").each do |scenario|
    it scenario.fetch("name") do
      errors = scenario.fetch("delays_ms").map do |ms|
        throttled(seconds: ms / 1000, nanos: (ms % 1000) * 1_000_000)
      end
      expected_waits = scenario.fetch("waits_ms").map { |ms| be_within(1e-9).of(ms / 1000.0) }

      outcome, calls = call_with(*errors)

      expect(waits).to match(expected_waits)
      expect(calls).to eq(expected_waits.length + 1)
      if scenario.fetch("succeeds")
        expect(outcome).to eq(:ok)
      else
        expect(outcome).to be_a(GRPC::ResourceExhausted)
      end
    end
  end

  it "raises RESOURCE_EXHAUSTED without RetryInfo at once" do
    quota = GRPC::ResourceExhausted.new("tenant has reached its task runs limit")

    expect(call_with(quota)).to eq([quota, 1])
    expect(waits).to be_empty
  end

  it "raises RESOURCE_EXHAUSTED with a zero RetryInfo delay at once" do
    zero = throttled(seconds: 0)

    expect(call_with(zero)).to eq([zero, 1])
    expect(waits).to be_empty
  end

  it "raises other codes at once even when they carry RetryInfo" do
    unavailable = GRPC::Unavailable.new("down", throttled(seconds: 1).metadata)

    expect(call_with(unavailable)).to eq([unavailable, 1])
    expect(waits).to be_empty
  end
end
