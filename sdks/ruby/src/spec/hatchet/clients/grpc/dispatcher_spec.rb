# frozen_string_literal: true

require "spec_helper"

RSpec.describe Hatchet::Clients::Grpc::Dispatcher do
  let(:logger) { instance_double(Logger, info: nil, warn: nil) }
  let(:config) do
    instance_double(
      Hatchet::Config,
      logger: logger,
      auth_metadata: {},
    )
  end
  let(:channel) { instance_double("GRPC::Core::Channel") }
  let(:grpc_stub) { instance_double(Dispatcher::Stub) }
  let(:response) { instance_double(WorkerRegisterResponse, worker_id: "worker-123") }

  subject(:dispatcher) { described_class.new(config: config, channel: channel) }

  before do
    dispatcher.instance_variable_set(:@stub, grpc_stub)
    allow(grpc_stub).to receive(:register).and_return(response)
  end

  describe "#register" do
    it "sends slot_config during worker registration" do
      dispatcher.register(
        name: "ruby-worker",
        actions: ["svc:durable_task"],
        slots: 10,
        slot_config: { "default" => 10, "durable" => 3 },
      )

      expect(grpc_stub).to have_received(:register) do |request, metadata:|
        expect(request.slot_config).to eq({ "default" => 10, "durable" => 3 })
        expect(metadata).to eq({})
      end
    end

    it "falls back to legacy registration without slot_config on gRPC errors" do
      allow(grpc_stub).to receive(:register).and_raise(GRPC::InvalidArgument).once.and_return(response)

      dispatcher.register(
        name: "ruby-worker",
        actions: ["svc:durable_task"],
        slots: 10,
        slot_config: { "default" => 10, "durable" => 3 },
      )

      expect(grpc_stub).to have_received(:register).twice
      expect(logger).to have_received(:warn).with(%r{without runtime_info/slot_config after GRPC::InvalidArgument})
    end
  end
end
