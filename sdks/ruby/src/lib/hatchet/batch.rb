# frozen_string_literal: true

module Hatchet
  # Configures a task as a batch task: concurrent runs are buffered and dispatched together
  # as a single execution once max_size is reached, max_interval_ms elapses, or (if
  # group_key is set) once group_max_runs concurrent batches per group are exceeded.
  #
  # When batch is set on a task, the task's block receives a single Hash argument mapping
  # each buffered run's task-run external id to that run's input, instead of a single input
  # value. Unless broadcast_output is true, the block must return a Hash with the exact same
  # key set, mapping each id to that run's output.
  #
  # @example
  #   Hatchet::BatchTaskConfig.new(max_size: 3, max_interval_ms: 200, group_key: "input.group")
  class BatchTaskConfig
    # @return [Integer] Maximum number of items buffered before the batch is flushed
    attr_reader :max_size

    # @return [Integer, nil] Maximum time to wait before flushing a partially-filled batch, in milliseconds
    attr_reader :max_interval_ms

    # @return [String, nil] CEL expression evaluated against each item's input to partition items into independent batches
    attr_reader :group_key

    # @return [Integer, nil] Maximum number of concurrent batches per group
    attr_reader :group_max_runs

    # @return [Boolean] When true, the block returns a single value broadcast to every member of the batch
    attr_reader :broadcast_output

    # @param max_size [Integer] Maximum items per batch. Must be positive.
    # @param max_interval_ms [Integer, nil] Time before batch flushes, in milliseconds. Must be positive when provided.
    # @param group_key [String, nil] CEL expression to partition batches, e.g. "input.group"
    # @param group_max_runs [Integer, nil] Concurrent batches per group. Must be positive when provided.
    # @param broadcast_output [Boolean] Whether the block's return value is broadcast to every batch member
    def initialize(max_size:, max_interval_ms: nil, group_key: nil, group_max_runs: nil, broadcast_output: false)
      raise ArgumentError, "max_size must be positive" unless max_size.positive?
      raise ArgumentError, "max_interval_ms must be positive when provided" if !max_interval_ms.nil? && !max_interval_ms.positive?
      raise ArgumentError, "group_max_runs must be positive when provided" if !group_max_runs.nil? && !group_max_runs.positive?

      @max_size = max_size
      @max_interval_ms = max_interval_ms
      @group_key = group_key
      @group_max_runs = group_max_runs
      @broadcast_output = broadcast_output
    end

    # @return [V1::TaskBatchConfig]
    def to_proto
      args = { batch_max_size: @max_size, broadcast_output: @broadcast_output }
      args[:batch_max_interval_ms] = @max_interval_ms if @max_interval_ms
      args[:batch_group_key] = @group_key if @group_key
      args[:batch_group_max_runs] = @group_max_runs if @group_max_runs

      ::V1::TaskBatchConfig.new(**args)
    end
  end
end
