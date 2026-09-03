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
  # @example Tenant-scoped, shared across workflows
  #   Hatchet::ConcurrencyExpression.new(
  #     expression: "input.group",
  #     max_runs: 1,
  #     limit_strategy: :group_round_robin,
  #     name: "tenant-wide-limit",
  #     is_tenant_scoped: true
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

    # @return [String, nil] Unique (per tenant) strategy name; required when tenant-scoped
    attr_reader :name

    # @return [Boolean] When true, the entry defines (or updates in place) a tenant-scoped
    #   strategy shared across workflows, keyed by name: every task declaring the same
    #   name consumes the same concurrency limit. The position in the concurrency list is
    #   the chain order, and chains sharing tenant-scoped strategies must order them
    #   consistently.
    attr_reader :is_tenant_scoped

    # @param expression [String] CEL expression evaluated against input
    # @param max_runs [Integer, String] Maximum concurrent runs, or a CEL expression computing them per group
    # @param limit_strategy [Symbol] Strategy when limit is reached
    # @param name [String, nil] Unique (per tenant) strategy name; required when tenant-scoped
    # @param is_tenant_scoped [Boolean] Share this strategy across workflows, keyed by name
    def initialize(expression:, max_runs: 1, limit_strategy: :cancel_in_progress, name: nil, is_tenant_scoped: false)
      raise ArgumentError, "a name is required for tenant-scoped concurrency" if is_tenant_scoped && (name.nil? || name.empty?)

      @expression = expression
      @max_runs = max_runs
      @limit_strategy = limit_strategy
      @name = name
      @is_tenant_scoped = is_tenant_scoped
    end

    # @deprecated kept for backwards compatibility; use {ConcurrencyProto::LIMIT_STRATEGY_MAP}
    LIMIT_STRATEGY_MAP = ConcurrencyProto::LIMIT_STRATEGY_MAP

    # Convert to a hash for API serialization
    # @return [Hash]
    def to_h
      h = {
        expression: @expression,
        max_runs: @max_runs,
        limit_strategy: @limit_strategy.to_s.upcase,
      }
      h[:name] = @name if @name
      h[:is_tenant_scoped] = true if @is_tenant_scoped
      h
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
      args[:name] = @name if @name
      args[:is_tenant_scoped] = true if @is_tenant_scoped

      ::V1::Concurrency.new(**args)
    end
  end

end
