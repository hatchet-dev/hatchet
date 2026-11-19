# frozen_string_literal: true

require "google/protobuf/well_known_types"
require "google/rpc/error_details_pb"

module Hatchet
  module Clients
    module Grpc
      # Retries a call the engine throttled, waiting the delay the server asked for.
      #
      # A call is retried only when the server says how long to wait: RESOURCE_EXHAUSTED with a
      # RetryInfo detail and a positive delay. Everything else is raised at once, including
      # RESOURCE_EXHAUSTED without RetryInfo, which is either a quota that a retry will not clear or
      # the per-API-token rate limit; neither carries a delay, so the SDK has no wait to honor.
      #
      # The delay is doubled for each retry already made, which is the client behavior RetryInfo
      # documents: a server that keeps throttling repeats the same delay, so without the doubling
      # the client would not back off.
      module ServerRetryDelay
        MAX_ATTEMPTS = 5

        # The longest wait the SDK accepts from a server; the wait blocks the caller. A call the server
        # asks to wait longer fails at once with the server's error, since every retry before that
        # delay would be refused.
        MAX_DELAY_SECONDS = 5.0

        # The most a wait is extended, as a share of the wait. Clients throttled at the same moment
        # get the same delay; the random extension keeps them from all coming back at the same
        # moment. It only extends the wait, since a retry before the server's delay would be refused.
        JITTER = 0.2

        module_function

        # @yield The gRPC call, run again for each retry
        # @return [Object] The block's result
        def call
          attempt = 1
          begin
            yield
          rescue ::GRPC::ResourceExhausted => e
            delay = retry_delay(e)
            raise unless delay&.positive? && delay <= MAX_DELAY_SECONDS && attempt < MAX_ATTEMPTS

            wait = [delay * (2**(attempt - 1)), MAX_DELAY_SECONDS].min
            sleep(wait * (1 + rand(0.0..JITTER)))
            attempt += 1
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
