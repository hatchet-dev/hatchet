# frozen_string_literal: true

# Hatchet clients module
# This module provides access to different client types (REST, gRPC, etc.)

module Hatchet
  # Client implementations for different protocols
  module Clients
    # Load REST client if available
    begin
      require_relative "clients/rest"
    rescue LoadError
      # REST client not generated yet - this is expected initially
      # Run `rake api:generate` to generate the REST client
    end

    # Factory methods for creating pre-configured clients
    class << self
      # Create a REST API client using the provided Hatchet configuration
      #
      # @param config [Hatchet::Config] The main Hatchet configuration
      # @return [Hatchet::Clients::Rest::ApiClient] Configured REST API client
      # @raise [LoadError] if REST client hasn't been generated yet
      #
      # @example Create a REST client
      #   config = Hatchet::Config.new(token: "your-jwt-token")
      #   rest_client = Hatchet::Clients.rest_client(config)
      #   workflows_api = Hatchet::Clients::Rest::WorkflowApi.new(rest_client)
      def rest_client(config)
        raise LoadError, "REST client not available. Run `rake api:generate` to generate it from the OpenAPI spec." unless rest_available?

        rest_config = Rest::Configuration.from_hatchet_config(config)
        Rest::ApiClient.new(rest_config)
      end

      # Check if REST client is available
      #
      # @return [Boolean] true if REST client has been generated and is available
      def rest_available?
        return false unless defined?(Hatchet::Clients::Rest)

        # Check if this is the real implementation or just the placeholder
        # The placeholder Configuration.from_hatchet_config raises LoadError
        begin
          # Try to access a method that should exist in the real implementation
          # If it's the placeholder, this will raise LoadError
          Hatchet::Clients::Rest::Configuration.method(:from_hatchet_config)

          # Try creating a dummy configuration to ensure the real client is loaded
          dummy_config = Struct.new(:token, :server_url, :listener_v2_timeout).new("test", "", nil)
          Hatchet::Clients::Rest::Configuration.from_hatchet_config(dummy_config)
          true
        rescue LoadError
          false
        end
      end

      # List available client types
      #
      # @return [Array<Symbol>] List of available client types
      def available_clients
        clients = []
        clients << :rest if rest_available?
        clients
      end
    end
  end
end
