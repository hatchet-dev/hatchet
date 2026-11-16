# frozen_string_literal: true

require "spec_helper"

RSpec.describe Hatchet::Clients::Grpc::Admin do
  let(:logger) { instance_double(Logger, info: nil, warn: nil, debug: nil) }
  let(:config) { instance_double(Hatchet::Config, logger: logger, auth_metadata: {}) }
  let(:channel) { instance_double("GRPC::Core::Channel") }
  let(:v0_stub) { instance_double(WorkflowService::Stub) }
  let(:v1_stub) { instance_double(V1::AdminService::Stub) }

  subject(:admin) { described_class.new(config: config, channel: channel) }

  before do
    admin.instance_variable_set(:@v0_stub, v0_stub)
    admin.instance_variable_set(:@v1_stub, v1_stub)
  end

  describe "#pause_rate_limit" do
    it "sends the key and time and returns the time the engine reports" do
      until_time = Time.utc(2030, 1, 1, 0, 0, 0.5r)
      response = PauseRateLimitResponse.new(
        paused_until: Google::Protobuf::Timestamp.new(seconds: Time.utc(2030, 1, 1, 1).to_i, nanos: 0),
      )
      allow(v0_stub).to receive(:pause_rate_limit).and_return(response)

      result = admin.pause_rate_limit(key: "upstream-api:tenant-123", until_time: until_time)

      expect(v0_stub).to have_received(:pause_rate_limit) do |request, metadata:|
        expect(request.key).to eq("upstream-api:tenant-123")
        expect(request.paused_until.seconds).to eq(until_time.to_i)
        expect(request.paused_until.nanos).to eq(500_000_000)
        expect(metadata).to eq({})
      end
      expect(result).to eq(Time.utc(2030, 1, 1, 1))
    end
  end
end
