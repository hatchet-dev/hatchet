# frozen_string_literal: true

module Hatchet
  # Concurrency limit strategies
  module ConcurrencyLimitStrategy
    CANCEL_IN_PROGRESS = :cancel_in_progress
    CANCEL_NEWEST = :cancel_newest
    GROUP_ROUND_ROBIN = :group_round_robin
    QUEUE = :queue
    CANCEL_QUEUED_EXCEPT_NEWEST = :cancel_queued_except_newest
    CANCEL_QUEUED_EXCEPT_OLDEST = :cancel_queued_except_oldest
  end

  # Shared serialization helpers for concurrency entries
  # @api private
  module ConcurrencyProto
    # Map Ruby symbol to v1 proto enum symbol
    LIMIT_STRATEGY_MAP = {
      cancel_in_progress: :CANCEL_IN_PROGRESS,
      cancel_newest: :CANCEL_NEWEST,
      group_round_robin: :GROUP_ROUND_ROBIN,
      queue: :QUEUE_NEWEST,
      drop_newest: :DROP_NEWEST,
      cancel_queued_except_newest: :CANCEL_QUEUED_EXCEPT_NEWEST,
      cancel_queued_except_oldest: :CANCEL_QUEUED_EXCEPT_OLDEST,
    }.freeze

    # Split the max_runs union onto the proto's static/expression field pair. A String is
    # a CEL expression over task input; the static field then carries the default of 1,
    # which only governs slots created before the expression existed (each new task's
    # slot carries its own evaluated value).
    # @param max_runs [Integer, String]
    # @return [Array(Integer, String, nil)]
    def self.split_max_runs(max_runs)
      return [1, max_runs] if max_runs.is_a?(String)

      [max_runs, nil]
    end
  end

  # Defines a concurrency expression for workflow or task-level concurrency control
  #
  # @example Workflow-level concurrency
  #   Hatchet::ConcurrencyExpression.new(
  #     expression: "input.group_key",
  #     max_runs: 5,
  #     limit_strategy: :cancel_in_progress
  #   )
  #
  # @example Dynamic per-group limits via a CEL expression
  #   Hatchet::ConcurrencyExpression.new(
  #     expression: "input.tier",
  #     max_runs: "input.tier == 'premium' ? 10 : 1",
  #     limit_strategy: :group_round_robin
  #   )
  #
  # @example Task-level concurrency with multiple keys
  #   [
  #     Hatchet::ConcurrencyExpression.new(expression: "input.digit", max_runs: 8, limit_strategy: :group_round_robin),
  #     Hatchet::ConcurrencyExpression.new(expression: "input.name", max_runs: 3, limit_strategy: :group_round_robin)
  #   ]
  class ConcurrencyExpression
    # @return [String] CEL expression evaluated against the workflow input
    attr_reader :expression

    # @return [Integer, String] Maximum concurrent runs for this key: a fixed number, or
    #   a CEL expression over task input computing the max runs for that task's
    #   concurrency group. With an expression, a group's effective limit is the value
    #   from its most recently created task.
    attr_reader :max_runs

    # @return [Symbol] Strategy when limit is exceeded (:cancel_in_progress, :cancel_newest, :group_round_robin, :queue, :cancel_queued_except_newest, :cancel_queued_except_oldest)
    attr_reader :limit_strategy

    # @param expression [String] CEL expression evaluated against input
    # @param max_runs [Integer, String] Maximum concurrent runs, or a CEL expression computing them per group
    # @param limit_strategy [Symbol] Strategy when limit is reached
    def initialize(expression:, max_runs: 1, limit_strategy: :cancel_in_progress)
      @expression = expression
      @max_runs = max_runs
      @limit_strategy = limit_strategy
    end

    # @deprecated kept for backwards compatibility; use {ConcurrencyProto::LIMIT_STRATEGY_MAP}
    LIMIT_STRATEGY_MAP = ConcurrencyProto::LIMIT_STRATEGY_MAP

    # Convert to a hash for API serialization
    # @return [Hash]
    def to_h
      {
        expression: @expression,
        max_runs: @max_runs,
        limit_strategy: @limit_strategy.to_s.upcase,
      }
    end

    # Convert to a V1::Concurrency protobuf message
    # @return [V1::Concurrency]
    def to_proto
      proto_strategy = ConcurrencyProto::LIMIT_STRATEGY_MAP[@limit_strategy] || :CANCEL_IN_PROGRESS
      static_max, max_runs_expression = ConcurrencyProto.split_max_runs(@max_runs)

      args = {
        expression: @expression,
        max_runs: static_max,
        limit_strategy: proto_strategy,
      }
      args[:max_runs_expression] = max_runs_expression if max_runs_expression

      ::V1::Concurrency.new(**args)
    end
  end

  # A tenant-scoped concurrency strategy, shared across workflows: every task declaring
  # the same name consumes the same concurrency limit, and re-declaring a name updates
  # the strategy in place. Declare it anywhere a {ConcurrencyExpression} is accepted; the
  # position in the concurrency list is the chain order, so it may come before or after
  # workflow-scoped entries. Chains sharing tenant-scoped strategies must order them
  # consistently, or registration is rejected.
  #
  # @example Two workflows sharing one limit
  #   shared = Hatchet::SharedConcurrency.new(
  #     name: "tenant-wide-limit",
  #     expression: "input.group",
  #     max_runs: 1,
  #     limit_strategy: :group_round_robin
  #   )
  class SharedConcurrency
    # @return [String] Unique (per tenant) name of the strategy
    attr_reader :name

    # @return [String] CEL expression evaluated against the workflow input
    attr_reader :expression

    # @return [Integer, String] Maximum concurrent runs: a fixed number (defaults to 1),
    #   or a CEL expression over task input computing the max runs per group
    attr_reader :max_runs

    # @return [Symbol] Strategy when limit is exceeded
    attr_reader :limit_strategy

    # @param name [String] Unique (per tenant) strategy name
    # @param expression [String] CEL expression evaluated against input
    # @param max_runs [Integer, String] Maximum concurrent runs, or a CEL expression computing them per group
    # @param limit_strategy [Symbol] Strategy when limit is reached
    def initialize(name:, expression:, max_runs: 1, limit_strategy: :cancel_in_progress)
      raise ArgumentError, "name must be non-empty" if name.nil? || name.empty?

      @name = name
      @expression = expression
      @max_runs = max_runs
      @limit_strategy = limit_strategy
    end

    # Convert to a hash for API serialization
    # @return [Hash]
    def to_h
      {
        name: @name,
        is_tenant_scoped: true,
        expression: @expression,
        max_runs: @max_runs,
        limit_strategy: @limit_strategy.to_s.upcase,
      }
    end

    # Convert to a V1::Concurrency protobuf message
    # @return [V1::Concurrency]
    def to_proto
      proto_strategy = ConcurrencyProto::LIMIT_STRATEGY_MAP[@limit_strategy] || :CANCEL_IN_PROGRESS
      static_max, max_runs_expression = ConcurrencyProto.split_max_runs(@max_runs)

      args = {
        name: @name,
        is_tenant_scoped: true,
        expression: @expression,
        max_runs: static_max,
        limit_strategy: proto_strategy,
      }
      args[:max_runs_expression] = max_runs_expression if max_runs_expression

      ::V1::Concurrency.new(**args)
    end
  end
end
