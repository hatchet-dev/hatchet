# frozen_string_literal: true

require "time"

module Hatchet
  module Features
    # Scheduled client for managing scheduled workflows within Hatchet
    #
    # This class provides a high-level interface for creating, deleting,
    # updating, bulk operations, listing, and retrieving scheduled workflow runs.
    #
    # @example Creating a scheduled workflow
    #   scheduled = scheduled_client.create(
    #     workflow_name: "my-workflow",
    #     trigger_at: Time.now + 3600,
    #     input: { key: "value" },
    #     additional_metadata: { source: "api" }
    #   )
    #
    # @since 0.1.0
    class Scheduled
      # Initializes a new Scheduled client instance
      #
      # @param rest_client [Object] The configured REST client for API communication
      # @param config [Hatchet::Config] The Hatchet configuration containing tenant_id and other settings
      # @return [void]
      # @since 0.1.0
      def initialize(rest_client, config)
        @rest_client = rest_client
        @config = config
        @workflow_api = HatchetSdkRest::WorkflowApi.new(rest_client)
        @workflow_run_api = HatchetSdkRest::WorkflowRunApi.new(rest_client)
      end

      # Create a new scheduled workflow run
      #
      # IMPORTANT: It's preferable to use Workflow.run to trigger workflows if possible.
      # This method is intended to be an escape hatch.
      #
      # @param workflow_name [String] The name of the workflow to schedule (namespace will be applied)
      # @param trigger_at [Time] The datetime when the run should be triggered
      # @param input [Hash] The input data for the scheduled workflow
      # @param additional_metadata [Hash] Additional metadata associated with the future run
      # @return [Object] The created scheduled workflow instance
      # @raise [HatchetSdkRest::ApiError] If the API request fails
      # @example
      #   scheduled = scheduled_client.create(
      #     workflow_name: "my-workflow",
      #     trigger_at: Time.now + 3600,
      #     input: { key: "value" },
      #     additional_metadata: { source: "api" }
      #   )
      def create(workflow_name:, trigger_at:, input: {}, additional_metadata: {})
        request = HatchetSdkRest::ScheduleWorkflowRunRequest.new(
          trigger_at: trigger_at.utc.iso8601,
          input: input,
          additional_metadata: additional_metadata,
        )

        @workflow_run_api.scheduled_workflow_run_create(
          @config.tenant_id,
          @config.apply_namespace(workflow_name),
          request,
        )
      end

      # Delete a scheduled workflow run by its ID
      #
      # @param scheduled_id [String] The ID of the scheduled workflow run to delete
      # @return [void]
      # @raise [HatchetSdkRest::ApiError] If the API request fails
      # @example
      #   scheduled_client.delete("scheduled-123")
      def delete(scheduled_id)
        @workflow_api.workflow_scheduled_delete(@config.tenant_id, scheduled_id)
      end

      # Reschedule a scheduled workflow run by its ID
      #
      # Note: the server may reject rescheduling if the scheduled run has already
      # triggered, or if it was created via code definition (not via API).
      #
      # @param scheduled_id [String] The ID of the scheduled workflow run to reschedule
      # @param trigger_at [Time] The new datetime when the run should be triggered
      # @return [Object] The updated scheduled workflow instance
      # @raise [HatchetSdkRest::ApiError] If the API request fails
      # @example
      #   scheduled_client.update("scheduled-123", trigger_at: Time.now + 7200)
      def update(scheduled_id, trigger_at:)
        request = HatchetSdkRest::UpdateScheduledWorkflowRunRequest.new(
          trigger_at: trigger_at.utc.iso8601,
        )

        @workflow_api.workflow_scheduled_update(@config.tenant_id, scheduled_id, request)
      end

      # Bulk delete scheduled workflow runs
      #
      # Provide either scheduled_ids (explicit list) or one or more filter fields.
      #
      # @param scheduled_ids [Array<String>, nil] Explicit list of scheduled workflow run IDs to delete
      # @param workflow_id [String, nil] Filter by workflow ID
      # @param parent_workflow_run_id [String, nil] Filter by parent workflow run ID
      # @param parent_step_run_id [String, nil] Filter by parent step run ID
      # @param statuses [Array<String>, nil] Filter by scheduled run statuses (warning: may not be supported)
      # @param additional_metadata [Hash, nil] Filter by additional metadata key/value pairs
      # @return [Object] The bulk delete response containing deleted IDs and per-item errors
      # @raise [ArgumentError] If neither scheduled_ids nor any filter field is provided
      # @raise [HatchetSdkRest::ApiError] If the API request fails
      def bulk_delete(scheduled_ids: nil, workflow_id: nil, parent_workflow_run_id: nil,
                      parent_step_run_id: nil, statuses: nil, additional_metadata: nil)
        warn "The 'statuses' filter is not supported for bulk delete and will be ignored." if statuses

        has_filter = [workflow_id, parent_workflow_run_id, parent_step_run_id, additional_metadata].any? { |v| !v.nil? }

        raise ArgumentError, "bulk_delete requires either scheduled_ids or at least one filter field." unless scheduled_ids || has_filter

        filter_obj = nil
        if has_filter
          filter_obj = HatchetSdkRest::ScheduledWorkflowsBulkDeleteFilter.new(
            workflow_id: workflow_id,
            parent_workflow_run_id: parent_workflow_run_id,
            parent_step_run_id: parent_step_run_id,
            additional_metadata: maybe_additional_metadata_to_kv(additional_metadata),
          )
        end

        request = HatchetSdkRest::ScheduledWorkflowsBulkDeleteRequest.new(
          scheduled_workflow_run_ids: scheduled_ids,
          filter: filter_obj,
        )

        @workflow_api.workflow_scheduled_bulk_delete(@config.tenant_id, request)
      end

      # Bulk reschedule scheduled workflow runs
      #
      # @param updates [Array<Hash>] Array of hashes with :id and :trigger_at keys
      # @return [Object] The bulk update response containing updated IDs and per-item errors
      # @raise [HatchetSdkRest::ApiError] If the API request fails
      # @example
      #   scheduled_client.bulk_update([
      #     { id: "scheduled-1", trigger_at: Time.now + 3600 },
      #     { id: "scheduled-2", trigger_at: Time.now + 7200 }
      #   ])
      def bulk_update(updates)
        update_items = updates.map do |u|
          HatchetSdkRest::ScheduledWorkflowsBulkUpdateItem.new(
            id: u[:id],
            trigger_at: u[:trigger_at].utc.iso8601,
          )
        end

        request = HatchetSdkRest::ScheduledWorkflowsBulkUpdateRequest.new(
          updates: update_items,
        )

        @workflow_api.workflow_scheduled_bulk_update(@config.tenant_id, request)
      end

      # List scheduled workflows based on provided filters
      #
      # @param offset [Integer, nil] The offset to use in pagination
      # @param limit [Integer, nil] The maximum number of scheduled workflows to return
      # @param workflow_id [String, nil] The ID of the workflow to filter by
      # @param parent_workflow_run_id [String, nil] The ID of the parent workflow run to filter by
      # @param statuses [Array<String>, nil] A list of statuses to filter by
      # @param additional_metadata [Hash, nil] Additional metadata to filter by
      # @param order_by_field [String, nil] The field to order the results by
      # @param order_by_direction [String, nil] The direction to order the results by
      # @return [Object] A list of scheduled workflows matching the provided filters
      # @raise [HatchetSdkRest::ApiError] If the API request fails
      # @example
      #   scheduled = scheduled_client.list(limit: 10, workflow_id: "wf-1")
      def list(offset: nil, limit: nil, workflow_id: nil, parent_workflow_run_id: nil,
               statuses: nil, additional_metadata: nil, order_by_field: nil, order_by_direction: nil)
        @workflow_api.workflow_scheduled_list(
          @config.tenant_id,
          {
            offset: offset,
            limit: limit,
            order_by_field: order_by_field,
            order_by_direction: order_by_direction,
            workflow_id: workflow_id,
            additional_metadata: maybe_additional_metadata_to_kv(additional_metadata),
            parent_workflow_run_id: parent_workflow_run_id,
            statuses: statuses,
          },
        )
      end

      # Retrieve a specific scheduled workflow by ID
      #
      # @param scheduled_id [String] The scheduled workflow trigger ID to retrieve
      # @return [Object] The requested scheduled workflow instance
      # @raise [HatchetSdkRest::ApiError] If the API request fails
      # @example
      #   scheduled = scheduled_client.get("scheduled-123")
      def get(scheduled_id)
        @workflow_api.workflow_scheduled_get(@config.tenant_id, scheduled_id)
      end

      private

      # Convert additional metadata hash to key-value array format expected by API
      #
      # @param metadata [Hash<String, String>, nil] Metadata hash
      # @return [Array<Hash>, nil] Array of {key: string, value: string} objects
      def maybe_additional_metadata_to_kv(metadata)
        return nil unless metadata

        metadata.map { |k, v| "#{k}:#{v}" }
      end
    end
  end
end
