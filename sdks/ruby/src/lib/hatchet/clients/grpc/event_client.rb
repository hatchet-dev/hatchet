# frozen_string_literal: true

require "json"
require "google/protobuf/timestamp_pb"

module Hatchet
  module Clients
    module Grpc
      # gRPC client for the Hatchet Events service.
      #
      # Handles pushing events, bulk events, logs, and stream events via gRPC.
      # Uses the generated EventsService::Stub from events.proto on the shared channel.
      #
      # @example
      #   event_client = EventClient.new(config: hatchet_config, channel: channel)
      #   event_client.push(key: "user:create", payload: { "name" => "Alice" })
      class EventClient
        MAX_LOG_MESSAGE_LENGTH = 10_000

        # @param config [Hatchet::Config] The Hatchet configuration
        # @param channel [GRPC::Core::Channel] Shared gRPC channel
        def initialize(config:, channel:)
          @config = config
          @logger = config.logger
          @channel = channel
          @stub = nil
        end

        # Push an event via gRPC.
        #
        # @param key [String] Event key (will be namespaced)
        # @param payload [Hash] Event payload
        # @param additional_metadata [Hash, nil] Additional metadata
        # @param priority [Integer, nil] Event priority
        # @param scope [String, nil] Event scope
        # @param namespace [String, nil] Optional namespace override
        # @return [Event] Push response
        def push(key:, payload:, additional_metadata: nil, priority: nil, scope: nil, namespace: nil)
          ensure_connected!

          now = Time.now
          timestamp = Google::Protobuf::Timestamp.new(
            seconds: now.to_i,
            nanos: now.nsec
          )

          namespaced_key = @config.apply_namespace(key, namespace_override: namespace)

          request_args = {
            key: namespaced_key,
            payload: JSON.generate(payload),
            event_timestamp: timestamp
          }

          if additional_metadata
            request_args[:additional_metadata] = if additional_metadata.is_a?(String)
                                                   additional_metadata
                                                 else
                                                   JSON.generate(additional_metadata)
                                                 end
          end

          request_args[:priority] = priority if priority
          request_args[:scope] = scope if scope

          request = ::PushEventRequest.new(**request_args)
          @stub.push(request, metadata: @config.auth_metadata)
        end

        # Push multiple events via gRPC.
        #
        # @param events [Array<Hash>] Array of event hashes with :key, :payload, :additional_metadata, :priority, :scope
        # @param namespace [String, nil] Optional namespace override applied to all events
        # @return [Events] Bulk push response
        def bulk_push(events, namespace: nil)
          ensure_connected!

          now = Time.now
          timestamp = Google::Protobuf::Timestamp.new(
            seconds: now.to_i,
            nanos: now.nsec
          )

          items = events.map do |e|
            request_args = {
              key: @config.apply_namespace(e[:key], namespace_override: namespace),
              payload: JSON.generate(e[:payload] || {}),
              event_timestamp: timestamp
            }

            if e[:additional_metadata]
              request_args[:additional_metadata] = if e[:additional_metadata].is_a?(String)
                                                     e[:additional_metadata]
                                                   else
                                                     JSON.generate(e[:additional_metadata])
                                                   end
            end

            request_args[:priority] = e[:priority] if e[:priority]
            request_args[:scope] = e[:scope] if e[:scope]

            ::PushEventRequest.new(**request_args)
          end

          request = ::BulkPushEventRequest.new(events: items)
          @stub.bulk_push(request, metadata: @config.auth_metadata)
        end

        # Put a log message for a task run.
        #
        # @param step_run_id [String] The task run external ID
        # @param message [String] Log message (truncated to 10K chars)
        # @return [PutLogResponse]
        def put_log(step_run_id:, message:)
          ensure_connected!

          now = Time.now
          timestamp = Google::Protobuf::Timestamp.new(
            seconds: now.to_i,
            nanos: now.nsec
          )

          truncated_message = message.length > MAX_LOG_MESSAGE_LENGTH ? message[0...MAX_LOG_MESSAGE_LENGTH] : message

          request = ::PutLogRequest.new(
            task_run_external_id: step_run_id,
            created_at: timestamp,
            message: truncated_message
          )

          @stub.put_log(request, metadata: @config.auth_metadata)
        end

        # Put a stream event for real-time streaming.
        #
        # @param step_run_id [String] The task run external ID
        # @param data [String] Stream data chunk (sent as bytes)
        # @return [PutStreamEventResponse]
        def put_stream(step_run_id:, data:)
          ensure_connected!

          now = Time.now
          timestamp = Google::Protobuf::Timestamp.new(
            seconds: now.to_i,
            nanos: now.nsec
          )

          # The message field in PutStreamEventRequest is bytes
          message_bytes = data.is_a?(String) ? data.b : data.to_s.b

          request = ::PutStreamEventRequest.new(
            task_run_external_id: step_run_id,
            created_at: timestamp,
            message: message_bytes
          )

          @stub.put_stream_event(request, metadata: @config.auth_metadata)
        end

        # Close the connection.
        def close
          @stub = nil
        end

        private

        def ensure_connected!
          return if @stub

          @stub = ::EventsService::Stub.new(
            @config.host_port,
            nil,
            channel_override: @channel
          )

          @logger.debug("Events gRPC stub connected via shared channel")
        end
      end
    end
  end
end
