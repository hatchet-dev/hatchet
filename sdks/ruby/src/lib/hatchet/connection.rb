# frozen_string_literal: true

require "digest"
require "grpc"
require "openssl"

module Hatchet
  # Connection management for gRPC channels.
  #
  # Creates a shared gRPC channel with proper TLS/mTLS configuration
  # and keepalive settings. The channel is shared across all gRPC stubs
  # (dispatcher, admin, events) for connection reuse and thread safety.
  module Connection
    DEFAULT_CHANNEL_OPTIONS = {
      "grpc.keepalive_time_ms" => 10_000,
      "grpc.keepalive_timeout_ms" => 60_000,
      "grpc.client_idle_timeout_ms" => 60_000,
      "grpc.http2.max_pings_without_data" => 0,
      "grpc.keepalive_permit_without_calls" => 1,
    }.freeze

    COMPROMISED_ROOT_FINGERPRINTS = %w[
      86A1ECBA089C4A8D3BBE2734C612BA341D813E043CF9E8A862CD5C57A36BBE6B
      D7A7A0FB5D7E2731D771E9484EBCDEF71D5F0C3E0A2948782BC83EE0EA699EF4
    ].freeze

    CERTIFICATE_PEM_PATTERN = /-----BEGIN CERTIFICATE-----.+?-----END CERTIFICATE-----/m
    TRUTHY_ENV_VALUES = %w[1 t true y yes].freeze

    # Create a new gRPC channel for the given configuration.
    #
    # @param config [Hatchet::Config] The Hatchet configuration
    # @return [GRPC::Core::Channel] A configured gRPC channel
    def self.new_channel(config)
      credentials = build_credentials(config.tls_config)

      channel_args = DEFAULT_CHANNEL_OPTIONS.merge(
        "grpc.max_send_message_length" => config.grpc_max_send_message_length,
        "grpc.max_receive_message_length" => config.grpc_max_recv_message_length,
      )

      if config.tls_config.strategy == "none"
        GRPC::Core::Channel.new(config.host_port, channel_args, :this_channel_is_insecure)
      else
        channel_args["grpc.ssl_target_name_override"] = config.tls_config.server_name
        GRPC::Core::Channel.new(config.host_port, channel_args, credentials)
      end
    end

    # Build gRPC channel credentials from TLS configuration.
    #
    # @param tls_config [Hatchet::TLSConfig] TLS configuration
    # @return [GRPC::Core::ChannelCredentials, Symbol] Credentials or :this_channel_is_insecure
    def self.build_credentials(tls_config)
      case tls_config.strategy
      when "none"
        :this_channel_is_insecure
      when "mtls"
        root_ca = File.read(tls_config.root_ca_file)
        private_key = File.read(tls_config.key_file)
        cert_chain = File.read(tls_config.cert_file)
        GRPC::Core::ChannelCredentials.new(root_ca, private_key, cert_chain)
      else # "tls" (default)
        root_ca = if tls_config.root_ca_file
                    File.read(tls_config.root_ca_file)
                  elsif use_filtered_default_roots?
                    filtered_default_roots
                  end
        GRPC::Core::ChannelCredentials.new(root_ca)
      end
    end

    def self.use_filtered_default_roots?
      return false unless ENV.fetch("GRPC_DEFAULT_SSL_ROOTS_FILE_PATH", "").empty?

      !TRUTHY_ENV_VALUES.include?(ENV.fetch("GRPC_USE_SYSTEM_ROOTS_OVER_LANGUAGE_CALLBACK", "").strip.downcase)
    end

    def self.filtered_default_roots
      return @filtered_default_roots if defined?(@filtered_default_roots)

      @filtered_default_roots = load_filtered_roots
    end

    def self.load_filtered_roots
      path = grpc_roots_path
      return nil unless path

      kept = File.read(path).scan(CERTIFICATE_PEM_PATTERN).reject do |pem|
        COMPROMISED_ROOT_FINGERPRINTS.include?(
          Digest::SHA256.hexdigest(OpenSSL::X509::Certificate.new(pem).to_der).upcase,
        )
      end
      kept.empty? ? nil : "#{kept.join("\n")}\n"
    rescue StandardError, LoadError
      nil
    end

    def self.grpc_roots_path
      dir = Gem.loaded_specs["grpc"]&.full_gem_path || Gem::Specification.find_by_name("grpc").full_gem_path
      path = File.join(dir, "etc", "roots.pem")
      path if File.file?(path)
    end

    private_class_method :load_filtered_roots, :grpc_roots_path
  end
end
