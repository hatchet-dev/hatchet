# frozen_string_literal: true

module Hatchet
  module Features
    # Workers client for managing workers programmatically within Hatchet
    #
    # This class provides a high-level interface for retrieving, listing, and
    # updating workers in the Hatchet system.
    #
    # @example Getting a worker
    #   worker = workers_client.get("worker-id")
    #
    # @example Listing all workers
    #   workers = workers_client.list
    #
    # @since 0.1.0
    class Workers
      # Initializes a new Workers client instance
      #
      # @param rest_client [Object] The configured REST client for API communication
      # @param config [Hatchet::Config] The Hatchet configuration containing tenant_id and other settings
      # @return [void]
      # @since 0.1.0
      def initialize(rest_client, config)
        @rest_client = rest_client
        @config = config
        @worker_api = HatchetSdkRest::WorkerApi.new(rest_client)
      end

      # Get a worker by its ID
      #
      # @param worker_id [String] The ID of the worker to retrieve
      # @return [Object] The worker details
      # @raise [HatchetSdkRest::ApiError] If the API request fails
      # @example
      #   worker = workers_client.get("worker-123")
      def get(worker_id)
        @worker_api.worker_get(worker_id)
      end

      # List all workers in the tenant
      #
      # @return [Object] A list of workers
      # @raise [HatchetSdkRest::ApiError] If the API request fails
      # @example
      #   workers = workers_client.list
      def list
        @worker_api.worker_list(@config.tenant_id)
      end

      # Update a worker by its ID
      #
      # @param worker_id [String] The ID of the worker to update
      # @param opts [Hash] The update options
      # @return [Object] The updated worker
      # @raise [HatchetSdkRest::ApiError] If the API request fails
      # @example
      #   updated = workers_client.update("worker-123", { is_paused: true })
      def update(worker_id, opts)
        update_request = HatchetSdkRest::UpdateWorkerRequest.new(opts)
        @worker_api.worker_update(worker_id, update_request)
      end
    end
  end
end
