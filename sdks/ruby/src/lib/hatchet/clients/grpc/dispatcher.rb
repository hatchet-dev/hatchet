# frozen_string_literal: true

require "google/protobuf/timestamp_pb"

module Hatchet
  module Clients
    module Grpc
      # gRPC client for the Hatchet Dispatcher service.
      #
      # Handles worker registration, action listening, result reporting,
      # heartbeats, and other dispatcher interactions.
      #
      # Uses the generated Dispatcher::Stub from dispatcher.proto for v0 RPCs,
      # and shares a gRPC channel provided by Hatchet::Connection.
      #
      # @example
      #   dispatcher = Dispatcher.new(config: hatchet_config, channel: channel)
      #   response = dispatcher.register(name: "my-worker", ...)
      #   dispatcher.listen(worker_id: response.worker_id) { |action| ... }
      class Dispatcher
        # @param config [Hatchet::Config] The Hatchet configuration
        # @param channel [GRPC::Core::Channel] Shared gRPC channel
        # @return [String, nil] Worker ID assigned after registration
        attr_reader :worker_id

        def initialize(config:, channel:)
          @config = config
          @logger = config.logger
          @channel = channel
          @stub = nil
          @worker_id = nil
        end

        # Register a worker with the dispatcher.
        #
        # @param name [String] Worker name
        # @param actions [Array<String>] List of action IDs this worker handles
        # @param slots [Integer] Number of concurrent task slots
        # @param labels [Hash] Worker labels (String keys, String or Integer values)
        # @return [WorkerRegisterResponse] Registration response with worker_id
        def register(name:, actions:, slots:, labels: {})
          ensure_connected!

          label_map = labels.each_with_object({}) do |(k, v), map|
            wl = if v.is_a?(Integer)
                   ::WorkerLabels.new(int_value: v)
                 else
                   ::WorkerLabels.new(str_value: v.to_s)
                 end
            map[k.to_s] = wl
          end

          runtime_info = ::RuntimeInfo.new(
            language: :RUBY,
            sdk_version: Hatchet::VERSION,
            language_version: RUBY_VERSION,
            os: RUBY_PLATFORM,
          )

          request = ::WorkerRegisterRequest.new(
            worker_name: name,
            actions: actions,
            slots: slots,
            labels: label_map,
            runtime_info: runtime_info,
          )

          begin
            response = @stub.register(request, metadata: @config.auth_metadata)
          rescue ::GRPC::Internal
            request = ::WorkerRegisterRequest.new(
              worker_name: name,
              actions: actions,
              slots: slots,
              labels: label_map,
            )
            response = @stub.register(request, metadata: @config.auth_metadata)
            @logger.warn("Registered without runtime_info — engine may not support RUBY language type. Consider upgrading your Hatchet engine.")
          end

          @worker_id = response.worker_id
          @logger.info("Registered worker '#{name}' with #{actions.length} action(s), worker_id=#{response.worker_id}")
          response
        end

        # Listen for action assignments via gRPC server-streaming (ListenV2).
        #
        # Returns an Enumerator of AssignedAction messages. The caller is
        # responsible for iterating and handling reconnection.
        #
        # @param worker_id [String] The registered worker ID
        # @return [Enumerator] Stream of AssignedAction messages
        def listen(worker_id:)
          ensure_connected!

          request = ::WorkerListenRequest.new(worker_id: worker_id)
          @stub.listen_v2(request, metadata: @config.auth_metadata)
        end

        # Send a heartbeat to keep the worker registration alive.
        #
        # @param worker_id [String] The worker ID
        # @return [HeartbeatResponse]
        def heartbeat(worker_id:)
          ensure_connected!

          now = Time.now
          timestamp = Google::Protobuf::Timestamp.new(
            seconds: now.to_i,
            nanos: now.nsec,
          )

          request = ::HeartbeatRequest.new(
            worker_id: worker_id,
            heartbeat_at: timestamp,
          )

          @stub.heartbeat(request, metadata: @config.auth_metadata)
        end

        # Send a step action event (completion/failure/started) back to the dispatcher.
        #
        # Accepts the full action object (AssignedAction) so all StepActionEvent
        # fields can be populated, matching the Python SDK pattern.
        #
        # @param action [AssignedAction] The assigned action object
        # @param event_type [Symbol] Protobuf enum value (e.g., :STEP_EVENT_TYPE_COMPLETED)
        # @param payload [String] JSON-serialized event payload
        # @param retry_count [Integer, nil] Current retry count
        # @param should_not_retry [Boolean, nil] Whether to suppress further retries
        # @return [ActionEventResponse]
        def send_step_action_event(action:, event_type:, payload: "{}", retry_count: nil, should_not_retry: nil)
          ensure_connected!

          now = Time.now
          timestamp = Google::Protobuf::Timestamp.new(
            seconds: now.to_i,
            nanos: now.nsec,
          )

          event_args = {
            worker_id: @worker_id || "",
            job_id: action.job_id,
            job_run_id: action.job_run_id,
            task_id: action.task_id,
            task_run_external_id: action.task_run_external_id,
            action_id: action.action_id,
            event_timestamp: timestamp,
            event_type: event_type,
            event_payload: payload,
          }

          event_args[:retry_count] = retry_count unless retry_count.nil?
          event_args[:should_not_retry] = should_not_retry unless should_not_retry.nil?

          request = ::StepActionEvent.new(**event_args)
          @stub.send_step_action_event(request, metadata: @config.auth_metadata)
        end

        # Refresh the timeout for a running task.
        #
        # @param step_run_id [String] The task run external ID
        # @param timeout_seconds [Integer, String] New timeout increment (in seconds or as a duration string)
        # @return [RefreshTimeoutResponse]
        def refresh_timeout(step_run_id:, timeout_seconds:)
          ensure_connected!

          increment = timeout_seconds.is_a?(String) ? timeout_seconds : "#{timeout_seconds}s"

          request = ::RefreshTimeoutRequest.new(
            task_run_external_id: step_run_id,
            increment_timeout_by: increment,
          )

          @stub.refresh_timeout(request, metadata: @config.auth_metadata)
        end

        # Release a worker slot for a task.
        #
        # @param step_run_id [String] The task run external ID
        # @return [ReleaseSlotResponse]
        def release_slot(step_run_id:)
          ensure_connected!

          request = ::ReleaseSlotRequest.new(
            task_run_external_id: step_run_id,
          )

          @stub.release_slot(request, metadata: @config.auth_metadata)
        end

        # Update worker labels.
        #
        # @param worker_id [String] The worker ID
        # @param labels [Hash] New labels to upsert (String keys, String/Integer values)
        # @return [UpsertWorkerLabelsResponse]
        def upsert_worker_labels(worker_id:, labels:)
          ensure_connected!

          label_map = labels.each_with_object({}) do |(k, v), map|
            wl = if v.is_a?(Integer)
                   ::WorkerLabels.new(int_value: v)
                 else
                   ::WorkerLabels.new(str_value: v.to_s)
                 end
            map[k.to_s] = wl
          end

          request = ::UpsertWorkerLabelsRequest.new(
            worker_id: worker_id,
            labels: label_map,
          )

          @stub.upsert_worker_labels(request, metadata: @config.auth_metadata)
        end

        # Open a bidirectional streaming subscription for workflow run events.
        #
        # The caller provides an Enumerable (typically an Enumerator backed by
        # a Queue) of SubscribeToWorkflowRunsRequest messages. The server
        # streams back WorkflowRunEvent messages as workflow runs complete.
        #
        # @param request_enum [Enumerable<SubscribeToWorkflowRunsRequest>] Outgoing request stream
        # @return [Enumerator<WorkflowRunEvent>] Incoming response stream
        def subscribe_to_workflow_runs(request_enum)
          ensure_connected!

          @stub.subscribe_to_workflow_runs(request_enum, metadata: @config.auth_metadata)
        end

        # Close the gRPC channel.
        def close
          @stub = nil
        end

        private

        def ensure_connected!
          return if @stub

          @stub = ::Dispatcher::Stub.new(
            @config.host_port,
            nil,
            channel_override: @channel,
          )

          @logger.debug("Dispatcher gRPC stub connected via shared channel")
        end
      end
    end
  end
end
