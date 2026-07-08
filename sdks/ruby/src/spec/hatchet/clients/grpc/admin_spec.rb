# frozen_string_literal: true

require "spec_helper"

RSpec.describe Hatchet::Clients::Grpc::Admin do
  let(:logger) { instance_double(Logger, info: nil, warn: nil, debug: nil) }
  let(:config) do
    instance_double(
      Hatchet::Config,
      logger: logger,
      auth_metadata: {},
    )
  end
  let(:channel) { instance_double("GRPC::Core::Channel") }
  let(:v0_stub) { instance_double(WorkflowService::Stub) }
  let(:response) { double("TriggerWorkflowResponse", workflow_run_id: "run-1") }

  subject(:admin) { described_class.new(config: config, channel: channel) }

  before do
    allow(config).to receive(:apply_namespace) { |name| name }
    allow(admin).to receive(:ensure_connected!)
    admin.instance_variable_set(:@v0_stub, v0_stub)
    allow(v0_stub).to receive(:trigger_workflow).and_return(response)
  end

  describe "#trigger_workflow" do
    it "maps display_name onto the trigger request" do
      admin.trigger_workflow("my-workflow", input: {}, options: { display_name: "Acme Corp" })

      expect(v0_stub).to have_received(:trigger_workflow) do |request, metadata:|
        expect(request.display_name).to eq("Acme Corp")
        expect(metadata).to eq({})
      end
    end

    it "leaves display_name unset when not provided" do
      admin.trigger_workflow("my-workflow", input: {})

      expect(v0_stub).to have_received(:trigger_workflow) do |request, _metadata|
        expect(request.display_name).to eq("")
      end
    end
  end

  describe "#bulk_trigger_workflow" do
    let(:bulk_response) { double("BulkTriggerWorkflowResponse", workflow_run_ids: %w[run-1 run-2]) }

    before do
      allow(v0_stub).to receive(:bulk_trigger_workflow).and_return(bulk_response)
    end

    it "maps a per-item display_name onto each request" do
      admin.bulk_trigger_workflow("my-workflow", [
        { input: {}, options: { display_name: "Alpha" } },
        { input: {}, options: { display_name: "Bravo" } },
      ],)

      expect(v0_stub).to have_received(:bulk_trigger_workflow) do |bulk_request, metadata:|
        names = bulk_request.workflows.map(&:display_name)
        expect(names).to eq(%w[Alpha Bravo])
        expect(metadata).to eq({})
      end
    end
  end
end
