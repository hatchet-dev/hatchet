module Hatchet
  module Features
  # Runs client for interacting with Hatchet workflow run management API
  #
  # This class provides a high-level interface for creating and managing workflow runs
  # in the Hatchet system. It wraps the generated REST API client with a more
  # convenient Ruby interface.
  #
  # @example Creating a workflow run
  #   runs = Features::Runs.new(rest_client, config)
  #   run_request = HatchetSdkRest::TriggerWorkflowRunRequest.new(
  #     input: { key: "value" },
  #     additional_metadata: { source: "api" }
  #   )
  #   response = runs.create("workflow-id", run_request)
  #
  # @since 0.0.0
  class Runs
    # Re-export commonly used workflow run classes for convenience
    TriggerWorkflowRunRequest = ::HatchetSdkRest::TriggerWorkflowRunRequest

    # Initializes a new Runs client instance
    #
    # @param rest_client [Object] The configured REST client for API communication
    # @param config [Hatchet::Config] The Hatchet configuration containing tenant_id and other settings
    # @return [void]
    # @since 0.0.0
    def initialize(rest_client, config)
      # @type [Object]
      @rest_client = rest_client
      # @type [Hatchet::Config]
      @config = config
      # @type [HatchetSdkRest::WorkflowRunsApi]
      @workflow_run_api = HatchetSdkRest::WorkflowRunsApi.new(rest_client)
    end

    # Creates a new workflow run in the Hatchet system
    #
    # This method triggers a new workflow or task run for the specified workflow using the
    # provided input data. The workflow run will be queued according to the
    # workflow definition on an available worker.
    #
    # @param trigger_request [HatchetSdkRest::V1TriggerWorkflowRunRequest] The workflow run request object containing
    #   workflow name, input data and optional metadata
    # @param opts [Hash] Optional parameters
    # @option opts [String] :version The workflow version. If not supplied, the latest version is fetched.
    # @return [Object] The API response containing the created workflow run details
    # @raise [ArgumentError] If the workflow or trigger_request parameters are nil or invalid
    # @raise [Hatchet::Error] If the API request fails or returns an error
    # @example Creating a workflow run
    #   trigger_request = HatchetSdkRest::V1TriggerWorkflowRunRequest.new(
    #     input: { user_id: 123, action: "process_data" },
    #     additional_metadata: { source: "api", priority: "high" }
    #   )
    #   response = runs.create("simple-workflow", trigger_request)
    # @since 0.0.2
    def create(trigger_request, opts = {})
      run = @workflow_run_api.v1_workflow_run_create(@config.tenant_id, trigger_request, opts)

      run.run
    end
  end
  end
end