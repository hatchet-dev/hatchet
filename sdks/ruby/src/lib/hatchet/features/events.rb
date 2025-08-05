module Hatchet
  module Features
  # Events client for interacting with Hatchet event management API
  #
  # This class provides a high-level interface for creating and managing events
  # in the Hatchet system. It wraps the generated REST API client with a more
  # convenient Ruby interface.
  #
  # @example Creating an event
  #   events = Features::Events.new(rest_client, config)
  #   event_request = HatchetSdkRest::CreateEventRequest.new(
  #     key: "test-event",
  #     data: { key: "value" },
  #     additional_metadata: { source: "api" }
  #   )
  #   response = events.create(event_request)
  #
  # @since 0.0.0
  class Events
    # Re-export commonly used event classes for convenience
    CreateEventRequest = ::HatchetSdkRest::CreateEventRequest
    
    # Initializes a new Events client instance
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
      # @type [HatchetSdkRest::EventApi]
      @event_api = HatchetSdkRest::EventApi.new(rest_client)
    end

    # Creates a new event in the Hatchet system
    #
    # This method sends an event creation request to the Hatchet API using the
    # configured tenant ID. The event will be processed and made available for
    # workflow triggers and event-driven automation.
    #
    # @param event [HatchetSdkRest::CreateEventRequest] The event request object containing
    #   event data, metadata, and other properties
    # @return [Object] The API response containing the created event details
    # @raise [ArgumentError] If the event parameter is nil or invalid
    # @raise [Hatchet::Error] If the API request fails or returns an error
    # @example Creating a simple event
    #   event_request = HatchetSdkRest::CreateEventRequest.new(
    #     key: "user-login",
    #     data: { user_id: 123, action: "login" },
    #     additional_metadata: { ip_address: "192.168.1.1" }
    #   )
    #   response = events.create(event_request)
    # @since 0.0.2
    def create(event)
      @event_api.event_create(@config.tenant_id, event)
    end
  end
  end
end