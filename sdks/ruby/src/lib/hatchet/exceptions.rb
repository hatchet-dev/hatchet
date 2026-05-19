# frozen_string_literal: true

module Hatchet
  # Base error class for all Hatchet errors (already defined in hatchet-sdk.rb as Hatchet::Error)

  # Raised when a task should not be retried
  class NonRetryableError < Error
    def initialize(message = "This task should not be retried")
      super
    end
  end

  # Raised when the tenant has exceeded its resource limits (e.g. task run quota)
  class ResourceExhaustedError < Error
    def initialize(message = "Resource exhausted: tenant has reached its task runs limit")
      super
    end
  end

  # Raised when a dedupe violation occurs (duplicate key)
  class DedupeViolationError < Error
    def initialize(message = "Dedupe violation: a run with this key already exists")
      super
    end
  end

  # Represents an error from a failed task run
  class TaskRunError < Error
    # @return [String] The external ID of the failed task run
    attr_reader :task_run_external_id

    # @return [String] The error message from the task
    attr_reader :exc

    # @param message [String] Error message
    # @param task_run_external_id [String] The external ID of the failed task run
    def initialize(message, task_run_external_id: nil)
      @task_run_external_id = task_run_external_id
      @exc = message
      super(message)
    end
  end

  # Raised by the engine when durable-task execution detects a non-deterministic
  # replay (the workflow did something different compared to the recorded log).
  class NonDeterminismError < Error
    # @return [String, nil]
    attr_reader :task_external_id
    # @return [Integer, nil]
    attr_reader :invocation_count
    # @return [Integer, nil]
    attr_reader :node_id

    def initialize(message, task_external_id: nil, invocation_count: nil, node_id: nil)
      @task_external_id = task_external_id
      @invocation_count = invocation_count
      @node_id = node_id
      super(message)
    end
  end

  # Raised inside a durable task thread when the eviction manager decides to
  # evict that invocation (e.g. TTL expired, capacity pressure, worker shutdown).
  class DurableTaskEvictedError < Error
    def initialize(message = "Durable task evicted")
      super
    end
  end

  # Raised when a workflow run fails with one or more task errors
  class FailedRunError < Error
    # @return [Array<TaskRunError>] The individual task run errors
    attr_reader :exceptions

    # @param exceptions [Array<TaskRunError>]
    def initialize(exceptions)
      @exceptions = exceptions
      messages = exceptions.map(&:message).join("; ")
      super("Workflow run failed with #{exceptions.length} error(s): #{messages}")
    end
  end
end
