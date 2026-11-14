# frozen_string_literal: true

require "google/protobuf/well_known_types"
require "google/rpc/error_details_pb"

module Hatchet
  module Clients
    module Grpc
      # Retries a call the engine throttled, waiting the delay the server asked for.
      #
      # The engine marks throttling as RESOURCE_EXHAUSTED with a RetryInfo detail. RESOURCE_EXHAUSTED
      # without RetryInfo is a quota that a retry will not clear, so it is raised at once, as is every
      # other error.
      module ServerRetryDelay
        MAX_ATTEMPTS = 5

        # Bounds each wait, and so a call's total wait, whatever delay the server sends.
        MAX_DELAY_SECONDS = 5.0

        module_function

        # @yield The gRPC call, run again for each retry
        # @return [Object] The block's result
        def call
          attempt = 1
          begin
            yield
          rescue ::GRPC::ResourceExhausted => e
            delay = retry_delay(e)
            raise if delay.nil? || attempt >= MAX_ATTEMPTS

            attempt += 1
            sleep(delay.clamp(0.0, MAX_DELAY_SECONDS))
            retry
          end
        end

        # @param error [GRPC::BadStatus]
        # @return [Float, nil] Seconds from the error's RetryInfo detail, or nil when it has none
        def retry_delay(error)
          error.to_rpc_status&.details&.each do |any|
            delay = any.unpack(::Google::Rpc::RetryInfo)&.retry_delay
            return delay.seconds + (delay.nanos / 1e9) if delay
          end
          nil
        end
      end
    end
  end
end
