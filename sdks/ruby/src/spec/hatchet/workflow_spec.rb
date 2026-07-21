# frozen_string_literal: true

require "spec_helper"

RSpec.describe Hatchet::Workflow do
  let(:config) { instance_double(Hatchet::Config, apply_namespace: "test-service") }

  describe "#display_name" do
    it "threads a workflow-level CEL expression into CreateWorkflowVersionRequest" do
      workflow = described_class.new(name: "CustomerWorkflow", display_name: "input.customerName")
      workflow.task(:step) { |_input, _ctx| nil }

      proto = workflow.to_proto(config)

      expect(proto.display_name).to eq("input.customerName")
      expect(proto.has_display_name?).to be(true)
    end

    it "leaves display_name unset on the request when not provided" do
      workflow = described_class.new(name: "PlainWorkflow")
      workflow.task(:step) { |_input, _ctx| nil }

      proto = workflow.to_proto(config)

      expect(proto.has_display_name?).to be(false)
    end

    it "names an individual DAG step from a per-task expression" do
      workflow = described_class.new(name: "DagWorkflow", display_name: "input.run")
      workflow.task(:enrich, display_name: "'enrich-' + input.name") { |_input, _ctx| nil }

      proto = workflow.to_proto(config)

      expect(proto.display_name).to eq("input.run")
      expect(proto.tasks.first.display_name).to eq("'enrich-' + input.name")
    end
  end
end
