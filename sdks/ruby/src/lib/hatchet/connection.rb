# frozen_string_literal: true

require "grpc"

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
        root_ca = tls_config.root_ca_file ? File.read(tls_config.root_ca_file) : nil
        GRPC::Core::ChannelCredentials.new(root_ca)
      end
    end
  end
end
