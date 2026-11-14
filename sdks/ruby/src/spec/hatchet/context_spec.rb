# frozen_string_literal: true

require "spec_helper"

RSpec.describe Hatchet::Context do
  describe "#pause_rate_limit" do
    let(:admin_grpc) { instance_double(Hatchet::Clients::Grpc::Admin) }
    let(:client) { instance_double(Hatchet::Client, admin_grpc: admin_grpc) }

    it "pauses the key through the client's admin client" do
      until_time = Time.utc(2030, 1, 1)
      allow(admin_grpc).to receive(:pause_rate_limit).and_return(until_time)
      context = described_class.new(workflow_run_id: "run-1", step_run_id: "step-1", client: client)

      result = context.pause_rate_limit("upstream-api:tenant-123", until_time: until_time)

      expect(admin_grpc).to have_received(:pause_rate_limit).with(key: "upstream-api:tenant-123", until_time: until_time)
      expect(result).to eq(until_time)
    end

    it "raises when the context has no client" do
      context = described_class.new(workflow_run_id: "run-1", step_run_id: "step-1")

      expect { context.pause_rate_limit("upstream-api", until_time: Time.now) }.to raise_error(Hatchet::Error)
    end
  end

  describe "#was_skipped?" do
    let(:context) do
      described_class.new(
        workflow_run_id: "run-1",
        step_run_id: "step-1",
        parent_outputs: {
          "skipped_parent" => { "skipped" => true },
          "ran_parent" => { "random_number" => 3 },
        },
      )
    end

    it "is true for a parent the engine marked as skipped" do
      expect(context.was_skipped?(:skipped_parent)).to be(true)
    end

    it "is false for a parent that ran" do
      expect(context.was_skipped?(:ran_parent)).to be(false)
    end

    it "is false for a task that is not among the parent outputs" do
      expect(context.was_skipped?(:unknown)).to be(false)
    end
  end
end
