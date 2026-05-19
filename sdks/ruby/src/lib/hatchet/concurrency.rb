# frozen_string_literal: true

module Hatchet
  # Concurrency limit strategies
  module ConcurrencyLimitStrategy
    CANCEL_IN_PROGRESS = :cancel_in_progress
    CANCEL_NEWEST = :cancel_newest
    GROUP_ROUND_ROBIN = :group_round_robin
    QUEUE = :queue
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
  # @example Task-level concurrency with multiple keys
  #   [
  #     Hatchet::ConcurrencyExpression.new(expression: "input.digit", max_runs: 8, limit_strategy: :group_round_robin),
  #     Hatchet::ConcurrencyExpression.new(expression: "input.name", max_runs: 3, limit_strategy: :group_round_robin)
  #   ]
  class ConcurrencyExpression
    # @return [String] CEL expression evaluated against the workflow input
    attr_reader :expression

    # @return [Integer] Maximum concurrent runs for this key
    attr_reader :max_runs

    # @return [Symbol] Strategy when limit is exceeded (:cancel_in_progress, :cancel_newest, :group_round_robin, :queue)
    attr_reader :limit_strategy

    # @param expression [String] CEL expression evaluated against input
    # @param max_runs [Integer] Maximum concurrent runs
    # @param limit_strategy [Symbol] Strategy when limit is reached
    def initialize(expression:, max_runs: 1, limit_strategy: :cancel_in_progress)
      @expression = expression
      @max_runs = max_runs
      @limit_strategy = limit_strategy
    end

    # Convert to a hash for API serialization
    # @return [Hash]
    def to_h
      {
        expression: @expression,
        max_runs: @max_runs,
        limit_strategy: @limit_strategy.to_s.upcase,
      }
    end

    # Map Ruby symbol to v1 proto enum symbol
    LIMIT_STRATEGY_MAP = {
      cancel_in_progress: :CANCEL_IN_PROGRESS,
      cancel_newest: :CANCEL_NEWEST,
      group_round_robin: :GROUP_ROUND_ROBIN,
      queue: :QUEUE_NEWEST,
      drop_newest: :DROP_NEWEST,
    }.freeze

    # Convert to a V1::Concurrency protobuf message
    # @return [V1::Concurrency]
    def to_proto
      proto_strategy = LIMIT_STRATEGY_MAP[@limit_strategy] || :CANCEL_IN_PROGRESS

      ::V1::Concurrency.new(
        expression: @expression,
        max_runs: @max_runs,
        limit_strategy: proto_strategy,
      )
    end
  end
end
