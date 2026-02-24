# frozen_string_literal: true

module Hatchet
  module Features
    # Rate Limits client for managing rate limits within Hatchet
    #
    # This class provides a high-level interface for creating and updating
    # rate limits using the gRPC Admin client.
    #
    # @example Setting a rate limit
    #   rate_limits_client.put(key: "api-calls", limit: 100, duration: :SECOND)
    #
    # @since 0.1.0
    class RateLimits
      # Initializes a new RateLimits client instance
      #
      # @param admin_grpc [Hatchet::Clients::Grpc::Admin] The gRPC Admin client
      # @param config [Hatchet::Config] The Hatchet configuration
      # @return [void]
      # @since 0.1.0
      def initialize(admin_grpc, config)
        @admin_grpc = admin_grpc
        @config = config
      end

      # Put a rate limit for a given key
      #
      # @param key [String] The key to set the rate limit for
      # @param limit [Integer] The rate limit to set
      # @param duration [Symbol] The duration of the rate limit (:SECOND, :MINUTE, :HOUR)
      # @return [void]
      # @raise [GRPC::BadStatus] If the gRPC request fails
      # @example
      #   rate_limits_client.put(key: "api-calls", limit: 100, duration: :SECOND)
      def put(key:, limit:, duration: :SECOND)
        @admin_grpc.put_rate_limit(key: key, limit: limit, duration: duration)
      end
    end
  end
end
