# frozen_string_literal: true

require "time"

RSpec.describe Hatchet::Features::Scheduled do
  let(:valid_token) { "eyJhbGciOiJIUzI1NiJ9.eyJzdWIiOiJ0ZXN0LXRlbmFudCJ9.signature" }
  let(:config) { Hatchet::Config.new(token: valid_token) }
  let(:rest_client) { instance_double("ApiClient") }
  let(:workflow_api) { instance_double("HatchetSdkRest::WorkflowApi") }
  let(:workflow_run_api) { instance_double("HatchetSdkRest::WorkflowRunApi") }
  let(:scheduled_client) { described_class.new(rest_client, config) }

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
    it "creates a new scheduled client with required dependencies" do
      expect(scheduled_client).to be_a(described_class)
      expect(scheduled_client.instance_variable_get(:@config)).to eq(config)
    end

    it "initializes API clients" do
      described_class.new(rest_client, config)
      expect(HatchetSdkRest::WorkflowApi).to have_received(:new).with(rest_client)
      expect(HatchetSdkRest::WorkflowRunApi).to have_received(:new).with(rest_client)
    end
  end

  describe "#create" do
    let(:schedule_request) { instance_double("HatchetSdkRest::ScheduleWorkflowRunRequest") }
    let(:created_scheduled) { instance_double("Object") }
    let(:trigger_at) { Time.now + 3600 }

    before do
      allow(HatchetSdkRest::ScheduleWorkflowRunRequest).to receive(:new).and_return(schedule_request)
      allow(workflow_run_api).to receive(:scheduled_workflow_run_create).and_return(created_scheduled)
    end

    it "creates a scheduled workflow run" do
      result = scheduled_client.create(
        workflow_name: "my-workflow",
        trigger_at: trigger_at,
        input: { key: "value" },
        additional_metadata: { source: "api" },
      )

      expect(result).to eq(created_scheduled)
      expect(HatchetSdkRest::ScheduleWorkflowRunRequest).to have_received(:new).with(
        trigger_at: trigger_at.utc.iso8601,
        input: { key: "value" },
        additional_metadata: { source: "api" },
      )
      expect(workflow_run_api).to have_received(:scheduled_workflow_run_create).with(
        "test-tenant", "my-workflow", schedule_request,
      )
    end

    it "applies namespace to workflow_name" do
      config_with_ns = Hatchet::Config.new(token: valid_token, namespace: "prod_")
      client_with_ns = described_class.new(rest_client, config_with_ns)

      client_with_ns.create(
        workflow_name: "my-workflow",
        trigger_at: trigger_at,
      )

      expect(workflow_run_api).to have_received(:scheduled_workflow_run_create).with(
        "test-tenant", "prod_my-workflow", schedule_request,
      )
    end
  end

  describe "#delete" do
    it "deletes a scheduled workflow run by ID" do
      allow(workflow_api).to receive(:workflow_scheduled_delete)

      scheduled_client.delete("scheduled-123")

      expect(workflow_api).to have_received(:workflow_scheduled_delete).with("test-tenant", "scheduled-123")
    end
  end

  describe "#update" do
    let(:update_request) { instance_double("HatchetSdkRest::UpdateScheduledWorkflowRunRequest") }
    let(:updated_scheduled) { instance_double("Object") }
    let(:new_trigger_at) { Time.now + 7200 }

    it "reschedules a scheduled workflow run" do
      allow(HatchetSdkRest::UpdateScheduledWorkflowRunRequest).to receive(:new).and_return(update_request)
      allow(workflow_api).to receive(:workflow_scheduled_update).and_return(updated_scheduled)

      result = scheduled_client.update("scheduled-123", trigger_at: new_trigger_at)

      expect(result).to eq(updated_scheduled)
      expect(HatchetSdkRest::UpdateScheduledWorkflowRunRequest).to have_received(:new).with(
        trigger_at: new_trigger_at.utc.iso8601,
      )
      expect(workflow_api).to have_received(:workflow_scheduled_update).with(
        "test-tenant", "scheduled-123", update_request,
      )
    end
  end

  describe "#bulk_delete" do
    let(:bulk_delete_request) { instance_double("HatchetSdkRest::ScheduledWorkflowsBulkDeleteRequest") }
    let(:bulk_delete_response) { instance_double("Object") }

    before do
      allow(HatchetSdkRest::ScheduledWorkflowsBulkDeleteRequest).to receive(:new).and_return(bulk_delete_request)
      allow(workflow_api).to receive(:workflow_scheduled_bulk_delete).and_return(bulk_delete_response)
    end

    it "bulk deletes by scheduled IDs" do
      allow(HatchetSdkRest::ScheduledWorkflowsBulkDeleteRequest).to receive(:new).and_return(bulk_delete_request)

      result = scheduled_client.bulk_delete(scheduled_ids: %w[s-1 s-2])

      expect(result).to eq(bulk_delete_response)
      expect(HatchetSdkRest::ScheduledWorkflowsBulkDeleteRequest).to have_received(:new).with(
        scheduled_workflow_run_ids: %w[s-1 s-2],
        filter: nil,
      )
    end

    it "bulk deletes by filter fields" do
      filter_obj = instance_double("HatchetSdkRest::ScheduledWorkflowsBulkDeleteFilter")
      allow(HatchetSdkRest::ScheduledWorkflowsBulkDeleteFilter).to receive(:new).and_return(filter_obj)

      scheduled_client.bulk_delete(workflow_id: "wf-1")

      expect(HatchetSdkRest::ScheduledWorkflowsBulkDeleteFilter).to have_received(:new).with(
        workflow_id: "wf-1",
        parent_workflow_run_id: nil,
        parent_step_run_id: nil,
        additional_metadata: nil,
      )
      expect(HatchetSdkRest::ScheduledWorkflowsBulkDeleteRequest).to have_received(:new).with(
        scheduled_workflow_run_ids: nil,
        filter: filter_obj,
      )
    end

    it "raises error when neither IDs nor filters provided" do
      expect { scheduled_client.bulk_delete }.to raise_error(
        ArgumentError, "bulk_delete requires either scheduled_ids or at least one filter field.",
      )
    end

    it "warns when statuses filter is used" do
      expect { scheduled_client.bulk_delete(scheduled_ids: ["s-1"], statuses: ["PENDING"]) }
        .to output(/statuses/).to_stderr
    end
  end

  describe "#bulk_update" do
    let(:update_item) { instance_double("HatchetSdkRest::ScheduledWorkflowsBulkUpdateItem") }
    let(:bulk_update_request) { instance_double("HatchetSdkRest::ScheduledWorkflowsBulkUpdateRequest") }
    let(:bulk_update_response) { instance_double("Object") }
    let(:trigger_at_1) { Time.now + 3600 }
    let(:trigger_at_2) { Time.now + 7200 }

    it "bulk updates scheduled workflow runs" do
      allow(HatchetSdkRest::ScheduledWorkflowsBulkUpdateItem).to receive(:new).and_return(update_item)
      allow(HatchetSdkRest::ScheduledWorkflowsBulkUpdateRequest).to receive(:new).and_return(bulk_update_request)
      allow(workflow_api).to receive(:workflow_scheduled_bulk_update).and_return(bulk_update_response)

      updates = [
        { id: "s-1", trigger_at: trigger_at_1 },
        { id: "s-2", trigger_at: trigger_at_2 },
      ]

      result = scheduled_client.bulk_update(updates)

      expect(result).to eq(bulk_update_response)
      expect(HatchetSdkRest::ScheduledWorkflowsBulkUpdateItem).to have_received(:new).twice
      expect(HatchetSdkRest::ScheduledWorkflowsBulkUpdateRequest).to have_received(:new).with(
        updates: [update_item, update_item],
      )
    end
  end

  describe "#list" do
    let(:scheduled_list) { instance_double("Object") }

    before do
      allow(workflow_api).to receive(:workflow_scheduled_list).and_return(scheduled_list)
    end

    it "lists scheduled workflows with default parameters" do
      result = scheduled_client.list

      expect(result).to eq(scheduled_list)
      expect(workflow_api).to have_received(:workflow_scheduled_list).with(
        "test-tenant",
        {
          offset: nil,
          limit: nil,
          order_by_field: nil,
          order_by_direction: nil,
          workflow_id: nil,
          additional_metadata: nil,
          parent_workflow_run_id: nil,
          statuses: nil,
        },
      )
    end

    it "lists scheduled workflows with custom parameters" do
      scheduled_client.list(
        offset: 10,
        limit: 50,
        workflow_id: "wf-1",
        additional_metadata: { "env" => "prod" },
        statuses: ["PENDING"],
      )

      expect(workflow_api).to have_received(:workflow_scheduled_list).with(
        "test-tenant",
        hash_including(
          offset: 10,
          limit: 50,
          workflow_id: "wf-1",
          additional_metadata: ["env:prod"],
          statuses: ["PENDING"],
        ),
      )
    end
  end

  describe "#get" do
    let(:scheduled_details) { instance_double("Object") }

    it "retrieves a scheduled workflow by ID" do
      allow(workflow_api).to receive(:workflow_scheduled_get).with("test-tenant", "scheduled-123").and_return(scheduled_details)

      result = scheduled_client.get("scheduled-123")

      expect(result).to eq(scheduled_details)
    end
  end
end
