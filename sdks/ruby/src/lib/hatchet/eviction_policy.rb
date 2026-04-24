# frozen_string_literal: true

module Hatchet
  # Task-scoped eviction parameters for *durable* tasks.
  #
  # Setting the durable task's eviction policy to ``nil`` means the task run is
  # never eligible for eviction.
  #
  # @example
  #   Hatchet::EvictionPolicy.new(
  #     ttl: 600,                     # 10 minutes, in seconds
  #     allow_capacity_eviction: true,
  #     priority: 0,
  #   )
  class EvictionPolicy
    # @return [Numeric, nil] Maximum continuous waiting duration in seconds before
    #   TTL-eligible eviction. Applies to time spent in SDK-instrumented
    #   "waiting" states (e.g. :meth:`DurableContext#sleep_for`,
    #   :meth:`DurableContext#wait_for`). ``nil`` disables TTL eviction.
    attr_reader :ttl

    # @return [Boolean] Whether this task may be evicted under durable-slot pressure.
    attr_reader :allow_capacity_eviction

    # @return [Integer] Lower values are evicted first when multiple candidates exist.
    attr_reader :priority

    # @param ttl [Numeric, nil] TTL in seconds (or nil to disable TTL-based eviction)
    # @param allow_capacity_eviction [Boolean]
    # @param priority [Integer]
    def initialize(ttl:, allow_capacity_eviction: true, priority: 0)
      @ttl = ttl
      @allow_capacity_eviction = allow_capacity_eviction
      @priority = priority
      freeze
    end

    def ==(other)
      other.is_a?(EvictionPolicy) &&
        other.ttl == ttl &&
        other.allow_capacity_eviction == allow_capacity_eviction &&
        other.priority == priority
    end
    alias eql? ==

    def hash
      [self.class, ttl, allow_capacity_eviction, priority].hash
    end
  end

  # Shared sensible defaults.
  #
  # NOTE: When changing these values, update the :param eviction_policy: docstrings
  # in :meth:`Workflow#durable_task` and :meth:`Client#durable_task` to match.
  DEFAULT_DURABLE_TASK_EVICTION_POLICY = EvictionPolicy.new(
    ttl: 15 * 60, # 15 minutes
    allow_capacity_eviction: true,
    priority: 0,
  )
end
