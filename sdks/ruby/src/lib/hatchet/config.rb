# frozen_string_literal: true

require "logger"
require "json"
require "base64"

module Hatchet
  # TLS configuration for client connections
  #
  # @example Basic TLS setup
  #   tls = TLSConfig.new(strategy: "tls", server_name: "api.hatchet.com")
  #
  # @example Custom certificate setup
  #   tls = TLSConfig.new(
  #     strategy: "mtls",
  #     cert_file: "/path/to/client.crt",
  #     key_file: "/path/to/client.key",
  #     root_ca_file: "/path/to/ca.crt"
  #   )
  class TLSConfig
    # @!attribute strategy
    #   @return [String] TLS strategy ("tls", "mtls", "insecure")
    # @!attribute cert_file
    #   @return [String, nil] Path to client certificate file for mTLS
    # @!attribute key_file
    #   @return [String, nil] Path to client private key file for mTLS
    # @!attribute root_ca_file
    #   @return [String, nil] Path to root CA certificate file
    # @!attribute server_name
    #   @return [String] Server name for TLS verification
    attr_accessor :strategy, :cert_file, :key_file, :root_ca_file, :server_name

    # Initialize TLS configuration
    #
    # @param options [Hash] TLS configuration options
    # @option options [String] :strategy TLS strategy (default: "tls")
    # @option options [String] :cert_file Path to client certificate file
    # @option options [String] :key_file Path to client private key file
    # @option options [String] :root_ca_file Path to root CA certificate file
    # @option options [String] :server_name Server name for TLS verification
    def initialize(**options)
      @strategy = options[:strategy] || env_var("HATCHET_CLIENT_TLS_STRATEGY") || "tls"
      @cert_file = options[:cert_file] || env_var("HATCHET_CLIENT_TLS_CERT_FILE")
      @key_file = options[:key_file] || env_var("HATCHET_CLIENT_TLS_KEY_FILE")
      @root_ca_file = options[:root_ca_file] || env_var("HATCHET_CLIENT_TLS_ROOT_CA_FILE")
      @server_name = options[:server_name] || env_var("HATCHET_CLIENT_TLS_SERVER_NAME") || ""
    end

    private

    def env_var(name)
      ENV.fetch(name, nil)
    end
  end

  # Healthcheck configuration for worker health monitoring
  #
  # @example Enable healthcheck on custom port
  #   healthcheck = HealthcheckConfig.new(enabled: true, port: 8080)
  class HealthcheckConfig
    # @!attribute port
    #   @return [Integer] Port number for healthcheck endpoint
    # @!attribute enabled
    #   @return [Boolean] Whether healthcheck is enabled
    attr_accessor :port, :enabled

    # Initialize healthcheck configuration
    #
    # @param options [Hash] Healthcheck configuration options
    # @option options [Integer] :port Port number for healthcheck endpoint (default: 8001)
    # @option options [Boolean] :enabled Whether healthcheck is enabled (default: false)
    def initialize(**options)
      @port = parse_int(options[:port] || env_var("HATCHET_CLIENT_WORKER_HEALTHCHECK_PORT")) || 8001
      @enabled = parse_bool(options[:enabled] || env_var("HATCHET_CLIENT_WORKER_HEALTHCHECK_ENABLED")) || false
    end

    private

    def env_var(name)
      ENV.fetch(name, nil)
    end

    def parse_int(value)
      return nil if value.nil?
      return value if value.is_a?(Integer)
      return nil if value.respond_to?(:empty?) && value.empty?

      Integer(value)
    rescue ArgumentError
      nil
    end

    def parse_bool(value)
      return nil if value.nil?
      return value if [true, false].include?(value)

      %w[true 1 yes on].include?(value.to_s.downcase)
    end
  end

  # Configuration class for Hatchet client settings
  #
  # This class manages all configuration options for the Hatchet client, including
  # token authentication, connection settings, and various client behaviors.
  # Configuration can be set via constructor options, environment variables, or
  # extracted from JWT tokens.
  #
  # @example Basic configuration
  #   config = Config.new(token: "your-jwt-token")
  #
  # @example Full configuration with all options
  #   config = Config.new(
  #     token: "your-jwt-token",
  #     host_port: "localhost:7070",
  #     server_url: "https://api.hatchet.com",
  #     namespace: "production",
  #     worker_preset_labels: { "env" => "prod", "region" => "us-east-1" }
  #   )
  #
  # @example Configuration via environment variables
  #   ENV["HATCHET_CLIENT_TOKEN"] = "your-jwt-token"
  #   ENV["HATCHET_CLIENT_HOST_PORT"] = "localhost:7070"
  #   config = Config.new  # Will load from environment
  class Config
    DEFAULT_HOST_PORT = "localhost:7070"
    DEFAULT_SERVER_URL = "https://app.dev.hatchet-tools.com"
    DEFAULT_GRPC_MAX_MESSAGE_LENGTH = 4 * 1024 * 1024 # 4MB

    ENV_FILE_NAMES = [".env", ".env.hatchet", ".env.dev", ".env.local"].freeze

    # @!attribute token
    #   @return [String] JWT token for authentication
    # @!attribute tenant_id
    #   @return [String] Tenant ID (extracted from JWT 'sub' field if not provided)
    # @!attribute host_port
    #   @return [String] gRPC server host and port
    # @!attribute server_url
    #   @return [String] Server URL for HTTP requests
    # @!attribute namespace
    #   @return [String] Namespace prefix for resource names
    # @!attribute logger
    #   @return [Logger] Logger instance
    # @!attribute listener_v2_timeout
    #   @return [Integer, nil] Timeout for listener v2 in milliseconds
    # @!attribute grpc_max_recv_message_length
    #   @return [Integer] Maximum gRPC receive message length in bytes
    # @!attribute grpc_max_send_message_length
    #   @return [Integer] Maximum gRPC send message length in bytes
    # @!attribute worker_preset_labels
    #   @return [Hash<String, String>] Default labels applied to all workers
    # @!attribute tls_config
    #   @return [TLSConfig] TLS configuration
    # @!attribute healthcheck
    #   @return [HealthcheckConfig] Healthcheck configuration
    attr_accessor :token, :host_port, :server_url, :namespace,
                  :logger, :listener_v2_timeout, :grpc_max_recv_message_length,
                  :grpc_max_send_message_length, :worker_preset_labels,
                  :tls_config, :healthcheck

    attr_reader :tenant_id

    # Initialize a new configuration instance
    #
    # Configuration values are loaded in the following priority order:
    # 1. Explicit constructor options (highest priority)
    # 2. Environment variables (HATCHET_CLIENT_*)
    # 3. JWT token payload (for tenant_id from 'sub' field)
    # 4. Default values (lowest priority)
    #
    # @param options [Hash] Configuration options
    # @option options [String] :token JWT token for authentication (required)
    # @option options [String] :host_port gRPC server host and port (default: "localhost:7070")
    # @option options [String] :server_url Server URL for HTTP requests (default: "https://app.dev.hatchet-tools.com")
    # @option options [String] :namespace Namespace prefix for resource names (default: "")
    # @option options [Logger] :logger Custom logger instance (default: Logger.new($stdout))
    # @option options [Integer] :listener_v2_timeout Timeout for listener v2 in milliseconds
    # @option options [Integer] :grpc_max_recv_message_length Maximum gRPC receive message length in bytes (default: 4MB)
    # @option options [Integer] :grpc_max_send_message_length Maximum gRPC send message length in bytes (default: 4MB)
    # @option options [Hash<String, String>] :worker_preset_labels Default labels applied to all workers (default: {})
    # @option options [TLSConfig] :tls_config Custom TLS configuration
    # @option options [HealthcheckConfig] :healthcheck Custom healthcheck configuration
    #
    # @raise [Error] if token is missing, empty, or not a valid JWT
    def initialize(**options)
      load_env_files
      @explicitly_set = options.keys.to_set

      @token = options[:token] || env_var("HATCHET_CLIENT_TOKEN") || ""
      @host_port = options[:host_port] || env_var("HATCHET_CLIENT_HOST_PORT") || DEFAULT_HOST_PORT
      @server_url = options[:server_url] || env_var("HATCHET_CLIENT_SERVER_URL") || DEFAULT_SERVER_URL
      @namespace = options[:namespace] || env_var("HATCHET_CLIENT_NAMESPACE") || ""
      @logger = options[:logger] || Logger.new($stdout)

      @listener_v2_timeout = parse_int(options[:listener_v2_timeout] || env_var("HATCHET_CLIENT_LISTENER_V2_TIMEOUT"))
      @grpc_max_recv_message_length = parse_int(
        options[:grpc_max_recv_message_length] ||
        env_var("HATCHET_CLIENT_GRPC_MAX_RECV_MESSAGE_LENGTH"),
      ) || DEFAULT_GRPC_MAX_MESSAGE_LENGTH
      @grpc_max_send_message_length = parse_int(
        options[:grpc_max_send_message_length] ||
        env_var("HATCHET_CLIENT_GRPC_MAX_SEND_MESSAGE_LENGTH"),
      ) || DEFAULT_GRPC_MAX_MESSAGE_LENGTH

      @worker_preset_labels = options[:worker_preset_labels] ||
                              parse_hash(env_var("HATCHET_CLIENT_WORKER_PRESET_LABELS")) || {}

      # Initialize nested configurations
      @tls_config = options[:tls_config] || TLSConfig.new
      @healthcheck = options[:healthcheck] || HealthcheckConfig.new

      # Initialize tenant_id from JWT token
      @tenant_id = ""

      validate!
      apply_token_defaults if valid_jwt_token?
      apply_address_defaults if valid_jwt_token?
      normalize_namespace!
    end

    def validate!
      raise Error, "Hatchet Token is required. Please set HATCHET_CLIENT_TOKEN in your environment." if token.nil? || token.empty?

      return if valid_jwt_token?

      raise Error,
            "Hatchet Token must be a valid JWT."
    end

    # Apply namespace prefix to a resource name
    #
    # @param resource_name [String, nil] The resource name to namespace
    # @param namespace_override [String, nil] Optional namespace to use instead of the configured one
    # @return [String, nil] The namespaced resource name, or nil if resource_name is nil
    #
    # @example Apply default namespace
    #   config = Config.new(token: "token", namespace: "prod")
    #   config.apply_namespace("workflow") #=> "prod_workflow"
    #
    # @example Apply custom namespace
    #   config.apply_namespace("workflow", namespace_override: "staging_") #=> "staging_workflow"
    #
    # @example Skip namespace if already present
    #   config.apply_namespace("prod_workflow") #=> "prod_workflow"
    def apply_namespace(resource_name, namespace_override: nil)
      return resource_name if resource_name.nil?

      namespace_to_use = namespace_override || namespace
      return resource_name if namespace_to_use.empty?
      return resource_name if resource_name.start_with?(namespace_to_use)

      "#{namespace_to_use}#{resource_name}"
    end

    def hash
      to_h.hash
    end

    # Convert configuration to a hash representation
    #
    # @return [Hash<Symbol, Object>] Hash containing all configuration values
    def to_h
      {
        token: token,
        host_port: host_port,
        server_url: server_url,
        namespace: namespace,
        listener_v2_timeout: listener_v2_timeout,
        grpc_max_recv_message_length: grpc_max_recv_message_length,
        grpc_max_send_message_length: grpc_max_send_message_length,
        worker_preset_labels: worker_preset_labels,
      }
    end

    # Returns authentication metadata for gRPC calls.
    #
    # @return [Hash<String, String>] Metadata hash with authorization bearer token
    def auth_metadata
      { "authorization" => "bearer #{token}" }
    end

    private

    def valid_jwt_token?
      !token.nil? && !token.empty? && token.start_with?("ey")
    end

    def apply_token_defaults
      # tenant_id is only set from JWT token, not from environment variables or parameters
      extracted_tenant_id = extract_tenant_id_from_jwt
      @tenant_id = extracted_tenant_id || ""
    end

    def apply_address_defaults
      jwt_server_url, jwt_host_port = extract_addresses_from_jwt

      @host_port = jwt_host_port if jwt_host_port && !explicitly_set?(:host_port)
      @server_url = jwt_server_url if jwt_server_url && !explicitly_set?(:server_url)

      # Set TLS server name if not already set
      return unless tls_config.server_name.empty?

      tls_config.server_name = host_port.split(":").first || "localhost"
    end

    def normalize_namespace!
      return if namespace.empty?

      @namespace = namespace.downcase
      @namespace += "_" unless @namespace.end_with?("_")
    end

    def load_env_files
      ENV_FILE_NAMES.each do |env_file|
        next unless File.exist?(env_file)

        File.foreach(env_file) do |line|
          line = line.strip
          next if line.empty? || line.start_with?("#")

          key, value = line.split("=", 2)
          next unless key && value

          ENV[key] ||= value.gsub(/\A['"]|['"]\z/, "") # Remove surrounding quotes
        end
      end
    end

    def env_var(name)
      ENV.fetch(name, nil)
    end

    def parse_int(value)
      return nil if value.nil?
      return value if value.is_a?(Integer)
      return nil if value.respond_to?(:empty?) && value.empty?

      Integer(value)
    rescue ArgumentError
      nil
    end

    def parse_hash(value)
      return nil if value.nil? || value.empty?

      result = {}
      value.split(",").each do |pair|
        key, val = pair.split("=", 2)
        result[key.strip] = val&.strip if key && val
      end
      result
    rescue StandardError
      nil
    end

    def explicitly_set?(attr)
      @explicitly_set.include?(attr)
    end

    def extract_tenant_id_from_jwt
      return nil unless valid_jwt_token?

      payload = decode_jwt_payload
      payload&.dig("sub")
    rescue StandardError
      nil
    end

    def extract_addresses_from_jwt
      return [nil, nil] unless valid_jwt_token?

      payload = decode_jwt_payload
      return [nil, nil] unless payload

      server_url = payload["server_url"]
      grpc_broadcast_address = payload["grpc_broadcast_address"]

      [server_url, grpc_broadcast_address]
    rescue StandardError
      [nil, nil]
    end

    def decode_jwt_payload
      return nil unless valid_jwt_token?

      # JWT has three parts separated by dots: header.payload.signature
      parts = token.split(".")
      return nil unless parts.length == 3

      # Decode the payload (second part)
      payload_part = parts[1]
      # Add padding if needed for Base64 decoding
      payload_part += "=" * (4 - (payload_part.length % 4)) if payload_part.length % 4 != 0

      decoded_payload = Base64.decode64(payload_part)
      JSON.parse(decoded_payload)
    rescue StandardError
      nil
    end
  end
end
