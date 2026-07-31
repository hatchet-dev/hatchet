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
      ::V1::IdempotencyConfig.new(
        expression: @expression,
        ttl_ms: @ttl_ms,
        method: ::V1::IdempotencyMethod::TTL,
      )
    end
  end

  # Status-based idempotency: keeps the idempotency key alive until the associated run
  # reaches a terminal status. +fallback_ttl_ms+ caps how long the key can live before
  # it's evicted, even if the run has not reached a terminal status.
  #
  # @example
  #   Hatchet::StatusBasedIdempotencyConfig.new(expression: "input.id", fallback_ttl_ms: 10_000)
  class StatusBasedIdempotencyConfig
    # @return [String] CEL expression evaluated against workflow input
    attr_reader :expression

    # @return [Integer] Fallback TTL in milliseconds; the longest the key can live
    attr_reader :fallback_ttl_ms

    # @param expression [String] CEL expression to derive the idempotency key
    # @param fallback_ttl_ms [Integer] Fallback TTL for the idempotency key in milliseconds
    def initialize(expression:, fallback_ttl_ms:)
      @expression = expression
      @fallback_ttl_ms = fallback_ttl_ms
    end

    # @return [V1::IdempotencyConfig]
    def to_proto
      ::V1::IdempotencyConfig.new(
        expression: @expression,
        ttl_ms: @fallback_ttl_ms,
        method: ::V1::IdempotencyMethod::STATUS,
      )
    end
  end
end
