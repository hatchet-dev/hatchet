# frozen_string_literal: true

require "time"

RSpec.describe Hatchet::Features::Runs do
  let(:valid_token) { "eyJhbGciOiJIUzI1NiJ9.eyJzdWIiOiJ0ZXN0LXRlbmFudCJ9.signature" }
  let(:config) { Hatchet::Config.new(token: valid_token) }
  let(:rest_client) { instance_double("ApiClient") }
  let(:workflow_runs_api) { instance_double("HatchetSdkRest::WorkflowRunsApi") }
  let(:task_api) { instance_double("HatchetSdkRest::TaskApi") }
  let(:runs_client) { described_class.new(rest_client, config) }

  before do
    allow(HatchetSdkRest::WorkflowRunsApi).to receive(:new).with(rest_client).and_return(workflow_runs_api)
    allow(HatchetSdkRest::TaskApi).to receive(:new).with(rest_client).and_return(task_api)
  end

  around do |example|
    # Store original env vars
    original_env = ENV.select { |k, _| k.start_with?("HATCHET_CLIENT_") }

    # Clear all HATCHET_CLIENT_ env vars before each test
    ENV.keys.select { |k| k.start_with?("HATCHET_CLIENT_") }.each { |k| ENV.delete(k) }

    example.run

    # Restore original env vars
    ENV.keys.select { |k| k.start_with?("HATCHET_CLIENT_") }.each { |k| ENV.delete(k) }
    original_env.each { |k, v| ENV[k] = v }
  end

  describe "#initialize" do
    it "creates a new runs client with required dependencies" do
      expect(runs_client).to be_a(described_class)
      expect(runs_client.instance_variable_get(:@config)).to eq(config)
      expect(runs_client.instance_variable_get(:@rest_client)).to eq(rest_client)
    end

    it "initializes API clients" do
      # Create the client to trigger the API initialization
      described_class.new(rest_client, config)

      expect(HatchetSdkRest::WorkflowRunsApi).to have_received(:new).with(rest_client)
      expect(HatchetSdkRest::TaskApi).to have_received(:new).with(rest_client)
    end
  end

  describe "#get_task_run" do
    let(:task_run_id) { "task-123" }
    let(:task_summary) { instance_double("HatchetSdkRest::V1TaskSummary") }

    it "retrieves task run details" do
      allow(task_api).to receive(:v1_task_get).with(task_run_id).and_return(task_summary)

      result = runs_client.get_task_run(task_run_id)

      expect(result).to eq(task_summary)
      expect(task_api).to have_received(:v1_task_get).with(task_run_id)
    end
  end

  describe "#get" do
    let(:workflow_run_id) { "workflow-123" }
    let(:workflow_run) { instance_double("HatchetSdkRest::V1WorkflowRun") }
    let(:workflow_run_details) { instance_double("HatchetSdkRest::V1WorkflowRunDetails", run: workflow_run) }

    it "retrieves and unwraps the workflow run" do
      allow(workflow_runs_api).to receive(:v1_workflow_run_get).with("workflow-123").and_return(workflow_run_details)

      result = runs_client.get(workflow_run_id)

      expect(result).to eq(workflow_run)
      expect(workflow_runs_api).to have_received(:v1_workflow_run_get).with("workflow-123")
    end

    it "converts run ID to string" do
      allow(workflow_runs_api).to receive(:v1_workflow_run_get).with("123").and_return(workflow_run_details)

      runs_client.get(123)

      expect(workflow_runs_api).to have_received(:v1_workflow_run_get).with("123")
    end
  end

  describe "#get_details" do
    let(:workflow_run_id) { "workflow-123" }
    let(:workflow_run_details) { instance_double("HatchetSdkRest::V1WorkflowRunDetails") }

    it "retrieves the full workflow run details" do
      allow(workflow_runs_api).to receive(:v1_workflow_run_get).with("workflow-123").and_return(workflow_run_details)

      result = runs_client.get_details(workflow_run_id)

      expect(result).to eq(workflow_run_details)
      expect(workflow_runs_api).to have_received(:v1_workflow_run_get).with("workflow-123")
    end
  end

  describe "#get_status" do
    let(:workflow_run_id) { "workflow-123" }
    let(:task_status) { instance_double("HatchetSdkRest::V1TaskStatus") }

    it "retrieves workflow run status" do
      allow(workflow_runs_api).to receive(:v1_workflow_run_get_status).with(workflow_run_id).and_return(task_status)

      result = runs_client.get_status(workflow_run_id)

      expect(result).to eq(task_status)
      expect(workflow_runs_api).to have_received(:v1_workflow_run_get_status).with(workflow_run_id)
    end
  end

  describe "#list" do
    let(:task_summary_list) { instance_double("HatchetSdkRest::V1TaskSummaryList") }
    let(:since_time) { Time.now - (24 * 60 * 60) }
    let(:until_time) { Time.now }

    it "lists workflow runs with default parameters" do
      allow(workflow_runs_api).to receive(:v1_workflow_run_list).and_return(task_summary_list)

      result = runs_client.list

      expect(result).to eq(task_summary_list)
      expect(workflow_runs_api).to have_received(:v1_workflow_run_list).with(
        "test-tenant",
        kind_of(String),
        false,
        hash_including(
          offset: nil,
          limit: nil,
          statuses: nil,
          _until: kind_of(String),
          additional_metadata: nil,
          workflow_ids: nil,
          worker_id: nil,
          parent_task_external_id: nil,
          triggering_event_external_id: nil,
          include_payloads: true,
        ),
      )
    end

    it "lists workflow runs with custom parameters" do
      allow(workflow_runs_api).to receive(:v1_workflow_run_list).and_return(task_summary_list)

      runs_client.list(
        since: since_time,
        only_tasks: true,
        offset: 10,
        limit: 50,
        statuses: ["RUNNING"],
        until_time: until_time,
        additional_metadata: { "env" => "test" },
        workflow_ids: ["workflow-1"],
        worker_id: "worker-123",
        parent_task_external_id: "parent-task-456",
        triggering_event_external_id: "event-789",
      )

      expect(workflow_runs_api).to have_received(:v1_workflow_run_list).with(
        "test-tenant",
        since_time.utc.iso8601,
        true,
        {
          offset: 10,
          limit: 50,
          statuses: ["RUNNING"],
          _until: until_time.utc.iso8601,
          additional_metadata: ["env:test"],
          workflow_ids: ["workflow-1"],
          worker_id: "worker-123",
          parent_task_external_id: "parent-task-456",
          triggering_event_external_id: "event-789",
          include_payloads: true,
        },
      )
    end

    it "warns for large date ranges" do
      large_since = Time.now - (10 * 24 * 60 * 60) # 10 days ago
      allow(workflow_runs_api).to receive(:v1_workflow_run_list).and_return(task_summary_list)

      expect { runs_client.list(since: large_since) }.to output(/performance issues/).to_stderr
    end
  end

  describe "#list_with_pagination" do
    let(:task_summary_1) { instance_double("HatchetSdkRest::V1TaskSummary", metadata: double(id: "run-1"), created_at: Time.now - 1000) }
    let(:task_summary_2) { instance_double("HatchetSdkRest::V1TaskSummary", metadata: double(id: "run-2"), created_at: Time.now - 2000) }
    let(:task_summary_list_1) { instance_double("HatchetSdkRest::V1TaskSummaryList", rows: [task_summary_1]) }
    let(:task_summary_list_2) { instance_double("HatchetSdkRest::V1TaskSummaryList", rows: [task_summary_2]) }

    it "paginates through date ranges and returns sorted unique results" do
      since_time = Time.now - (2 * 24 * 60 * 60) # 2 days ago
      until_time = Time.now

      allow(workflow_runs_api).to receive(:v1_workflow_run_list)
        .and_return(task_summary_list_1, task_summary_list_2)

      result = runs_client.list_with_pagination(since: since_time, until_time: until_time)

      expect(result).to be_an(Array)
      expect(result.length).to eq(2)
      expect(result.first).to eq(task_summary_1) # Most recent first
      expect(result.last).to eq(task_summary_2)

      # Should make multiple API calls for date range partitioning
      expect(workflow_runs_api).to have_received(:v1_workflow_run_list).at_least(:twice)
    end

    it "handles duplicate runs across date ranges" do
      # Same run appears in multiple date ranges
      duplicate_summary = instance_double("HatchetSdkRest::V1TaskSummary", metadata: double(id: "duplicate-run"),
                                                                           created_at: Time.now - 500,)
      list_with_duplicate_1 = instance_double("HatchetSdkRest::V1TaskSummaryList", rows: [duplicate_summary])
      list_with_duplicate_2 = instance_double("HatchetSdkRest::V1TaskSummaryList", rows: [duplicate_summary])

      allow(workflow_runs_api).to receive(:v1_workflow_run_list)
        .and_return(list_with_duplicate_1, list_with_duplicate_2)

      result = runs_client.list_with_pagination(since: Time.now - (2 * 24 * 60 * 60))

      expect(result.length).to eq(1)
      expect(result.first).to eq(duplicate_summary)
    end
  end

  describe "#create" do
    let(:workflow_name) { "test-workflow" }
    let(:input) { { "key" => "value" } }
    let(:workflow_run_details) { instance_double("HatchetSdkRest::V1WorkflowRunDetails") }
    let(:workflow_run) { instance_double("HatchetSdkRest::V1WorkflowRun") }

    it "creates a workflow run with basic parameters" do
      trigger_request = instance_double("HatchetSdkRest::V1TriggerWorkflowRunRequest")
      allow(HatchetSdkRest::V1TriggerWorkflowRunRequest).to receive(:new).and_return(trigger_request)
      allow(workflow_run_details).to receive(:run).and_return(workflow_run)
      allow(workflow_runs_api).to receive(:v1_workflow_run_create).and_return(workflow_run_details)

      result = runs_client.create(name: workflow_name, input: input)

      expect(result).to eq(workflow_run)
      expect(HatchetSdkRest::V1TriggerWorkflowRunRequest).to have_received(:new).with(
        workflow_name: workflow_name,
        input: input,
        additional_metadata: nil,
        priority: nil,
      )
      expect(workflow_runs_api).to have_received(:v1_workflow_run_create).with("test-tenant", trigger_request)
    end

    it "creates a workflow run with all parameters" do
      trigger_request = instance_double("HatchetSdkRest::V1TriggerWorkflowRunRequest")
      allow(HatchetSdkRest::V1TriggerWorkflowRunRequest).to receive(:new).and_return(trigger_request)
      allow(workflow_run_details).to receive(:run).and_return(workflow_run)
      allow(workflow_runs_api).to receive(:v1_workflow_run_create).and_return(workflow_run_details)

      additional_metadata = { "source" => "api" }
      priority = 5

      runs_client.create(name: workflow_name, input: input, additional_metadata: additional_metadata, priority: priority)

      expect(HatchetSdkRest::V1TriggerWorkflowRunRequest).to have_received(:new).with(
        workflow_name: workflow_name,
        input: input,
        additional_metadata: additional_metadata,
        priority: priority,
      )
    end

    it "applies namespace to workflow name" do
      config_with_namespace = Hatchet::Config.new(token: valid_token, namespace: "prod")
      runs_client_with_ns = described_class.new(rest_client, config_with_namespace)

      allow(HatchetSdkRest::WorkflowRunsApi).to receive(:new).with(rest_client).and_return(workflow_runs_api)
      allow(HatchetSdkRest::TaskApi).to receive(:new).with(rest_client).and_return(task_api)

      trigger_request = instance_double("HatchetSdkRest::V1TriggerWorkflowRunRequest")
      allow(HatchetSdkRest::V1TriggerWorkflowRunRequest).to receive(:new).and_return(trigger_request)
      allow(workflow_run_details).to receive(:run).and_return(workflow_run)
      allow(workflow_runs_api).to receive(:v1_workflow_run_create).and_return(workflow_run_details)

      runs_client_with_ns.create(name: workflow_name, input: input)

      expect(HatchetSdkRest::V1TriggerWorkflowRunRequest).to have_received(:new).with(
        workflow_name: "prod_test-workflow",
        input: input,
        additional_metadata: nil,
        priority: nil,
      )
    end
  end

  describe "#replay" do
    let(:run_id) { "run-123" }
    let(:replay_request) { instance_double("HatchetSdkRest::V1ReplayTaskRequest") }

    it "replays a single run" do
      allow(task_api).to receive(:v1_task_replay)
      allow_any_instance_of(Hatchet::Features::BulkCancelReplayOpts).to receive(:to_replay_request).and_return(replay_request)

      runs_client.replay(run_id)

      expect(task_api).to have_received(:v1_task_replay).with("test-tenant", replay_request)
    end
  end

  describe "#bulk_replay" do
    let(:ids) { %w[run-1 run-2] }
    let(:opts) { Hatchet::Features::BulkCancelReplayOpts.new(ids: ids) }
    let(:replay_request) { instance_double("HatchetSdkRest::V1ReplayTaskRequest") }

    it "replays multiple runs in bulk" do
      allow(task_api).to receive(:v1_task_replay)
      allow(opts).to receive(:to_replay_request).and_return(replay_request)

      runs_client.bulk_replay(opts)

      expect(task_api).to have_received(:v1_task_replay).with("test-tenant", replay_request)
    end
  end

  describe "#cancel" do
    let(:run_id) { "run-123" }
    let(:cancel_request) { instance_double("HatchetSdkRest::V1CancelTaskRequest") }

    it "cancels a single run" do
      allow(task_api).to receive(:v1_task_cancel)
      allow_any_instance_of(Hatchet::Features::BulkCancelReplayOpts).to receive(:to_cancel_request).and_return(cancel_request)

      runs_client.cancel(run_id)

      expect(task_api).to have_received(:v1_task_cancel).with("test-tenant", cancel_request)
    end
  end

  describe "#bulk_cancel" do
    let(:ids) { %w[run-1 run-2] }
    let(:opts) { Hatchet::Features::BulkCancelReplayOpts.new(ids: ids) }
    let(:cancel_request) { instance_double("HatchetSdkRest::V1CancelTaskRequest") }

    it "cancels multiple runs in bulk" do
      allow(task_api).to receive(:v1_task_cancel)
      allow(opts).to receive(:to_cancel_request).and_return(cancel_request)

      runs_client.bulk_cancel(opts)

      expect(task_api).to have_received(:v1_task_cancel).with("test-tenant", cancel_request)
    end
  end

  describe "#get_result" do
    let(:run_id) { "run-123" }
    let(:workflow_run_details) { instance_double("HatchetSdkRest::V1WorkflowRunDetails") }
    let(:workflow_run) { instance_double("HatchetSdkRest::V1WorkflowRun", output: { "result" => "success" }) }

    it "retrieves workflow run result" do
      allow(workflow_run_details).to receive(:run).and_return(workflow_run)
      allow(workflow_runs_api).to receive(:v1_workflow_run_get).with(run_id).and_return(workflow_run_details)

      result = runs_client.get_result(run_id)

      expect(result).to eq({ "result" => "success" })
    end
  end

  describe "#bulk_replay_by_filters_with_pagination" do
    let(:external_ids) { (1..10).map { |i| "run-#{i}" } }

    before do
      allow(workflow_runs_api).to receive(:v1_workflow_run_external_ids_list).and_return(external_ids)
      allow(task_api).to receive(:v1_task_replay)
      allow_any_instance_of(Hatchet::Features::BulkCancelReplayOpts).to receive(:to_replay_request)
        .and_return(instance_double("HatchetSdkRest::V1ReplayTaskRequest"))
    end

    it "replays runs in chunks" do
      runs_client.bulk_replay_by_filters_with_pagination(
        sleep_time: 0,
        chunk_size: 5,
        since: Time.now - 3600,
        until_time: Time.now,
      )

      expect(workflow_runs_api).to have_received(:v1_workflow_run_external_ids_list)
      expect(task_api).to have_received(:v1_task_replay).twice
    end

    it "uses default FAILED and CANCELLED statuses" do
      runs_client.bulk_replay_by_filters_with_pagination(sleep_time: 0)

      expect(workflow_runs_api).to have_received(:v1_workflow_run_external_ids_list).with(
        "test-tenant",
        kind_of(String),
        hash_including(statuses: %w[FAILED CANCELLED]),
      )
    end
  end

  describe "#bulk_cancel_by_filters_with_pagination" do
    let(:external_ids) { (1..3).map { |i| "run-#{i}" } }

    before do
      allow(workflow_runs_api).to receive(:v1_workflow_run_external_ids_list).and_return(external_ids)
      allow(task_api).to receive(:v1_task_cancel)
      allow_any_instance_of(Hatchet::Features::BulkCancelReplayOpts).to receive(:to_cancel_request)
        .and_return(instance_double("HatchetSdkRest::V1CancelTaskRequest"))
    end

    it "cancels runs in chunks" do
      runs_client.bulk_cancel_by_filters_with_pagination(sleep_time: 0, chunk_size: 2)

      expect(workflow_runs_api).to have_received(:v1_workflow_run_external_ids_list)
      expect(task_api).to have_received(:v1_task_cancel).twice
    end

    it "uses default RUNNING and QUEUED statuses" do
      runs_client.bulk_cancel_by_filters_with_pagination(sleep_time: 0)

      expect(workflow_runs_api).to have_received(:v1_workflow_run_external_ids_list).with(
        "test-tenant",
        kind_of(String),
        hash_including(statuses: %w[RUNNING QUEUED]),
      )
    end
  end

  describe "private methods" do
    describe "#partition_date_range" do
      it "partitions date range into daily chunks" do
        since_time = Time.new(2023, 1, 1, 0, 0, 0)
        until_time = Time.new(2023, 1, 3, 0, 0, 0)

        result = runs_client.send(:partition_date_range, since: since_time, until_time: until_time)

        expect(result.length).to eq(2)
        expect(result[0]).to eq([since_time, Time.new(2023, 1, 2, 0, 0, 0)])
        expect(result[1]).to eq([Time.new(2023, 1, 2, 0, 0, 0), until_time])
      end

      it "handles single day range" do
        since_time = Time.new(2023, 1, 1, 12, 0, 0)
        until_time = Time.new(2023, 1, 1, 18, 0, 0)

        result = runs_client.send(:partition_date_range, since: since_time, until_time: until_time)

        expect(result.length).to eq(1)
        expect(result[0]).to eq([since_time, until_time])
      end
    end

    describe "#maybe_additional_metadata_to_kv" do
      it "converts hash to key-value array" do
        metadata = { "env" => "test", "version" => "1.0" }

        result = runs_client.send(:maybe_additional_metadata_to_kv, metadata)

        expect(result).to eq(["env:test", "version:1.0"])
      end

      it "returns nil for nil input" do
        result = runs_client.send(:maybe_additional_metadata_to_kv, nil)

        expect(result).to be_nil
      end

      it "converts keys and values to strings" do
        metadata = { 123 => 456 }

        result = runs_client.send(:maybe_additional_metadata_to_kv, metadata)

        expect(result).to eq(["123:456"])
      end
    end
  end
