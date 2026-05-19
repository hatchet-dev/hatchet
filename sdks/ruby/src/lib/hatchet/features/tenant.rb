# frozen_string_literal: true

module Hatchet
  module Features
    # Tenant client for interacting with the current Hatchet tenant
    #
    # This class provides a high-level interface for retrieving tenant information
    # from the Hatchet system.
    #
    # @example Getting the current tenant
    #   tenant_info = tenant_client.get
    #
    # @since 0.1.0
    class Tenant
      # Initializes a new Tenant client instance
      #
      # @param rest_client [Object] The configured REST client for API communication
      # @param config [Hatchet::Config] The Hatchet configuration containing tenant_id and other settings
      # @return [void]
      # @since 0.1.0
      def initialize(rest_client, config)
        @rest_client = rest_client
        @config = config
        @tenant_api = HatchetSdkRest::TenantApi.new(rest_client)
      end

      # Get the current tenant
      #
      # @return [Object] The tenant details
      # @raise [HatchetSdkRest::ApiError] If the API request fails
      # @example
      #   tenant = tenant_client.get
      def get
        @tenant_api.tenant_get(@config.tenant_id)
      end
    end
  end
end
