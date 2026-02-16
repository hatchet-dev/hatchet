# frozen_string_literal: true

module Hatchet
  # Rate limit durations
  module RateLimitDuration
    SECOND = :second
    MINUTE = :minute
    HOUR = :hour
    DAY = :day
    WEEK = :week
    MONTH = :month
    YEAR = :year
  end

  # Defines a rate limit for a task
  #
  # @example Static rate limit
  #   Hatchet::RateLimit.new(static_key: "api-calls", units: 1)
  #
  # @example Dynamic rate limit per user
  #   Hatchet::RateLimit.new(
  #     dynamic_key: "input.user_id",
  #     units: 1,
  #     limit: 10,
  #     duration: :minute
  #   )
  class RateLimit
    # @return [String, nil] Static rate limit key
    attr_reader :static_key

    # @return [String, nil] Dynamic rate limit key (CEL expression)
    attr_reader :dynamic_key

    # @return [Integer] Number of units consumed per execution
    attr_reader :units

    # @return [Integer, nil] Maximum number of units allowed in the duration
    attr_reader :limit

    # @return [Symbol, nil] Duration window for the rate limit
    attr_reader :duration

    # @param static_key [String, nil] Static rate limit key
    # @param dynamic_key [String, nil] Dynamic rate limit key expression
    # @param units [Integer] Units consumed per execution
    # @param limit [Integer, nil] Max units in the duration window
    # @param duration [Symbol, nil] Duration window (:second, :minute, :hour, etc.)
    def initialize(static_key: nil, dynamic_key: nil, units: 1, limit: nil, duration: nil)
      raise ArgumentError, "Must specify either static_key or dynamic_key" if static_key.nil? && dynamic_key.nil?
      raise ArgumentError, "Cannot specify both static_key and dynamic_key" if static_key && dynamic_key

      @static_key = static_key
      @dynamic_key = dynamic_key
      @units = units
      @limit = limit
      @duration = duration
    end

    # @return [Hash]
    def to_h
      h = { units: @units }
      h[:static_key] = @static_key if @static_key
      h[:dynamic_key] = @dynamic_key if @dynamic_key
      h[:limit] = @limit if @limit
      h[:duration] = @duration.to_s.upcase if @duration
      h
    end
  end
end
