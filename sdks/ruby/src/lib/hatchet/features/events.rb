# frozen_string_literal: true

require "time"

module Hatchet
  module Features
    # Events client for interacting with Hatchet event management API
    #
    # This class provides a high-level interface for creating and managing events
    # in the Hatchet system. It uses gRPC for event creation (push/bulk_push) and
    # the REST API for read operations (list, get, etc.).
    #
    # @example Creating an event
    #   response = events.push(
    #     key: "user-login",
    #     data: { user_id: 123, action: "login" },
    #     additional_metadata: { ip_address: "192.168.1.1" }
    #   )
    #
    # @since 0.1.0
    class Events
      # Re-export commonly used event classes for convenience
      CreateEventRequest = ::HatchetSdkRest::CreateEventRequest
      BulkCreateEventRequest = ::HatchetSdkRest::BulkCreateEventRequest
      EventList = ::HatchetSdkRest::V1EventList

      # Initializes a new Events client instance
      #
      # @param rest_client [Object] The configured REST client for API communication
      # @param event_grpc [Hatchet::Clients::Grpc::EventClient] The gRPC event client for push operations
      # @param config [Hatchet::Config] The Hatchet configuration containing tenant_id and other settings
      # @return [void]
      # @since 0.1.0
      def initialize(rest_client, event_grpc, config)
        @rest_client = rest_client
        @event_grpc = event_grpc
        @config = config
        @event_api = HatchetSdkRest::EventApi.new(rest_client)
      end

      # Creates a new event in the Hatchet system
      #
      # This method sends an event creation request via gRPC. The event will be
      # processed and made available for workflow triggers and event-driven automation.
      #
      # @param key [String] The event key/name
      # @param data [Hash] The event payload data
      # @param additional_metadata [Hash, nil] Additional metadata for the event
      # @param priority [Integer, nil] Event priority
      # @param scope [String, nil] The scope for event filtering
      # @param namespace [String, nil] Override namespace for this event
      # @return [Object] The gRPC response containing the created event details
      # @raise [ArgumentError] If required parameters are missing
      # @raise [Hatchet::Error] If the API request fails or returns an error
      # @example Creating a simple event
      #   response = events.create(
      #     key: "user-login",
      #     data: { user_id: 123, action: "login" },
      #     additional_metadata: { ip_address: "192.168.1.1" }
      #   )
      # @since 0.1.0
      def create(key:, data:, additional_metadata: nil, priority: nil, scope: nil, namespace: nil)
        @event_grpc.push(
          key: key,
          payload: data,
          additional_metadata: additional_metadata,
          priority: priority,
          scope: scope,
          namespace: namespace,
        )
      end

      # Push a single event to Hatchet
      #
      # @param event_key [String] The event key/name
      # @param payload [Hash] The event payload data
      # @param additional_metadata [Hash, nil] Additional metadata for the event
      # @param namespace [String, nil] Override namespace for this event
      # @param priority [Integer, nil] Event priority
      # @return [Object] The gRPC response containing the created event details
      # @raise [Hatchet::Error] If the API request fails or returns an error
      # @example Push a simple event
      #   response = events.push(
      #     "user-signup",
      #     { user_id: 456, email: "user@example.com" },
      #     additional_metadata: { source: "web" }
      #   )
      def push(event_key, payload, additional_metadata: nil, namespace: nil, priority: nil)
        create(
          key: event_key,
          data: payload,
          additional_metadata: additional_metadata,
          priority: priority,
          namespace: namespace,
        )
      end

      # Create events in bulk
      #
      # @param events [Array<Hash>] Array of event hashes, each containing :key, :data, and optionally :additional_metadata and :priority
      # @param namespace [String, nil] Override namespace for all events
      # @return [Object] The gRPC response containing the created events
      # @raise [Hatchet::Error] If the API request fails or returns an error
      # @example Bulk create events
      #   events_data = [
      #     { key: "user-signup", data: { user_id: 1 } },
      #     { key: "user-login", data: { user_id: 1 }, priority: 1 }
      #   ]
      #   response = events.bulk_push(events_data)
      def bulk_push(events, namespace: nil)
        grpc_events = events.map do |event|
          {
            key: event[:key],
            payload: event[:data] || {},
            additional_metadata: event[:additional_metadata],
            priority: event[:priority],
          }
        end

        @event_grpc.bulk_push(grpc_events, namespace: namespace)
      end

      # List events with filtering options
      #
      # @param offset [Integer, nil] Pagination offset
      # @param limit [Integer, nil] Maximum number of events to return
      # @param keys [Array<String>, nil] Filter by event keys
      # @param since [Time, nil] Filter events after this time
      # @param until_time [Time, nil] Filter events before this time
      # @param workflow_ids [Array<String>, nil] Filter by workflow IDs
      # @param workflow_run_statuses [Array<String>, nil] Filter by workflow run statuses
      # @param event_ids [Array<String>, nil] Filter by specific event IDs
      # @param additional_metadata [Hash<String, String>, nil] Filter by additional metadata
      # @param scopes [Array<String>, nil] Filter by event scopes
      # @return [HatchetSdkRest::V1EventList] List of events matching the filters
      # @raise [Hatchet::Error] If the API request fails or returns an error
      # @example List recent events
      #   events = events_client.list(
      #     limit: 10,
      #     since: Time.now - 24 * 60 * 60,
      #     keys: ["user-signup", "user-login"]
      #   )
      def list(
        offset: nil,
        limit: nil,
        keys: nil,
        since: nil,
        until_time: nil,
        workflow_ids: nil,
        workflow_run_statuses: nil,
        event_ids: nil,
        additional_metadata: nil,
        scopes: nil
      )
        @event_api.v1_event_list(
          @config.tenant_id,
          {
            offset: offset,
            limit: limit,
            keys: keys,
            since: since&.utc&.iso8601,
            until: until_time&.utc&.iso8601,
            workflow_ids: workflow_ids,
            workflow_run_statuses: workflow_run_statuses,
            event_ids: event_ids,
            additional_metadata: maybe_additional_metadata_to_kv(additional_metadata),
            scopes: scopes,
          },
        )
      end

      # Get a specific event by ID
      #
      # @param event_id [String] The event ID
      # @return [Object] The event details
      # @raise [Hatchet::Error] If the API request fails or returns an error
      def get(event_id)
        @event_api.v1_event_get(@config.tenant_id, event_id)
      end

      # Get event data for a specific event
      #
      # @param event_id [String] The event ID
      # @return [Object] The event data
      # @raise [Hatchet::Error] If the API request fails or returns an error
      def get_data(event_id)
        @event_api.event_data_get_with_tenant(event_id, @config.tenant_id)
      end

      # List available event keys for the tenant
      #
      # @return [Object] List of available event keys
      # @raise [Hatchet::Error] If the API request fails or returns an error
      def list_keys
        @event_api.v1_event_key_list(@config.tenant_id)
      end

      # Cancel events matching the given criteria
      #
      # @param event_ids [Array<String>, nil] Specific event IDs to cancel
      # @param keys [Array<String>, nil] Event keys to cancel
      # @param since [Time, nil] Cancel events after this time
      # @param until_time [Time, nil] Cancel events before this time
      # @return [Object] The cancellation response
      # @raise [Hatchet::Error] If the API request fails or returns an error
      def cancel(event_ids: nil, keys: nil, since: nil, until_time: nil)
        cancel_request = HatchetSdkRest::CancelEventRequest.new(
          event_ids: event_ids,
          keys: keys,
          since: since&.utc&.iso8601,
          until: until_time&.utc&.iso8601,
        )

        @event_api.event_update_cancel(@config.tenant_id, cancel_request)
      end

      # Replay events matching the given criteria
      #
      # @param event_ids [Array<String>, nil] Specific event IDs to replay
      # @param keys [Array<String>, nil] Event keys to replay
      # @param since [Time, nil] Replay events after this time
      # @param until_time [Time, nil] Replay events before this time
      # @return [Object] The replay response
      # @raise [Hatchet::Error] If the API request fails or returns an error
      def replay(event_ids: nil, keys: nil, since: nil, until_time: nil)
        replay_request = HatchetSdkRest::ReplayEventRequest.new(
          event_ids: event_ids,
          keys: keys,
          since: since&.utc&.iso8601,
          until: until_time&.utc&.iso8601,
        )

        @event_api.event_update_replay(@config.tenant_id, replay_request)
      end

      private

      # Apply namespace to event key
      #
      # @param event_key [String] The original event key
      # @param namespace_override [String, nil] Optional namespace override
      # @return [String] The namespaced event key
      def apply_namespace(event_key, namespace_override = nil)
        @config.apply_namespace(event_key, namespace_override: namespace_override)
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
