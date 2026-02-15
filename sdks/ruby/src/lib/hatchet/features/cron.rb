# frozen_string_literal: true

module Hatchet
  module Features
    # Cron client for managing cron workflow triggers within Hatchet
    #
    # This class provides a high-level interface for creating, deleting,
    # listing, and retrieving cron workflow triggers.
    #
    # @example Creating a cron trigger
    #   cron = cron_client.create(
    #     workflow_name: "my-workflow",
    #     cron_name: "daily-run",
    #     expression: "0 0 * * *",
    #     input: { key: "value" },
    #     additional_metadata: { source: "api" }
    #   )
    #
    # @since 0.1.0
    class Cron
      CRON_ALIASES = %w[@yearly @annually @monthly @weekly @daily @hourly].freeze

      # Initializes a new Cron client instance
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

      # Create a new workflow cron trigger
      #
      # @param workflow_name [String] The name of the workflow to trigger (namespace will be applied)
      # @param cron_name [String] The name of the cron trigger
      # @param expression [String] The cron expression defining the schedule
      # @param input [Hash] The input data for the cron workflow
      # @param additional_metadata [Hash] Additional metadata associated with the cron trigger
      # @param priority [Integer, nil] The priority of the cron workflow trigger
      # @return [Object] The created cron workflow instance
      # @raise [ArgumentError] If the cron expression is invalid
      # @raise [HatchetSdkRest::ApiError] If the API request fails
      # @example
      #   cron = cron_client.create(
      #     workflow_name: "my-workflow",
      #     cron_name: "hourly-run",
      #     expression: "0 * * * *",
      #     input: { key: "value" },
      #     additional_metadata: { source: "api" }
      #   )
      def create(workflow_name:, cron_name:, expression:, input: {}, additional_metadata: {}, priority: nil)
        validated_expression = validate_cron_expression(expression)

        request = HatchetSdkRest::CreateCronWorkflowTriggerRequest.new(
          cron_name: cron_name,
          cron_expression: validated_expression,
          input: input,
          additional_metadata: additional_metadata,
          priority: priority,
        )

        @workflow_run_api.cron_workflow_trigger_create(
          @config.tenant_id,
          @config.apply_namespace(workflow_name),
          request,
        )
      end

      # Delete a workflow cron trigger
      #
      # @param cron_id [String] The ID of the cron trigger to delete
      # @return [void]
      # @raise [HatchetSdkRest::ApiError] If the API request fails
      # @example
      #   cron_client.delete("cron-123")
      def delete(cron_id)
        @workflow_api.workflow_cron_delete(@config.tenant_id, cron_id.to_s)
      end

      # List cron workflow triggers matching the specified criteria
      #
      # @param offset [Integer, nil] The offset to start the list from
      # @param limit [Integer, nil] The maximum number of items to return
      # @param workflow_id [String, nil] The ID of the workflow to filter by
      # @param additional_metadata [Hash, nil] Filter by additional metadata keys
      # @param order_by_field [String, nil] The field to order the list by
      # @param order_by_direction [String, nil] The direction to order the list by
      # @param workflow_name [String, nil] The name of the workflow to filter by
      # @param cron_name [String, nil] The name of the cron trigger to filter by
      # @return [Object] A list of cron workflows
      # @raise [HatchetSdkRest::ApiError] If the API request fails
      # @example
      #   crons = cron_client.list(limit: 10, workflow_name: "my-workflow")
      def list(offset: nil, limit: nil, workflow_id: nil, additional_metadata: nil,
               order_by_field: nil, order_by_direction: nil, workflow_name: nil, cron_name: nil)
        @workflow_api.cron_workflow_list(
          @config.tenant_id,
          {
            offset: offset,
            limit: limit,
            workflow_id: workflow_id,
            additional_metadata: maybe_additional_metadata_to_kv(additional_metadata),
            order_by_field: order_by_field,
            order_by_direction: order_by_direction,
            workflow_name: workflow_name,
            cron_name: cron_name,
          },
        )
      end

      # Retrieve a specific workflow cron trigger by ID
      #
      # @param cron_id [String] The cron trigger ID to retrieve
      # @return [Object] The requested cron workflow instance
      # @raise [HatchetSdkRest::ApiError] If the API request fails
      # @example
      #   cron = cron_client.get("cron-123")
      def get(cron_id)
        @workflow_api.workflow_cron_get(@config.tenant_id, cron_id.to_s)
      end

      private

      # Validate a cron expression
      #
      # @param expression [String] The cron expression to validate
      # @return [String] The validated cron expression
      # @raise [ArgumentError] If the expression is invalid
      def validate_cron_expression(expression)
        raise ArgumentError, "Cron expression is required" if expression.nil? || expression.empty?

        stripped = expression.strip

        # Allow cron aliases
        return stripped if CRON_ALIASES.include?(stripped)

        parts = stripped.split
        raise ArgumentError, "Cron expression must have 5 parts: minute hour day month weekday" unless parts.length == 5

        parts.each do |part|
          unless part == "*" || part.gsub("*/", "").gsub("-", "").gsub(",", "").match?(/\A\d+\z/)
            raise ArgumentError, "Invalid cron expression part: #{part}"
          end
        end

        expression
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
