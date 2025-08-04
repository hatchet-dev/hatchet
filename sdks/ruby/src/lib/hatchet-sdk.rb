# frozen_string_literal: true
# typed: strict

require_relative "hatchet/version"
require_relative "hatchet/config"

# Ruby SDK for Hatchet workflow engine
#
# @see https://docs.hatchet.run for Hatchet documentation
module Hatchet
  # Base error class for all Hatchet-related errors
  class Error < StandardError; end

  # The main client for interacting with Hatchet services.
  #
  # @example Basic usage with API token
  #   hatchet = Hatchet::Client.new()
  #
  # @example With custom configuration
  #   hatchet = Hatchet::Client.new(
  #     token: "your-jwt-token",
  #     namespace: "production"
  #   )
  class Client
    # @return [Config] The configuration object used by this client
    attr_reader :config

    # Initialize a new Hatchet client with the given configuration options.
    #
    # @param options [Hash] Configuration options for the client
    # @option options [String] :token The JWT token for authentication (required)
    # @option options [String] :tenant_id Override tenant ID (extracted from JWT token 'sub' field if not provided)
    # @option options [String] :host_port gRPC server host and port (default: "localhost:7070")
    # @option options [String] :server_url Server URL for HTTP requests (default: "https://app.dev.hatchet-tools.com")
    # @option options [String] :namespace Namespace prefix for resource names (default: "")
    # @option options [Logger] :logger Custom logger instance (default: Logger.new($stdout))
    # @option options [Integer] :listener_v2_timeout Timeout for listener v2 in milliseconds
    # @option options [Integer] :grpc_max_recv_message_length Maximum gRPC receive message length (default: 4MB)
    # @option options [Integer] :grpc_max_send_message_length Maximum gRPC send message length (default: 4MB)
    # @option options [Hash] :worker_preset_labels Hash of preset labels for workers
    # @option options [Boolean] :enable_force_kill_sync_threads Enable force killing of sync threads (default: false)
    # @option options [Boolean] :enable_thread_pool_monitoring Enable thread pool monitoring (default: false)
    # @option options [Integer] :terminate_worker_after_num_tasks Terminate worker after this many tasks
    # @option options [Boolean] :disable_log_capture Disable log capture (default: false)
    # @option options [Boolean] :grpc_enable_fork_support Enable gRPC fork support (default: false)
    # @option options [TLSConfig] :tls_config Custom TLS configuration
    # @option options [HealthcheckConfig] :healthcheck Custom healthcheck configuration
    # @option options [OpenTelemetryConfig] :otel Custom OpenTelemetry configuration
    #
    # @raise [Error] if token or configuration is missing or invalid
    #
    # @example Initialize with minimal configuration
    #   client = Hatchet::Client.new()
    #
    # @example Initialize with custom options
    #   client = Hatchet::Client.new(
    #     token: "eyJhbGciOiJIUzI1NiJ9...",
    #     namespace: "my_app",
    #     worker_preset_labels: { "env" => "production", "version" => "1.0.0" }
    #   )
    def initialize(**options)
      @config = Config.new(**options)
    end
  end
end
