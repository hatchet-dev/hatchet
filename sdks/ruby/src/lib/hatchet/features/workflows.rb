# frozen_string_literal: true

module Hatchet
  module Features
    # Workflows client for managing workflow definitions within Hatchet
    #
    # Note that workflows are the declaration, _not_ the individual runs.
    # If you're looking for runs, use the Runs client instead.
    #
    # @example Getting a workflow
    #   workflow = hatchet.workflows.get("workflow-id")
    #
    # @example Listing workflows
    #   workflows = hatchet.workflows.list(workflow_name: "my-workflow", limit: 10)
    #
    # @since 0.1.0
    class Workflows
      # Initializes a new Workflows client instance
      #
      # @param rest_client [Object] The configured REST client for API communication
      # @param config [Hatchet::Config] The Hatchet configuration containing tenant_id and other settings
      # @return [void]
      # @since 0.1.0
      def initialize(rest_client, config)
        @rest_client = rest_client
        @config = config
        @workflow_api = HatchetSdkRest::WorkflowApi.new(rest_client)
      end

      # Get a workflow by its ID
      #
      # @param workflow_id [String] The ID of the workflow to retrieve
      # @return [Object] The workflow details
      # @raise [HatchetSdkRest::ApiError] If the API request fails
      # @example
      #   workflow = hatchet.workflows.get("workflow-123")
      def get(workflow_id)
        @workflow_api.workflow_get(workflow_id)
      end

      # List all workflows in the tenant matching optional filters
      #
      # @param workflow_name [String, nil] The name of the workflow to filter by (namespace will be applied)
      # @param limit [Integer, nil] The maximum number of items to return
      # @param offset [Integer, nil] The offset to start the list from
      # @return [Object] A list of workflows
      # @raise [HatchetSdkRest::ApiError] If the API request fails
      # @example
      #   workflows = hatchet.workflows.list(workflow_name: "my-workflow", limit: 10, offset: 0)
      def list(workflow_name: nil, limit: nil, offset: nil)
        @workflow_api.workflow_list(
          @config.tenant_id,
          {
            limit: limit,
            offset: offset,
            name: workflow_name ? @config.apply_namespace(workflow_name) : nil,
          },
        )
      end

      # Get a workflow version by the workflow ID and an optional version
      #
      # @param workflow_id [String] The ID of the workflow to retrieve the version for
      # @param version [String, nil] The version to retrieve. If nil, the latest version is returned
      # @return [Object] The workflow version
      # @raise [HatchetSdkRest::ApiError] If the API request fails
      # @example
      #   version = hatchet.workflows.get_version("workflow-123", version: "v2")
      def get_version(workflow_id, version: nil)
        @workflow_api.workflow_version_get(workflow_id, { version: version })
      end

      # Permanently delete a workflow
      #
      # **DANGEROUS: This will delete a workflow and all of its data**
      #
      # @param workflow_id [String] The ID of the workflow to delete
      # @return [void]
      # @raise [HatchetSdkRest::ApiError] If the API request fails
      # @example
      #   hatchet.workflows.delete("workflow-123")
      def delete(workflow_id)
        @workflow_api.workflow_delete(workflow_id)
      end

      # Pause a workflow. While paused, new runs of the workflow are queued but not started.
      #
      # @param workflow_id [String] The ID of the workflow to pause
      # @param queue_ttl [Integer, String] How long runs stay queued while the workflow is paused before they are dropped, in seconds or as a duration string (e.g. "1h30m")
      # @param paused_workflow_cron_run_queue_behavior [String] The behavior of cron runs triggered while the workflow is paused ("QUEUE" or "DROP")
      # @param paused_workflow_scheduled_run_queue_behavior [String] The behavior of scheduled runs triggered while the workflow is paused ("QUEUE" or "DROP")
      # @return [HatchetSdkRest::Workflow] The updated workflow
      # @raise [HatchetSdkRest::ApiError] If the API request fails
      # @example
      #   hatchet.workflows.pause("workflow-123", queue_ttl: 3600)
      def pause(
        workflow_id,
        queue_ttl:,
        paused_workflow_cron_run_queue_behavior: HatchetSdkRest::WorkflowPauseScheduledCronRunQueueBehavior::QUEUE,
        paused_workflow_scheduled_run_queue_behavior: HatchetSdkRest::WorkflowPauseScheduledCronRunQueueBehavior::QUEUE
      )
        pause_request = HatchetSdkRest::PauseWorkflowRequestPause.new(
          action: "pause",
          paused_workflow_queue_ttl: queue_ttl.is_a?(String) ? queue_ttl : "#{queue_ttl}s",
          paused_workflow_cron_run_queue_behavior: paused_workflow_cron_run_queue_behavior,
          paused_workflow_scheduled_run_queue_behavior: paused_workflow_scheduled_run_queue_behavior,
        )

        @workflow_api.workflow_update(
          workflow_id,
          HatchetSdkRest::WorkflowUpdateRequest.new(pause: pause_request),
        )
      end

      # Unpause a workflow
      #
      # @param workflow_id [String] The ID of the workflow to unpause
      # @return [HatchetSdkRest::Workflow] The updated workflow
      # @raise [HatchetSdkRest::ApiError] If the API request fails
      # @example
      #   hatchet.workflows.unpause("workflow-123")
      def unpause(workflow_id)
        @workflow_api.workflow_update(
          workflow_id,
          HatchetSdkRest::WorkflowUpdateRequest.new(
            pause: HatchetSdkRest::PauseWorkflowRequestUnpause.new(action: "unpause"),
          ),
        )
      end
    end
  end
end
