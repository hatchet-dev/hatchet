# frozen_string_literal: true

require "time"
require_relative "../support/integration_helper"

RSpec.describe "Hatchet::Features::Runs Integration", :integration do
  let(:client) { create_test_client }
  let(:runs_client) { client.runs }

  describe "API connectivity and basic operations" do
    it "can list workflow runs without error" do
      expect { runs_client.list(limit: 5) }.not_to raise_error
    end

    it "returns a TaskSummaryList when listing runs" do
      result = runs_client.list(limit: 5)
      expect(result).to be_a(HatchetSdkRest::V1TaskSummaryList)
      expect(result).to respond_to(:rows)
      expect(result).to respond_to(:pagination)
    end

    it "can list runs with pagination without error" do
      since_time = Time.now - 7 * 24 * 60 * 60  # 7 days ago
      expect { runs_client.list_with_pagination(since: since_time, limit: 10) }.not_to raise_error
    end

    it "returns an array when using list_with_pagination" do
      since_time = Time.now - 24 * 60 * 60  # 1 day ago
      result = runs_client.list_with_pagination(since: since_time, limit: 5)
      expect(result).to be_an(Array)
      # Each item should be a task summary
      result.each do |item|
        expect(item).to be_a(HatchetSdkRest::V1TaskSummary)
      end
    end

    it "can filter runs by various parameters" do
      since_time = Time.now - 24 * 60 * 60  # 1 day ago

      expect do
        runs_client.list(
          since: since_time,
          only_tasks: true,
          limit: 5,
          additional_metadata: { "test" => "integration" }
        )
      end.not_to raise_error
    end

    it "handles empty results gracefully" do
      # Use a very specific time range that likely has no results
      since_time = Time.new(2020, 1, 1)
      until_time = Time.new(2020, 1, 2)

      result = runs_client.list(since: since_time, until_time: until_time, limit: 1)
      expect(result).to be_a(HatchetSdkRest::V1TaskSummaryList)
      expect(result.rows).to be_an(Array)
    end
  end

  describe "workflow run operations" do
    let(:test_workflow_run_id) { get_test_workflow_run_id(runs_client) }

    it "can retrieve workflow run details" do
      expect { runs_client.get(test_workflow_run_id) }.not_to raise_error
    end

    it "returns WorkflowRunDetails when getting a workflow run" do
      result = runs_client.get(test_workflow_run_id)
      expect(result).to be_a(HatchetSdkRest::V1WorkflowRunDetails)
      expect(result).to respond_to(:run)
      expect(result).to respond_to(:task_events)
      expect(result).to respond_to(:tasks)
    end

    it "can get workflow run status" do
      expect { runs_client.get_status(test_workflow_run_id) }.not_to raise_error
    end

    it "returns a task status when getting status" do
      result = runs_client.get_status(test_workflow_run_id)
      expect(result).to be_a(String)
      expect(result).to match(/PENDING|QUEUED|RUNNING|SUCCEEDED|FAILED|CANCELLED|COMPLETED/)
    end

    it "can get workflow run result" do
      expect { runs_client.get_result(test_workflow_run_id) }.not_to raise_error
    end
  end

  describe "task run operations" do
    let(:test_task_run_id) { get_test_task_run_id(runs_client) }

    it "can retrieve task run details" do
      expect { runs_client.get_task_run(test_task_run_id) }.not_to raise_error
    end

    it "returns TaskSummary when getting a task run" do
      result = runs_client.get_task_run(test_task_run_id)
      expect(result).to be_a(HatchetSdkRest::V1TaskSummary)
      expect(result).to respond_to(:metadata)
      expect(result).to respond_to(:status)
    end
  end

  describe "workflow creation (if allowed)" do
    # These tests will only work if the tenant allows API workflow creation
    # and you have a test workflow defined

    it "can attempt to create a workflow run (may fail if no test workflow exists)" do
      # Use a generic workflow name that might exist for testing
      test_workflow_name = "simple"
      test_input = { "test" => true, "timestamp" => Time.now.to_i }

      safely_attempt_operation("Create workflow run") do
        result = runs_client.create(test_workflow_name, test_input)
        expect(result).to be_a(HatchetSdkRest::V1WorkflowRunDetails)
        result
      end
    end
  end

  describe "bulk operations (use with caution)" do
    # These tests are more cautious since they could affect real data

    it "can create BulkCancelReplayOpts objects" do
      # Test the helper classes work correctly
      opts_with_ids = Hatchet::Features::BulkCancelReplayOpts.new(ids: ["test-id"])
      expect(opts_with_ids.ids).to eq(["test-id"])
      expect(opts_with_ids.filters).to be_nil

      filter = Hatchet::Features::RunFilter.new(since: Time.now - 3600)
      opts_with_filters = Hatchet::Features::BulkCancelReplayOpts.new(filters: filter)
      expect(opts_with_filters.filters).to eq(filter)
      expect(opts_with_filters.ids).to be_nil
    end

    # Note: We don't actually test bulk cancel/replay operations in integration tests
    # as they could affect real workflow runs. The structure validation above
    # combined with unit tests should be sufficient.
  end

  describe "error handling" do
    it "raises appropriate errors for invalid workflow run IDs" do
      invalid_id = "non-existent-workflow-run-id"

      expect { runs_client.get(invalid_id) }.to raise_error(StandardError)
    end

    it "raises appropriate errors for invalid task run IDs" do
      invalid_id = "non-existent-task-run-id"

      expect { runs_client.get_task_run(invalid_id) }.to raise_error(StandardError)
    end

    it "handles invalid date ranges gracefully" do
      # Future date range should return empty results, not error
      future_since = Time.now + 24 * 60 * 60
      future_until = Time.now + 48 * 60 * 60

      expect do
        result = runs_client.list(since: future_since, until_time: future_until, limit: 1)
        expect(result.rows).to be_empty
      end.not_to raise_error
    end
  end

  describe "response data structure validation" do
    it "validates TaskSummaryList structure" do
      result = runs_client.list(limit: 1)

      expect(result).to be_a(HatchetSdkRest::V1TaskSummaryList)
      expect(result.rows).to be_an(Array)
      expect(result.pagination).not_to be_nil

      if result.rows.any?
        task_summary = result.rows.first
        expect(task_summary).to be_a(HatchetSdkRest::V1TaskSummary)
        expect(task_summary.metadata).not_to be_nil
        expect(task_summary.status).not_to be_nil
        expect(task_summary.created_at).to be_a(Time)

        expect(task_summary.metadata.id).to be_a(String)
      end
    end

    it "validates WorkflowRunDetails structure when retrieving a run" do
      recent_runs = runs_client.list(limit: 1)
      skip "No runs available for structure validation" if recent_runs.rows.empty?

      run_id = recent_runs.rows.first.metadata.id
      result = runs_client.get(run_id)

      expect(result).to be_a(HatchetSdkRest::V1WorkflowRunDetails)
      expect(result.run).not_to be_nil
      expect(result.task_events).to be_an(Array)

      expect(result.run.metadata).not_to be_nil
      expect(result.run.status).not_to be_nil
    end
  end

  describe "configuration and client setup" do
    it "uses the correct tenant ID from configuration" do
      expect(client.config.tenant_id).not_to be_empty
      expect(client.config.token).not_to be_empty
    end

    it "can access the rest client" do
      expect(client.rest_client).not_to be_nil
    end

    it "initializes runs client correctly" do
      expect(runs_client).to be_a(Hatchet::Features::Runs)
      expect(runs_client.instance_variable_get(:@config)).to eq(client.config)
    end
  end
end
