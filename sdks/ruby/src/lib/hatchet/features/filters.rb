# frozen_string_literal: true

module Hatchet
  module Features
    # Filters client for interacting with Hatchet's filters API
    #
    # This class provides a high-level interface for creating, retrieving,
    # listing, updating, and deleting filters in the Hatchet system.
    #
    # @example Listing filters
    #   filters = filters_client.list(limit: 10, workflow_ids: ["wf-1"])
    #
    # @example Creating a filter
    #   filter = filters_client.create(
    #     workflow_id: "wf-1",
    #     expression: 'input.priority > 5',
    #     scope: "high-priority"
    #   )
    #
    # @since 0.1.0
    class Filters
      # Initializes a new Filters client instance
      #
      # @param rest_client [Object] The configured REST client for API communication
      # @param config [Hatchet::Config] The Hatchet configuration containing tenant_id and other settings
      # @return [void]
      # @since 0.1.0
      def initialize(rest_client, config)
        @rest_client = rest_client
        @config = config
        @filter_api = HatchetSdkRest::FilterApi.new(rest_client)
      end

      # List filters for the current tenant
      #
      # @param limit [Integer, nil] The maximum number of filters to return
      # @param offset [Integer, nil] The number of filters to skip
      # @param workflow_ids [Array<String>, nil] A list of workflow IDs to filter by
      # @param scopes [Array<String>, nil] A list of scopes to filter by
      # @return [Object] A list of filters matching the specified criteria
      # @raise [HatchetSdkRest::ApiError] If the API request fails
      # @example
      #   filters = filters_client.list(limit: 10, workflow_ids: ["wf-1"])
      def list(limit: nil, offset: nil, workflow_ids: nil, scopes: nil)
        @filter_api.v1_filter_list(
          @config.tenant_id,
          {
            limit: limit,
            offset: offset,
            workflow_ids: workflow_ids,
            scopes: scopes
          }
        )
      end

      # Get a filter by its ID
      #
      # @param filter_id [String] The ID of the filter to retrieve
      # @return [Object] The filter details
      # @raise [HatchetSdkRest::ApiError] If the API request fails
      # @example
      #   filter = filters_client.get("filter-123")
      def get(filter_id)
        @filter_api.v1_filter_get(@config.tenant_id, filter_id)
      end

      # Create a new filter
      #
      # @param workflow_id [String] The ID of the workflow to associate with the filter
      # @param expression [String] The CEL expression to evaluate for the filter
      # @param scope [String] The scope for the filter
      # @param payload [Hash, nil] The payload to send with the filter
      # @return [Object] The created filter
      # @raise [HatchetSdkRest::ApiError] If the API request fails
      # @example
      #   filter = filters_client.create(
      #     workflow_id: "wf-1",
      #     expression: 'input.value > 10',
      #     scope: "my-scope",
      #     payload: { threshold: 10 }
      #   )
      def create(workflow_id:, expression:, scope:, payload: nil)
        request = HatchetSdkRest::V1CreateFilterRequest.new(
          workflow_id: workflow_id,
          expression: expression,
          scope: scope,
          payload: payload
        )
        @filter_api.v1_filter_create(@config.tenant_id, request)
      end

      # Delete a filter by its ID
      #
      # @param filter_id [String] The ID of the filter to delete
      # @return [Object] The deleted filter
      # @raise [HatchetSdkRest::ApiError] If the API request fails
      # @example
      #   filters_client.delete("filter-123")
      def delete(filter_id)
        @filter_api.v1_filter_delete(@config.tenant_id, filter_id)
      end

      # Update a filter by its ID
      #
      # @param filter_id [String] The ID of the filter to update
      # @param updates [Hash] The updates to apply to the filter
      # @return [Object] The updated filter
      # @raise [HatchetSdkRest::ApiError] If the API request fails
      # @example
      #   filters_client.update("filter-123", { expression: 'input.value > 20' })
      def update(filter_id, updates)
        update_request = HatchetSdkRest::V1UpdateFilterRequest.new(updates)
        @filter_api.v1_filter_update(@config.tenant_id, filter_id, update_request)
      end
    end
  end
end
