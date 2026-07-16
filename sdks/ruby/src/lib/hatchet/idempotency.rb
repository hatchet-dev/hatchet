# frozen_string_literal: true

module Hatchet
  # TTL-based idempotency: prevents duplicate runs within a sliding time window.
  #
  # @example
  #   Hatchet::TTLBasedIdempotencyConfig.new(expression: "input.id", ttl_ms: 60_000)
  class TTLBasedIdempotencyConfig
    # @return [String] CEL expression evaluated against workflow input
    attr_reader :expression

    # @return [Integer] How long the idempotency key lives, in milliseconds
    attr_reader :ttl_ms

    # @param expression [String] CEL expression to derive the idempotency key
    # @param ttl_ms [Integer] TTL for the idempotency key in milliseconds
    def initialize(expression:, ttl_ms:)
      @expression = expression
      @ttl_ms = ttl_ms
    end

    # @return [V1::IdempotencyConfig]
    def to_proto
      ::V1::IdempotencyConfig.new(expression: @expression, ttl_ms: @ttl_ms)
    end
  end
end
