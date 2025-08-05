# frozen_string_literal: true

require 'time'

module Hatchet
  module Features
    # Filter options for listing task runs
    class RunFilter
      attr_accessor :since, :until_time, :statuses, :workflow_ids, :additional_metadata

      def initialize(since:, until_time: nil, statuses: nil, workflow_ids: nil, additional_metadata: nil)
        @since = since
        @until_time = until_time
        @statuses = statuses
        @workflow_ids = workflow_ids
        @additional_metadata = additional_metadata
      end
    end

    # Options for bulk cancel and replay operations
    class BulkCancelReplayOpts
      attr_accessor :ids, :filters

      def initialize(ids: nil, filters: nil)
        raise ArgumentError, "ids or filters must be set" if !ids && !filters
        raise ArgumentError, "ids and filters cannot both be set" if ids && filters
        
        @ids = ids
        @filters = filters
      end

      def v1_task_filter
        return nil unless @filters

        HatchetSdkRest::V1TaskFilter.new(
          since: @filters.since,
          until: @filters.until_time,
          statuses: @filters.statuses,
          workflow_ids: @filters.workflow_ids,
          additional_metadata: maybe_additional_metadata_to_kv(@filters.additional_metadata)
        )
      end

      def to_cancel_request
        HatchetSdkRest::V1CancelTaskRequest.new(
          external_ids: @ids,
          filter: v1_task_filter
        )
      end

      def to_replay_request
        HatchetSdkRest::V1ReplayTaskRequest.new(
          external_ids: @ids,
          filter: v1_task_filter
        )
      end

      private

      def maybe_additional_metadata_to_kv(metadata)
        return nil unless metadata
        metadata.map { |k, v| { key: k.to_s, value: v.to_s } }
      end
    end

    # Runs client for interacting with Hatchet workflow run management API
    #
    # This class provides a high-level interface for creating and managing workflow runs
    # in the Hatchet system. It wraps the generated REST API client with a more
    # convenient Ruby interface.
    #
    # @example Creating a workflow run
    #   runs = Features::Runs.new(rest_client, config)
    #   response = runs.create(
    #     "my-workflow",
    #     { key: "value" },
    #     additional_metadata: { source: "api" }
    #   )
    #
    # @since 0.0.0
    class Runs
      # Re-export commonly used workflow run classes for convenience
      TriggerWorkflowRunRequest = ::HatchetSdkRest::V1TriggerWorkflowRunRequest
      WorkflowRunDetails = ::HatchetSdkRest::V1WorkflowRunDetails
      TaskSummary = ::HatchetSdkRest::V1TaskSummary
      TaskSummaryList = ::HatchetSdkRest::V1TaskSummaryList
      TaskStatus = ::HatchetSdkRest::V1TaskStatus
      
      DEFAULT_SINCE_DAYS = 1
      LARGE_DATE_RANGE_WARNING_DAYS = 7

      # Initializes a new Runs client instance
      #
      # @param rest_client [Object] The configured REST client for API communication
      # @param config [Hatchet::Config] The Hatchet configuration containing tenant_id and other settings
      # @return [void]
      # @since 0.0.0
      def initialize(rest_client, config)
        @rest_client = rest_client
        @config = config
        @workflow_runs_api = HatchetSdkRest::WorkflowRunsApi.new(rest_client)
        @task_api = HatchetSdkRest::TaskApi.new(rest_client)
      end

      # Get task run details for a given task run ID
      #
      # @param task_run_id [String] The ID of the task run to retrieve details for
      # @return [HatchetSdkRest::V1TaskSummary] Task run details for the specified task run ID
      # @raise [Hatchet::Error] If the API request fails or returns an error
      def get_task_run(task_run_id)
        @task_api.v1_task_get(task_run_id)
      end

      # Get workflow run details for a given workflow run ID
      #
      # @param workflow_run_id [String] The ID of the workflow run to retrieve details for
      # @return [HatchetSdkRest::V1WorkflowRunDetails] Workflow run details for the specified workflow run ID
      # @raise [Hatchet::Error] If the API request fails or returns an error
      def get(workflow_run_id)
        @workflow_runs_api.v1_workflow_run_get(workflow_run_id.to_s)
      end

      # Get workflow run status for a given workflow run ID
      #
      # @param workflow_run_id [String] The ID of the workflow run to retrieve status for
      # @return [HatchetSdkRest::V1TaskStatus] The task status
      # @raise [Hatchet::Error] If the API request fails or returns an error
      def get_status(workflow_run_id)
        @workflow_runs_api.v1_workflow_run_get_status(workflow_run_id)
      end

      # List task runs according to a set of filters, paginating through days
      #
      # @param since [Time, nil] The start time for filtering task runs
      # @param only_tasks [Boolean] Whether to only list task runs
      # @param offset [Integer, nil] The offset for pagination
      # @param limit [Integer, nil] The maximum number of task runs to return
      # @param statuses [Array<HatchetSdkRest::V1TaskStatus>, nil] The statuses to filter task runs by
      # @param until_time [Time, nil] The end time for filtering task runs
      # @param additional_metadata [Hash<String, String>, nil] Additional metadata to filter task runs by
      # @param workflow_ids [Array<String>, nil] The workflow IDs to filter task runs by
      # @param worker_id [String, nil] The worker ID to filter task runs by
      # @param parent_task_external_id [String, nil] The parent task external ID to filter task runs by
      # @param triggering_event_external_id [String, nil] The event id that triggered the task run
      # @return [Array<HatchetSdkRest::V1TaskSummary>] A list of task runs matching the specified filters
      # @raise [Hatchet::Error] If the API request fails or returns an error
      def list_with_pagination(
        since: nil,
        only_tasks: false,
        offset: nil,
        limit: nil,
        statuses: nil,
        until_time: nil,
        additional_metadata: nil,
        workflow_ids: nil,
        worker_id: nil,
        parent_task_external_id: nil,
        triggering_event_external_id: nil
      )
        date_ranges = partition_date_range(
          since: since || (Time.now - DEFAULT_SINCE_DAYS * 24 * 60 * 60),
          until_time: until_time || Time.now
        )

        responses = date_ranges.map do |start_time, end_time|
          @workflow_runs_api.v1_workflow_run_list(
            @config.tenant_id,
            start_time.utc.iso8601,
            only_tasks,
            {
              offset: offset,
              limit: limit,
              statuses: statuses,
              until: end_time.utc.iso8601,
              additional_metadata: maybe_additional_metadata_to_kv(additional_metadata),
              workflow_ids: workflow_ids,
              worker_id: worker_id,
              parent_task_external_id: parent_task_external_id,
              triggering_event_external_id: triggering_event_external_id
            }
          )
        end

        # Hack for uniqueness
        run_id_to_run = {}
        responses.each do |record|
          record.rows.each do |run|
            run_id_to_run[run.metadata.id] = run
          end
        end

        run_id_to_run.values.sort_by(&:created_at).reverse
      end

      # List task runs according to a set of filters
      #
      # @param since [Time, nil] The start time for filtering task runs
      # @param only_tasks [Boolean] Whether to only list task runs
      # @param offset [Integer, nil] The offset for pagination
      # @param limit [Integer, nil] The maximum number of task runs to return
      # @param statuses [Array<HatchetSdkRest::V1TaskStatus>, nil] The statuses to filter task runs by
      # @param until_time [Time, nil] The end time for filtering task runs
      # @param additional_metadata [Hash<String, String>, nil] Additional metadata to filter task runs by
      # @param workflow_ids [Array<String>, nil] The workflow IDs to filter task runs by
      # @param worker_id [String, nil] The worker ID to filter task runs by
      # @param parent_task_external_id [String, nil] The parent task external ID to filter task runs by
      # @param triggering_event_external_id [String, nil] The event id that triggered the task run
      # @return [HatchetSdkRest::V1TaskSummaryList] A list of task runs matching the specified filters
      # @raise [Hatchet::Error] If the API request fails or returns an error
      def list(
        since: nil,
        only_tasks: false,
        offset: nil,
        limit: nil,
        statuses: nil,
        until_time: nil,
        additional_metadata: nil,
        workflow_ids: nil,
        worker_id: nil,
        parent_task_external_id: nil,
        triggering_event_external_id: nil
      )
        since = since || (Time.now - DEFAULT_SINCE_DAYS * 24 * 60 * 60)
        until_time = until_time || Time.now

        if (until_time - since) / (24 * 60 * 60) >= LARGE_DATE_RANGE_WARNING_DAYS
          warn "Listing runs with a date range longer than #{LARGE_DATE_RANGE_WARNING_DAYS} days may result in performance issues. " +
               "Consider using `list_with_pagination` instead."
        end

        @workflow_runs_api.v1_workflow_run_list(
          @config.tenant_id,
          since.utc.iso8601,
          only_tasks,
          {
            offset: offset,
            limit: limit,
            statuses: statuses,
            until: until_time.utc.iso8601,
            additional_metadata: maybe_additional_metadata_to_kv(additional_metadata),
            workflow_ids: workflow_ids,
            worker_id: worker_id,
            parent_task_external_id: parent_task_external_id,
            triggering_event_external_id: triggering_event_external_id
          }
        )
      end

      # Creates a new workflow run in the Hatchet system
      #
      # This method triggers a new workflow or task run for the specified workflow using the
      # provided input data. The workflow run will be queued according to the
      # workflow definition on an available worker.
      #
      # IMPORTANT: It's preferable to use `Workflow.run` (and similar) to trigger workflows if possible. 
      # This method is intended to be an escape hatch.
      #
      # @param workflow_name [String] The name of the workflow to trigger
      # @param input [Hash] The input data for the workflow run
      # @param additional_metadata [Hash, nil] Additional metadata associated with the workflow run
      # @param priority [Integer, nil] The priority of the workflow run
      # @return [HatchetSdkRest::V1WorkflowRunDetails] The details of the triggered workflow run
      # @raise [ArgumentError] If the workflow_name or input parameters are nil or invalid
      # @raise [Hatchet::Error] If the API request fails or returns an error
      # @example Creating a workflow run
      #   response = runs.create(
      #     "simple-workflow",
      #     { user_id: 123, action: "process_data" },
      #     additional_metadata: { source: "api", priority: "high" }
      #   )
      def create(workflow_name, input, additional_metadata: nil, priority: nil)
        trigger_request = HatchetSdkRest::V1TriggerWorkflowRunRequest.new(
          workflow_name: @config.apply_namespace(workflow_name),
          input: input,
          additional_metadata: additional_metadata,
          priority: priority
        )

        @workflow_runs_api.v1_workflow_run_create(@config.tenant_id, trigger_request)
      end

      # Replay a task or workflow run
      #
      # @param run_id [String] The external ID of the task or workflow run to replay
      # @return [void]
      # @raise [Hatchet::Error] If the API request fails or returns an error
      def replay(run_id)
        bulk_replay(BulkCancelReplayOpts.new(ids: [run_id]))
      end

      # Replay task or workflow runs in bulk, according to a set of filters
      #
      # @param opts [BulkCancelReplayOpts] Options for bulk replay, including filters and IDs
      # @return [void]
      # @raise [Hatchet::Error] If the API request fails or returns an error
      def bulk_replay(opts)
        @task_api.v1_task_replay(
          @config.tenant_id,
          opts.to_replay_request
        )
      end

      # Cancel a task or workflow run
      #
      # @param run_id [String] The external ID of the task or workflow run to cancel
      # @return [void]
      # @raise [Hatchet::Error] If the API request fails or returns an error
      def cancel(run_id)
        bulk_cancel(BulkCancelReplayOpts.new(ids: [run_id]))
      end

      # Cancel task or workflow runs in bulk, according to a set of filters
      #
      # @param opts [BulkCancelReplayOpts] Options for bulk cancel, including filters and IDs
      # @return [void]
      # @raise [Hatchet::Error] If the API request fails or returns an error
      def bulk_cancel(opts)
        @task_api.v1_task_cancel(
          @config.tenant_id,
          opts.to_cancel_request
        )
      end

      # Get the result of a workflow run by its external ID
      #
      # @param run_id [String] The external ID of the workflow run to retrieve the result for
      # @return [Hash] The result of the workflow run
      # @raise [Hatchet::Error] If the API request fails or returns an error
      def get_result(run_id)
        details = get(run_id)
        details.run.output
      end

      private

      # Partition a date range into daily chunks to avoid API limits
      #
      # @param since [Time] Start time
      # @param until_time [Time] End time
      # @return [Array<Array<Time, Time>>] Array of [start_time, end_time] pairs
      def partition_date_range(since:, until_time:)
        ranges = []
        current = since
        
        while current < until_time
          next_day = [current + 24 * 60 * 60, until_time].min
          ranges << [current, next_day]
          current = next_day
        end
        
        ranges
      end

      # Convert additional metadata hash to key-value array format expected by API
      #
      # @param metadata [Hash<String, String>, nil] Metadata hash
      # @return [Array<Hash>, nil] Array of {key: string, value: string} objects
      def maybe_additional_metadata_to_kv(metadata)
        return nil unless metadata
        metadata.map { |k, v| { key: k.to_s, value: v.to_s } }
      end
    end
  end
end