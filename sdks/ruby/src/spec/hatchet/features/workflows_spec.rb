# frozen_string_literal: true

RSpec.describe Hatchet::Features::Workflows do
  let(:valid_token) { "eyJhbGciOiJIUzI1NiJ9.eyJzdWIiOiJ0ZXN0LXRlbmFudCJ9.signature" }
  let(:config) { Hatchet::Config.new(token: valid_token) }
  let(:rest_client) { instance_double("ApiClient") }
  let(:workflow_api) { instance_double("HatchetSdkRest::WorkflowApi") }
  let(:workflows_client) { described_class.new(rest_client, config) }

  before do
    allow(HatchetSdkRest::WorkflowApi).to receive(:new).with(rest_client).and_return(workflow_api)
  end

  around do |example|
    original_env = ENV.select { |k, _| k.start_with?("HATCHET_CLIENT_") }
    ENV.keys.select { |k| k.start_with?("HATCHET_CLIENT_") }.each { |k| ENV.delete(k) }
    example.run
    ENV.keys.select { |k| k.start_with?("HATCHET_CLIENT_") }.each { |k| ENV.delete(k) }
    original_env.each { |k, v| ENV[k] = v }
  end

  describe "#initialize" do
    it "creates a new workflows client with required dependencies" do
      expect(workflows_client).to be_a(described_class)
      expect(workflows_client.instance_variable_get(:@config)).to eq(config)
      expect(workflows_client.instance_variable_get(:@rest_client)).to eq(rest_client)
    end

    it "initializes workflow API client" do
      described_class.new(rest_client, config)
      expect(HatchetSdkRest::WorkflowApi).to have_received(:new).with(rest_client)
    end
  end

  describe "#get" do
    let(:workflow_id) { "workflow-123" }
    let(:workflow_details) { instance_double("Object") }

    it "retrieves a workflow by ID" do
      allow(workflow_api).to receive(:workflow_get).with(workflow_id).and_return(workflow_details)

      result = workflows_client.get(workflow_id)

      expect(result).to eq(workflow_details)
      expect(workflow_api).to have_received(:workflow_get).with(workflow_id)
    end
  end

  describe "#list" do
    let(:workflow_list) { instance_double("Object") }

    before do
      allow(workflow_api).to receive(:workflow_list).and_return(workflow_list)
    end

    it "lists workflows with default parameters" do
      result = workflows_client.list

      expect(result).to eq(workflow_list)
      expect(workflow_api).to have_received(:workflow_list).with(
        "test-tenant",
        { limit: nil, offset: nil, name: nil },
      )
    end

    it "lists workflows with custom parameters" do
      workflows_client.list(workflow_name: "my-workflow", limit: 10, offset: 5)

      expect(workflow_api).to have_received(:workflow_list).with(
        "test-tenant",
        { limit: 10, offset: 5, name: "my-workflow" },
      )
    end

    it "applies namespace to workflow_name" do
      config_with_ns = Hatchet::Config.new(token: valid_token, namespace: "prod_")
      client_with_ns = described_class.new(rest_client, config_with_ns)

      client_with_ns.list(workflow_name: "my-workflow")

      expect(workflow_api).to have_received(:workflow_list).with(
        "test-tenant",
        { limit: nil, offset: nil, name: "prod_my-workflow" },
      )
    end
  end

  describe "#get_version" do
    let(:workflow_id) { "workflow-123" }
    let(:workflow_version) { instance_double("Object") }

    it "retrieves latest workflow version by default" do
      allow(workflow_api).to receive(:workflow_version_get).with(workflow_id, { version: nil }).and_return(workflow_version)

      result = workflows_client.get_version(workflow_id)

      expect(result).to eq(workflow_version)
    end

    it "retrieves a specific workflow version" do
      allow(workflow_api).to receive(:workflow_version_get).with(workflow_id, { version: "v2" }).and_return(workflow_version)

      result = workflows_client.get_version(workflow_id, version: "v2")

      expect(result).to eq(workflow_version)
    end
  end

  describe "#delete" do
    let(:workflow_id) { "workflow-123" }

    it "deletes a workflow by ID" do
      allow(workflow_api).to receive(:workflow_delete).with(workflow_id)

      workflows_client.delete(workflow_id)

      expect(workflow_api).to have_received(:workflow_delete).with(workflow_id)
    end
  end
end
