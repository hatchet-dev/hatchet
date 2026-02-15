# frozen_string_literal: true

RSpec.describe Hatchet::Features::Cron do
  let(:valid_token) { "eyJhbGciOiJIUzI1NiJ9.eyJzdWIiOiJ0ZXN0LXRlbmFudCJ9.signature" }
  let(:config) { Hatchet::Config.new(token: valid_token) }
  let(:rest_client) { instance_double("ApiClient") }
  let(:workflow_api) { instance_double("HatchetSdkRest::WorkflowApi") }
  let(:workflow_run_api) { instance_double("HatchetSdkRest::WorkflowRunApi") }
  let(:cron_client) { described_class.new(rest_client, config) }

  before do
    allow(HatchetSdkRest::WorkflowApi).to receive(:new).with(rest_client).and_return(workflow_api)
    allow(HatchetSdkRest::WorkflowRunApi).to receive(:new).with(rest_client).and_return(workflow_run_api)
  end

  around do |example|
    original_env = ENV.select { |k, _| k.start_with?("HATCHET_CLIENT_") }
    ENV.keys.select { |k| k.start_with?("HATCHET_CLIENT_") }.each { |k| ENV.delete(k) }
    example.run
    ENV.keys.select { |k| k.start_with?("HATCHET_CLIENT_") }.each { |k| ENV.delete(k) }
    original_env.each { |k, v| ENV[k] = v }
  end

  describe "#initialize" do
    it "creates a new cron client with required dependencies" do
      expect(cron_client).to be_a(described_class)
      expect(cron_client.instance_variable_get(:@config)).to eq(config)
    end

    it "initializes API clients" do
      described_class.new(rest_client, config)
      expect(HatchetSdkRest::WorkflowApi).to have_received(:new).with(rest_client)
      expect(HatchetSdkRest::WorkflowRunApi).to have_received(:new).with(rest_client)
    end
  end

  describe "#create" do
    let(:cron_request) { instance_double("HatchetSdkRest::CreateCronWorkflowTriggerRequest") }
    let(:created_cron) { instance_double("Object") }

    before do
      allow(HatchetSdkRest::CreateCronWorkflowTriggerRequest).to receive(:new).and_return(cron_request)
      allow(workflow_run_api).to receive(:cron_workflow_trigger_create).and_return(created_cron)
    end

    it "creates a cron trigger with valid expression" do
      result = cron_client.create(
        workflow_name: "my-workflow",
        cron_name: "daily-run",
        expression: "0 0 * * *",
        input: { key: "value" },
        additional_metadata: { source: "api" },
      )

      expect(result).to eq(created_cron)
      expect(HatchetSdkRest::CreateCronWorkflowTriggerRequest).to have_received(:new).with(
        cron_name: "daily-run",
        cron_expression: "0 0 * * *",
        input: { key: "value" },
        additional_metadata: { source: "api" },
        priority: nil,
      )
      expect(workflow_run_api).to have_received(:cron_workflow_trigger_create).with(
        "test-tenant", "my-workflow", cron_request,
      )
    end

    it "applies namespace to workflow_name" do
      config_with_ns = Hatchet::Config.new(token: valid_token, namespace: "prod_")
      client_with_ns = described_class.new(rest_client, config_with_ns)

      client_with_ns.create(
        workflow_name: "my-workflow",
        cron_name: "daily-run",
        expression: "0 0 * * *",
      )

      expect(workflow_run_api).to have_received(:cron_workflow_trigger_create).with(
        "test-tenant", "prod_my-workflow", cron_request,
      )
    end

    it "creates a cron trigger with priority" do
      cron_client.create(
        workflow_name: "my-workflow",
        cron_name: "daily-run",
        expression: "0 0 * * *",
        priority: 5,
      )

      expect(HatchetSdkRest::CreateCronWorkflowTriggerRequest).to have_received(:new).with(
        hash_including(priority: 5),
      )
    end

    it "accepts cron aliases" do
      cron_client.create(
        workflow_name: "my-workflow",
        cron_name: "daily-run",
        expression: "@daily",
      )

      expect(HatchetSdkRest::CreateCronWorkflowTriggerRequest).to have_received(:new).with(
        hash_including(cron_expression: "@daily"),
      )
    end

    it "raises error for empty expression" do
      expect do
        cron_client.create(
          workflow_name: "my-workflow",
          cron_name: "daily-run",
          expression: "",
        )
      end.to raise_error(ArgumentError, "Cron expression is required")
    end

    it "raises error for invalid expression format" do
      expect do
        cron_client.create(
          workflow_name: "my-workflow",
          cron_name: "daily-run",
          expression: "invalid",
        )
      end.to raise_error(ArgumentError, /Cron expression must have 5 parts/)
    end

    it "raises error for invalid cron parts" do
      expect do
        cron_client.create(
          workflow_name: "my-workflow",
          cron_name: "daily-run",
          expression: "abc * * * *",
        )
      end.to raise_error(ArgumentError, /Invalid cron expression part/)
    end
  end

  describe "#delete" do
    it "deletes a cron trigger by ID" do
      allow(workflow_api).to receive(:workflow_cron_delete)

      cron_client.delete("cron-123")

      expect(workflow_api).to have_received(:workflow_cron_delete).with("test-tenant", "cron-123")
    end
  end

  describe "#list" do
    let(:cron_list) { instance_double("Object") }

    before do
      allow(workflow_api).to receive(:cron_workflow_list).and_return(cron_list)
    end

    it "lists cron triggers with default parameters" do
      result = cron_client.list

      expect(result).to eq(cron_list)
      expect(workflow_api).to have_received(:cron_workflow_list).with(
        "test-tenant",
        {
          offset: nil,
          limit: nil,
          workflow_id: nil,
          additional_metadata: nil,
          order_by_field: nil,
          order_by_direction: nil,
          workflow_name: nil,
          cron_name: nil,
        },
      )
    end

    it "lists cron triggers with custom parameters" do
      cron_client.list(
        offset: 10,
        limit: 50,
        workflow_id: "wf-1",
        additional_metadata: { "env" => "prod" },
        workflow_name: "my-workflow",
        cron_name: "daily",
      )

      expect(workflow_api).to have_received(:cron_workflow_list).with(
        "test-tenant",
        hash_including(
          offset: 10,
          limit: 50,
          workflow_id: "wf-1",
          additional_metadata: [{ key: "env", value: "prod" }],
          workflow_name: "my-workflow",
          cron_name: "daily",
        ),
      )
    end
  end

  describe "#get" do
    let(:cron_details) { instance_double("Object") }

    it "retrieves a cron trigger by ID" do
      allow(workflow_api).to receive(:workflow_cron_get).with("test-tenant", "cron-123").and_return(cron_details)

      result = cron_client.get("cron-123")

      expect(result).to eq(cron_details)
    end
  end
end
