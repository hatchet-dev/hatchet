# frozen_string_literal: true

require "json"
require "google/protobuf/timestamp_pb"

module Hatchet
  module Clients
    module Grpc
      # gRPC client for the Hatchet Admin service (workflow registration & triggering).
      #
      # Uses two stubs on the shared channel:
      # - V1::AdminService::Stub (v1) for: PutWorkflow, GetRunDetails, CancelTasks, ReplayTasks
      # - WorkflowService::Stub (v0) for: TriggerWorkflow, BulkTriggerWorkflow, ScheduleWorkflow, PutRateLimit
      #
      # The v0 WorkflowService is retained for triggering because it supports
      # parent-child linking fields that the v1 TriggerWorkflowRun does not expose.
      #
      # @example
      #   admin = Admin.new(config: hatchet_config, channel: channel)
      #   admin.put_workflow(workflow.to_proto(config))
      #   ref = admin.trigger_workflow("MyWorkflow", input: { "key" => "value" })
      class Admin
        BULK_TRIGGER_BATCH_SIZE = 1000

        # @param config [Hatchet::Config] The Hatchet configuration
        # @param channel [GRPC::Core::Channel] Shared gRPC channel
        def initialize(config:, channel:)
          @config = config
          @logger = config.logger
          @channel = channel
          @v0_stub = nil
          @v1_stub = nil
        end

        # Register a workflow definition with the server via v1 AdminService.
        #
        # @param workflow_proto [V1::CreateWorkflowVersionRequest] The workflow proto
        # @return [V1::CreateWorkflowVersionResponse] Registration response
        def put_workflow(workflow_proto)
          ensure_connected!

          response = @v1_stub.put_workflow(workflow_proto, metadata: @config.auth_metadata)
          @logger.debug("Registered workflow: #{workflow_proto.name}")
          response
        end

        # Trigger a workflow run via v0 WorkflowService.
        #
        # @param workflow_name [String] The workflow name (will be namespaced)
        # @param input [Hash] Workflow input
        # @param options [Hash] Trigger options
        # @option options [String] :parent_id Parent workflow run ID
        # @option options [String] :parent_task_run_external_id Parent step run ID
        # @option options [Integer] :child_index Child workflow index
        # @option options [String] :child_key Child workflow key
        # @option options [Hash] :additional_metadata Additional metadata
        # @option options [String] :desired_worker_id Desired worker for sticky dispatch
        # @option options [Integer] :priority Priority value
        # @return [String] The workflow run ID
        # @raise [DedupeViolationError] If a deduplication violation occurs
        def trigger_workflow(workflow_name, input: {}, options: {})
          ensure_connected!

          name = @config.apply_namespace(workflow_name)

          request_args = {
            name: name,
            input: JSON.generate(input),
          }

          request_args[:parent_id] = options[:parent_id] if options[:parent_id]
          request_args[:parent_task_run_external_id] = options[:parent_task_run_external_id] if options[:parent_task_run_external_id]
          request_args[:child_index] = options[:child_index] if options[:child_index]
          request_args[:child_key] = options[:child_key] if options[:child_key]
          request_args[:desired_worker_id] = options[:desired_worker_id] if options[:desired_worker_id]
          request_args[:priority] = options[:priority] if options[:priority]

          if options[:additional_metadata]
            request_args[:additional_metadata] = if options[:additional_metadata].is_a?(String)
                                                   options[:additional_metadata]
                                                 else
                                                   JSON.generate(options[:additional_metadata])
                                                 end
          end

          request = ::TriggerWorkflowRequest.new(**request_args)

          begin
            response = @v0_stub.trigger_workflow(request, metadata: @config.auth_metadata)
            response.workflow_run_id
          rescue ::GRPC::AlreadyExists => e
            raise DedupeViolationError, "Deduplication violation: #{e.message}"
          end
        end

        # Trigger multiple workflow runs in bulk via v0 WorkflowService.
        #
        # Automatically batches requests in groups of 1000.
        #
        # @param workflow_name [String] The workflow name (will be namespaced)
        # @param items [Array<Hash>] Array of { input:, options: } items
        # @return [Array<String>] Array of workflow run IDs
        def bulk_trigger_workflow(workflow_name, items)
          ensure_connected!

          name = @config.apply_namespace(workflow_name)

          requests = items.map do |item|
            input = item[:input] || {}
            opts = item[:options] || {}

            request_args = {
              name: name,
              input: JSON.generate(input),
            }

            request_args[:parent_id] = opts[:parent_id] if opts[:parent_id]
            request_args[:parent_task_run_external_id] = opts[:parent_task_run_external_id] if opts[:parent_task_run_external_id]
            request_args[:child_index] = opts[:child_index] if opts[:child_index]
            request_args[:child_key] = opts[:child_key] if opts[:child_key]
            request_args[:desired_worker_id] = opts[:desired_worker_id] if opts[:desired_worker_id]
            request_args[:priority] = opts[:priority] if opts[:priority]

            if opts[:additional_metadata]
              request_args[:additional_metadata] = if opts[:additional_metadata].is_a?(String)
                                                     opts[:additional_metadata]
                                                   else
                                                     JSON.generate(opts[:additional_metadata])
                                                   end
            end

            ::TriggerWorkflowRequest.new(**request_args)
          end

          # Batch in groups of BULK_TRIGGER_BATCH_SIZE
          all_run_ids = []
          requests.each_slice(BULK_TRIGGER_BATCH_SIZE) do |batch|
            bulk_request = ::BulkTriggerWorkflowRequest.new(workflows: batch)
            response = @v0_stub.bulk_trigger_workflow(bulk_request, metadata: @config.auth_metadata)
            all_run_ids.concat(response.workflow_run_ids.to_a)
          end

          all_run_ids
        end

        # Schedule a workflow for future execution via v0 WorkflowService.
        #
        # @param workflow_name [String] The workflow name (will be namespaced)
        # @param run_at [Time] When to run
        # @param input [Hash] Workflow input
        # @param options [Hash] Trigger options
        # @return [WorkflowVersion] Schedule response
        # @raise [DedupeViolationError] If a deduplication violation occurs
        def schedule_workflow(workflow_name, run_at:, input: {}, options: {})
          ensure_connected!

          name = @config.apply_namespace(workflow_name)

          schedule_timestamp = Google::Protobuf::Timestamp.new(
            seconds: run_at.to_i,
            nanos: run_at.respond_to?(:nsec) ? run_at.nsec : 0,
          )

          request_args = {
            name: name,
            schedules: [schedule_timestamp],
            input: JSON.generate(input),
          }

          request_args[:parent_id] = options[:parent_id] if options[:parent_id]
          request_args[:parent_task_run_external_id] = options[:parent_task_run_external_id] if options[:parent_task_run_external_id]
          request_args[:child_index] = options[:child_index] if options[:child_index]
          request_args[:child_key] = options[:child_key] if options[:child_key]
          request_args[:priority] = options[:priority] if options[:priority]

          if options[:additional_metadata]
            request_args[:additional_metadata] = if options[:additional_metadata].is_a?(String)
                                                   options[:additional_metadata]
                                                 else
                                                   JSON.generate(options[:additional_metadata])
                                                 end
          end

          request = ::ScheduleWorkflowRequest.new(**request_args)

          begin
            @v0_stub.schedule_workflow(request, metadata: @config.auth_metadata)
          rescue ::GRPC::AlreadyExists => e
            raise DedupeViolationError, "Deduplication violation: #{e.message}"
          end
        end

        # Get run details via v1 AdminService.
        #
        # @param external_id [String] The workflow run external ID
        # @return [V1::GetRunDetailsResponse]
        def get_run_details(external_id:)
          ensure_connected!

          request = ::V1::GetRunDetailsRequest.new(external_id: external_id)
          @v1_stub.get_run_details(request, metadata: @config.auth_metadata)
        end

        # Put a rate limit via v0 WorkflowService.
        #
        # @param key [String] Rate limit key
        # @param limit [Integer] Rate limit value
        # @param duration [Symbol] Rate limit duration enum
        # @return [PutRateLimitResponse]
        def put_rate_limit(key:, limit:, duration:)
          ensure_connected!

          request = ::PutRateLimitRequest.new(
            key: key,
            limit: limit,
            duration: duration,
          )

          @v0_stub.put_rate_limit(request, metadata: @config.auth_metadata)
        end

        # Close the connection.
        def close
          @v0_stub = nil
          @v1_stub = nil
        end

        private

        def ensure_connected!
          return if @v0_stub && @v1_stub

          @v0_stub = ::WorkflowService::Stub.new(
            @config.host_port,
            nil,
            channel_override: @channel,
          )

          @v1_stub = ::V1::AdminService::Stub.new(
            @config.host_port,
            nil,
            channel_override: @channel,
          )

          @logger.debug("Admin gRPC stubs (v0 + v1) connected via shared channel")
        end
      end
    end
  end
end
