# frozen_string_literal: true

module Hatchet
  # Thread-local context variables for parent-child workflow dispatch linking.
  #
  # When the worker runner invokes a task block, it sets these thread-local
  # variables from the Action object. When a child workflow is spawned from
  # within that task (via `workflow.run` or `task.run`), the admin client reads
  # these variables to auto-populate parent linkage fields.
  #
  # IMPORTANT: These must be cleaned up in an `ensure` block after each task
  # execution to prevent leaking into the next task on the same thread (thread
  # pool reuse scenario).
  #
  # @example Setting context vars (runner side)
  #   Hatchet::ContextVars.set(
  #     workflow_run_id: action.workflow_run_id,
  #     step_run_id: action.step_run_id,
  #     worker_id: action.worker_id,
  #     action_key: action.key,
  #     additional_metadata: action.additional_metadata,
  #     retry_count: action.retry_count
  #   )
  #
  # @example Reading context vars (admin client side)
  #   parent_id = Hatchet::ContextVars.workflow_run_id
  #   parent_step_run_id = Hatchet::ContextVars.step_run_id
  module ContextVars
    KEYS = %i[
      hatchet_workflow_run_id
      hatchet_step_run_id
      hatchet_worker_id
      hatchet_action_key
      hatchet_additional_metadata
      hatchet_retry_count
    ].freeze

    class << self
      # Set all context variables for the current thread
      #
      # @param workflow_run_id [String] The workflow run ID
      # @param step_run_id [String] The step run ID
      # @param worker_id [String] The worker ID
      # @param action_key [String] The action key
      # @param additional_metadata [Hash] Additional metadata
      # @param retry_count [Integer] Retry count
      def set(workflow_run_id:, step_run_id:, worker_id:, action_key:, additional_metadata: {}, retry_count: 0)
        Thread.current[:hatchet_workflow_run_id] = workflow_run_id
        Thread.current[:hatchet_step_run_id] = step_run_id
        Thread.current[:hatchet_worker_id] = worker_id
        Thread.current[:hatchet_action_key] = action_key
        Thread.current[:hatchet_additional_metadata] = additional_metadata
        Thread.current[:hatchet_retry_count] = retry_count
      end

      # Clear all context variables for the current thread.
      # MUST be called in an ensure block after task execution.
      def clear
        KEYS.each { |key| Thread.current[key] = nil }
      end

      # @return [String, nil] The current workflow run ID
      def workflow_run_id
        Thread.current[:hatchet_workflow_run_id]
      end

      # @return [String, nil] The current step run ID
      def step_run_id
        Thread.current[:hatchet_step_run_id]
      end

      # @return [String, nil] The current worker ID
      def worker_id
        Thread.current[:hatchet_worker_id]
      end

      # @return [String, nil] The current action key
      def action_key
        Thread.current[:hatchet_action_key]
      end

      # @return [Hash] The current additional metadata
      def additional_metadata
        Thread.current[:hatchet_additional_metadata] || {}
      end

      # @return [Integer] The current retry count
      def retry_count
        Thread.current[:hatchet_retry_count] || 0
      end
    end

    # Thread-safe counter for tracking spawn indices per action key.
    # This provides deterministic child_index values for replay consistency.
    class SpawnIndexTracker
      def initialize
        @mutex = Mutex.new
        @indices = Hash.new(0)
      end

      # Get and increment the spawn index for the given action key.
      #
      # @param action_key [String] The action key
      # @return [Integer] The current spawn index (before increment)
      def next_index(action_key)
        @mutex.synchronize do
          index = @indices[action_key]
          @indices[action_key] = index + 1
          index
        end
      end

      # Reset the counter (e.g., between test runs)
      def reset
        @mutex.synchronize { @indices.clear }
      end
    end
  end
end
