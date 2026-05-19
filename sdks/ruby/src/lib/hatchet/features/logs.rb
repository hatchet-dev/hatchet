# frozen_string_literal: true

require "time"

module Hatchet
  module Features
    # Logs client for interacting with Hatchet's logs API
    #
    # This class provides a high-level interface for listing log lines
    # associated with task runs in the Hatchet system.
    #
    # @example Listing logs for a task run
    #   logs = logs_client.list("task-run-id", limit: 100)
    #
    # @since 0.1.0
    class Logs
      # Initializes a new Logs client instance
      #
      # @param rest_client [Object] The configured REST client for API communication
      # @param config [Hatchet::Config] The Hatchet configuration containing tenant_id and other settings
      # @return [void]
      # @since 0.1.0
      def initialize(rest_client, config)
        @rest_client = rest_client
        @config = config
        @log_api = HatchetSdkRest::LogApi.new(rest_client)
      end

      # List log lines for a given task run
      #
      # @param task_run_id [String] The ID of the task run to list logs for
      # @param limit [Integer] Maximum number of log lines to return (default: 1000)
      # @param since [Time, nil] The start time to get logs for
      # @param until_time [Time, nil] The end time to get logs for
      # @return [Object] A list of log lines for the specified task run
      # @raise [HatchetSdkRest::ApiError] If the API request fails
      # @example
      #   logs = logs_client.list("task-run-123", limit: 500, since: Time.now - 3600)
      def list(task_run_id, limit: 1000, since: nil, until_time: nil)
        @log_api.v1_log_line_list(
          task_run_id,
          {
            limit: limit,
            since: since&.utc&.iso8601,
            _until: until_time&.utc&.iso8601,
          },
        )
      end
    end
  end
end
