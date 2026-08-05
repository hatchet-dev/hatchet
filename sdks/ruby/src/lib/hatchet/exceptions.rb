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

  # Raised when a gRPC call is rejected because the outgoing message exceeded the
  # configured max send size (see Hatchet::Config#grpc_max_send_message_length).
  class PayloadTooLargeError < Error
    # @return [Integer] The exact serialized size, in bytes, of the message that was rejected
    attr_reader :payload_bytes

    def initialize(payload_bytes, details)
      @payload_bytes = payload_bytes
      super(
        "Payload too large: attempted to send #{payload_bytes}, " \
        "which exceeds the gRPC max message size configured for this client. Increase " \
        "grpc_max_send_message_length in your Hatchet::Config (or the " \
        "HATCHET_CLIENT_GRPC_MAX_SEND_MESSAGE_LENGTH env var), or reduce the payload size. " \
        "(#{details})",
      )
    end
  end

  # Raised when a dedupe violation occurs (duplicate key)
  class DedupeViolationError < Error
    def initialize(message = "Dedupe violation: a run with this key already exists")
      super
    end
  end

  # Raised when an idempotency key collision occurs.
  # Contains the ID of the existing run that claimed the key.
  class IdempotencyCollisionError < Error
    # @return [String] The external ID of the existing workflow run
    attr_reader :existing_run_external_id

    def initialize(existing_run_external_id)
      @existing_run_external_id = existing_run_external_id
      super("idempotency key collision: existing run #{existing_run_external_id} already exists")
    end
  end

  # Raised when one or more runs in a bulk trigger collide on idempotency keys.
  # Contains the IDs of successfully triggered runs and the individual collision errors.
  class BulkTriggerIdempotencyCollisionError < Error
    # @return [Array<String>] External IDs of successfully triggered workflow runs
    attr_reader :successful_workflow_run_external_ids

    # @return [Array<IdempotencyCollisionError>] The individual collision errors
    attr_reader :collisions

    def initialize(successful_workflow_run_external_ids:, collisions:)
      @successful_workflow_run_external_ids = successful_workflow_run_external_ids
      @collisions = collisions
      super("idempotency key collision in bulk trigger: #{collisions.length} collision(s)")
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

  # If `grpc_error` (a rescued ::GRPC::ResourceExhausted) was caused by an oversized
  # outgoing message rather than e.g. a tenant quota limit, raises PayloadTooLargeError
  # with the exact serialized size of `request`. Otherwise this is a no-op.
  def self.raise_if_grpc_payload_too_large!(grpc_error, request)
    details = grpc_error.message.to_s
    lowered = details.downcase
    return unless lowered.include?("larger than") || lowered.include?("too large")

    raise PayloadTooLargeError.new(request.to_proto.bytesize, details)
  end
end