end

RSpec.describe Hatchet::Features::RunFilter do
  let(:since_time) { Time.now - 3600 }

  describe "#initialize" do
    it "creates a filter with required parameters" do
      filter = described_class.new(since: since_time)

      expect(filter.since).to eq(since_time)
      expect(filter.until_time).to be_nil
      expect(filter.statuses).to be_nil
      expect(filter.workflow_ids).to be_nil
      expect(filter.additional_metadata).to be_nil
    end

    it "creates a filter with all parameters" do
      until_time = Time.now
      statuses = %w[RUNNING COMPLETED]
      workflow_ids = %w[workflow-1 workflow-2]
      additional_metadata = { "env" => "test" }

      filter = described_class.new(
        since: since_time,
        until_time: until_time,
        statuses: statuses,
        workflow_ids: workflow_ids,
        additional_metadata: additional_metadata,
      )

      expect(filter.since).to eq(since_time)
      expect(filter.until_time).to eq(until_time)
      expect(filter.statuses).to eq(statuses)
      expect(filter.workflow_ids).to eq(workflow_ids)
      expect(filter.additional_metadata).to eq(additional_metadata)
    end
  end
end

RSpec.describe Hatchet::Features::BulkCancelReplayOpts do
  describe "#initialize" do
    it "creates options with IDs" do
      ids = %w[run-1 run-2]
      opts = described_class.new(ids: ids)

      expect(opts.ids).to eq(ids)
      expect(opts.filters).to be_nil
    end

    it "creates options with filters" do
      filter = Hatchet::Features::RunFilter.new(since: Time.now - 3600)
      opts = described_class.new(filters: filter)

      expect(opts.ids).to be_nil
      expect(opts.filters).to eq(filter)
    end

    it "raises error when neither ids nor filters provided" do
      expect { described_class.new }.to raise_error(ArgumentError, "ids or filters must be set")
    end

    it "raises error when both ids and filters provided" do
      expect do
        described_class.new(
          ids: ["run-1"],
          filters: Hatchet::Features::RunFilter.new(since: Time.now - 3600),
        )
      end.to raise_error(ArgumentError, "ids and filters cannot both be set")
    end
  end

  describe "#v1_task_filter" do
    it "returns nil when no filters present" do
      opts = described_class.new(ids: ["run-1"])

      expect(opts.v1_task_filter).to be_nil
    end

    it "creates V1TaskFilter from filters" do
      since_time = Time.now - 3600
      until_time = Time.now
      statuses = ["RUNNING"]
      workflow_ids = ["workflow-1"]
      additional_metadata = { "env" => "test" }

      filter = Hatchet::Features::RunFilter.new(
        since: since_time,
        until_time: until_time,
        statuses: statuses,
        workflow_ids: workflow_ids,
        additional_metadata: additional_metadata,
      )
      opts = described_class.new(filters: filter)

      task_filter = instance_double("HatchetSdkRest::V1TaskFilter")
      allow(HatchetSdkRest::V1TaskFilter).to receive(:new).and_return(task_filter)

      result = opts.v1_task_filter

      expect(result).to eq(task_filter)
      expect(HatchetSdkRest::V1TaskFilter).to have_received(:new).with(
        since: since_time.utc.iso8601,
        _until: until_time.utc.iso8601,
        statuses: statuses,
        workflow_ids: workflow_ids,
        additional_metadata: ["env:test"],
      )
    end
  end

  describe "#to_cancel_request" do
    it "creates cancel request with IDs" do
      ids = %w[run-1 run-2]
      opts = described_class.new(ids: ids)

      cancel_request = instance_double("HatchetSdkRest::V1CancelTaskRequest")
      allow(HatchetSdkRest::V1CancelTaskRequest).to receive(:new).and_return(cancel_request)

      result = opts.to_cancel_request

      expect(result).to eq(cancel_request)
      expect(HatchetSdkRest::V1CancelTaskRequest).to have_received(:new).with(
        external_ids: ids,
        filter: nil,
      )
    end

    it "creates cancel request with filters" do
      filter = Hatchet::Features::RunFilter.new(since: Time.now - 3600)
      opts = described_class.new(filters: filter)

      task_filter = instance_double("HatchetSdkRest::V1TaskFilter")
      allow(opts).to receive(:v1_task_filter).and_return(task_filter)

      cancel_request = instance_double("HatchetSdkRest::V1CancelTaskRequest")
      allow(HatchetSdkRest::V1CancelTaskRequest).to receive(:new).and_return(cancel_request)

      result = opts.to_cancel_request

      expect(result).to eq(cancel_request)
      expect(HatchetSdkRest::V1CancelTaskRequest).to have_received(:new).with(
        external_ids: nil,
        filter: task_filter,
      )
    end
  end

  describe "#to_replay_request" do
    it "creates replay request with IDs" do
      ids = %w[run-1 run-2]
      opts = described_class.new(ids: ids)

      replay_request = instance_double("HatchetSdkRest::V1ReplayTaskRequest")
      allow(HatchetSdkRest::V1ReplayTaskRequest).to receive(:new).and_return(replay_request)

      result = opts.to_replay_request

      expect(result).to eq(replay_request)
      expect(HatchetSdkRest::V1ReplayTaskRequest).to have_received(:new).with(
        external_ids: ids,
        filter: nil,
      )
    end

    it "creates replay request with filters" do
      filter = Hatchet::Features::RunFilter.new(since: Time.now - 3600)
      opts = described_class.new(filters: filter)

      task_filter = instance_double("HatchetSdkRest::V1TaskFilter")
      allow(opts).to receive(:v1_task_filter).and_return(task_filter)

      replay_request = instance_double("HatchetSdkRest::V1ReplayTaskRequest")
      allow(HatchetSdkRest::V1ReplayTaskRequest).to receive(:new).and_return(replay_request)

      result = opts.to_replay_request

      expect(result).to eq(replay_request)
      expect(HatchetSdkRest::V1ReplayTaskRequest).to have_received(:new).with(
        external_ids: nil,
        filter: task_filter,
      )
    end
  end
end
