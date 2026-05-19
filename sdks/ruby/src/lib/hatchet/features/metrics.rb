# frozen_string_literal: true

require "time"

module Hatchet
  module Features
    # Simple struct for task metrics grouped by status
    TaskMetrics = Struct.new(:cancelled, :completed, :failed, :queued, :running, keyword_init: true)

    # Metrics client for reading metrics out of Hatchet's metrics API
    #
    # This class provides a high-level interface for retrieving queue metrics,
    # Prometheus metrics, task statistics, and task metrics from the Hatchet system.
    #
    # @example Getting queue metrics
    #   metrics = metrics_client.get_queue_metrics
    #
    # @example Getting task metrics
    #   task_metrics = metrics_client.get_task_metrics(
    #     since: Time.now - 86400,
    #     workflow_ids: ["wf-1"]
    #   )
    #
    # @since 0.1.0
    class Metrics
      # Initializes a new Metrics client instance
      #
      # @param rest_client [Object] The configured REST client for API communication
      # @param config [Hatchet::Config] The Hatchet configuration containing tenant_id and other settings
      # @return [void]
      # @since 0.1.0
      def initialize(rest_client, config)
        @rest_client = rest_client
        @config = config
        @task_api = HatchetSdkRest::TaskApi.new(rest_client)
        @tenant_api = HatchetSdkRest::TenantApi.new(rest_client)
      end

      # Retrieve the current queue metrics for the tenant
      #
      # @return [Hash] The current queue metrics
      # @raise [HatchetSdkRest::ApiError] If the API request fails
      # @example
      #   queues = metrics_client.get_queue_metrics
      def get_queue_metrics
        result = @tenant_api.tenant_get_step_run_queue_metrics(@config.tenant_id)
        result.queues || {}
      end

      # Scrape Prometheus metrics for the tenant
      #
      # @return [String] The metrics in Prometheus text format
      # @raise [HatchetSdkRest::ApiError] If the API request fails
      # @example
      #   prometheus_text = metrics_client.scrape_tenant_prometheus_metrics
      def scrape_tenant_prometheus_metrics
        @tenant_api.tenant_get_prometheus_metrics(@config.tenant_id)
      end

      # Get task statistics for the tenant
      #
      # @return [Object] The task statistics
      # @raise [HatchetSdkRest::ApiError] If the API request fails
      # @example
      #   stats = metrics_client.get_task_stats
      def get_task_stats
        @tenant_api.tenant_get_task_stats(@config.tenant_id)
      end

      # Retrieve task metrics grouped by status (queued, running, completed, failed, cancelled)
      #
      # @param since [Time, nil] Start time for the metrics query (defaults to the past day if unset)
      # @param until_time [Time, nil] End time for the metrics query
      # @param workflow_ids [Array<String>, nil] List of workflow IDs to filter the metrics by
      # @param parent_task_external_id [String, nil] ID of the parent task to filter by
      # @param triggering_event_external_id [String, nil] ID of the triggering event to filter by
      # @return [TaskMetrics] Task metrics with counts per status
      # @raise [HatchetSdkRest::ApiError] If the API request fails
      # @example
      #   metrics = metrics_client.get_task_metrics(
      #     since: Time.now - 86400,
      #     workflow_ids: ["wf-1"]
      #   )
      #   puts "Completed: #{metrics.completed}, Failed: #{metrics.failed}"
      def get_task_metrics(since: nil, until_time: nil, workflow_ids: nil, parent_task_external_id: nil, triggering_event_external_id: nil)
        since_time = since || (Time.now - (24 * 60 * 60))
        until_val = until_time || Time.now

        result = @task_api.v1_task_list_status_metrics(
          @config.tenant_id,
          since_time.utc.iso8601,
          {
            _until: until_val.utc.iso8601,
            workflow_ids: workflow_ids,
            parent_task_external_id: parent_task_external_id,
            triggering_event_external_id: triggering_event_external_id,
          },
        )

        # Build metrics hash from the status metric objects
        metrics = { cancelled: 0, completed: 0, failed: 0, queued: 0, running: 0 }

        if result.is_a?(Array)
          result.each do |m|
            status_name = m.respond_to?(:status) ? m.status.to_s.downcase.to_sym : nil
            count = m.respond_to?(:count) ? m.count : 0
            metrics[status_name] = count if metrics.key?(status_name)
          end
        end

        TaskMetrics.new(**metrics)
      end
    end
  end
end
