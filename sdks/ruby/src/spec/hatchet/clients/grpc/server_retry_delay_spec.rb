# frozen_string_literal: true

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
  def call_with(*errors)
    calls = 0
    result = described_class.call do
      calls += 1
      raise errors[calls - 1] if calls <= errors.length

      :ok
    end
    [result, calls]
  end

  before { allow(described_class).to receive(:sleep) }

  it "waits the server's RetryInfo delay before retrying" do
    result, calls = call_with(throttled(nanos: 300_000_000))

    expect(result).to eq(:ok)
    expect(calls).to eq(2)
    expect(described_class).to have_received(:sleep).with(0.3).once
  end

  it "raises RESOURCE_EXHAUSTED without RetryInfo at once" do
    quota = GRPC::ResourceExhausted.new("tenant has reached its task runs limit")

    expect { call_with(quota) }.to raise_error(quota)
    expect(described_class).not_to have_received(:sleep)
  end

  it "raises RESOURCE_EXHAUSTED with a zero RetryInfo delay at once" do
    zero = throttled(seconds: 0)

    expect { call_with(zero) }.to raise_error(zero)
    expect(described_class).not_to have_received(:sleep)
  end

  it "raises other codes at once even when they carry RetryInfo" do
    unavailable = GRPC::Unavailable.new("down", throttled(seconds: 1).metadata)

    expect { call_with(unavailable) }.to raise_error(unavailable)
    expect(described_class).not_to have_received(:sleep)
  end

  it "does not carry a delay over to a later attempt without RetryInfo" do
    quota = GRPC::ResourceExhausted.new("quota")

    expect { call_with(throttled(seconds: 1), quota) }.to raise_error(quota)
    expect(described_class).to have_received(:sleep).with(1.0).once
  end

  it "caps a longer server delay" do
    _, calls = call_with(throttled(seconds: 600))

    expect(calls).to eq(2)
    expect(described_class::MAX_DELAY_SECONDS).to eq(5.0)
    expect(described_class).to have_received(:sleep).with(described_class::MAX_DELAY_SECONDS).once
  end

  it "stops after MAX_ATTEMPTS and raises the last error" do
    errors = Array.new(described_class::MAX_ATTEMPTS) { throttled(nanos: 10_000_000) }

    expect { call_with(*errors) }.to raise_error(errors.last)
    expect(described_class).to have_received(:sleep).exactly(described_class::MAX_ATTEMPTS - 1).times
  end

  describe "at the call sites" do
    let(:logger) { instance_double(Logger, info: nil, warn: nil, debug: nil) }
    let(:config) do
      instance_double(Hatchet::Config, logger: logger, auth_metadata: {}).tap do |c|
        allow(c).to receive(:apply_namespace) { |name, **| name }
      end
    end
    let(:channel) { instance_double("GRPC::Core::Channel") }

    it "retries a throttled trigger and returns the run id" do
      admin = Hatchet::Clients::Grpc::Admin.new(config: config, channel: channel)
      v0_stub = instance_double(WorkflowService::Stub)
      admin.instance_variable_set(:@v0_stub, v0_stub)
      admin.instance_variable_set(:@v1_stub, instance_double(V1::AdminService::Stub))

      responses = [throttled(nanos: 200_000_000), TriggerWorkflowResponse.new(workflow_run_id: "run-1")]
      allow(v0_stub).to receive(:trigger_workflow) do
        response = responses.shift
        raise response if response.is_a?(Exception)

        response
      end

      expect(admin.trigger_workflow("wf")).to eq("run-1")
      expect(v0_stub).to have_received(:trigger_workflow).twice
      expect(described_class).to have_received(:sleep).with(0.2).once
    end

    it "raises ResourceExhaustedError once a throttled trigger runs out of attempts" do
      admin = Hatchet::Clients::Grpc::Admin.new(config: config, channel: channel)
      v0_stub = instance_double(WorkflowService::Stub)
      admin.instance_variable_set(:@v0_stub, v0_stub)
      admin.instance_variable_set(:@v1_stub, instance_double(V1::AdminService::Stub))
      allow(v0_stub).to receive(:trigger_workflow) { raise throttled(seconds: 1) }

      expect { admin.trigger_workflow("wf") }.to raise_error(Hatchet::ResourceExhaustedError)
      expect(v0_stub).to have_received(:trigger_workflow).exactly(described_class::MAX_ATTEMPTS).times
    end

    it "retries a throttled event push" do
      events = Hatchet::Clients::Grpc::EventClient.new(config: config, channel: channel)
      stub = instance_double(EventsService::Stub)
      events.instance_variable_set(:@stub, stub)

      responses = [throttled(seconds: 1), Event.new(event_id: "event-1")]
      allow(stub).to receive(:push) do
        response = responses.shift
        raise response if response.is_a?(Exception)

        response
      end

      expect(events.push(key: "user:create", payload: {}).event_id).to eq("event-1")
      expect(stub).to have_received(:push).twice
    end
  end
end
