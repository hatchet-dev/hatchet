# frozen_string_literal: true

module Hatchet
  module Features
    # Workflows client for managing workflow definitions within Hatchet
    #
    # Note that workflows are the declaration, _not_ the individual runs.
    # If you're looking for runs, use the Runs client instead.
    #
    # @example Getting a workflow
    #   workflow = workflows_client.get("workflow-id")
    #
    # @example Listing workflows
    #   workflows = workflows_client.list(workflow_name: "my-workflow", limit: 10)
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
      #   workflow = workflows_client.get("workflow-123")
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
      #   workflows = workflows_client.list(workflow_name: "my-workflow", limit: 10, offset: 0)
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
      #   version = workflows_client.get_version("workflow-123", version: "v2")
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
      #   workflows_client.delete("workflow-123")
      def delete(workflow_id)
        @workflow_api.workflow_delete(workflow_id)
      end
    end
  end
end
