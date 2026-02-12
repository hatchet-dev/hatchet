# frozen_string_literal: true

module Hatchet
  # Context object passed to task execution blocks.
  #
  # Provides access to workflow run metadata, parent task outputs, logging,
  # cancellation, and other runtime capabilities.
  #
  # @example Accessing parent output
  #   workflow.task(:step2, parents: [step1]) do |input, ctx|
  #     parent_result = ctx.task_output(step1)
  #     { "sum" => parent_result["value"] + 1 }
  #   end
  class Context
    # @return [String] The workflow run ID
    attr_reader :workflow_run_id

    # @return [String] The step run ID
    attr_reader :step_run_id

    # @return [Hash] Additional metadata attached to this run
    attr_reader :additional_metadata

    # @return [Integer] Current retry count (0 on first attempt)
    attr_reader :retry_count

    # @return [Integer] Current attempt number (retry_count + 1)
    attr_reader :attempt_number

    # @return [Object, nil] Lifespan data shared across tasks in the worker
    attr_reader :lifespan

    # @return [Hash] Resolved dependency values
    attr_reader :deps

    # @return [Integer, nil] Task priority
    attr_reader :priority

    # @return [Hash, nil] Filter payload for event-triggered workflows
    attr_reader :filter_payload

    # @param workflow_run_id [String] The workflow run ID
    # @param step_run_id [String] The step run ID
    # @param action [Object, nil] The action object from the dispatcher
    # @param client [Hatchet::Client, nil] The Hatchet client
    # @param dispatcher_client [Hatchet::Clients::Grpc::Dispatcher, nil] gRPC dispatcher client
    # @param event_client [Hatchet::Clients::Grpc::EventClient, nil] gRPC event client
    # @param additional_metadata [Hash] Additional metadata
    # @param retry_count [Integer] Current retry count
    # @param lifespan [Object, nil] Lifespan data
    # @param parent_outputs [Hash] Hash of parent task name -> output
    # @param deps [Hash] Resolved dependency values
    # @param priority [Integer, nil] Priority
    # @param filter_payload [Hash, nil] Filter payload
    # @param worker_context [Object, nil] Worker context for worker-level operations
    def initialize(
      workflow_run_id:,
      step_run_id:,
      action: nil,
      client: nil,
      dispatcher_client: nil,
      event_client: nil,
      additional_metadata: {},
      retry_count: 0,
      lifespan: nil,
      parent_outputs: {},
      deps: {},
      priority: nil,
      filter_payload: nil,
      worker_context: nil
    )
      @workflow_run_id = workflow_run_id
      @step_run_id = step_run_id
      @action = action
      @client = client
      @dispatcher_client = dispatcher_client
      @event_client = event_client
      @additional_metadata = additional_metadata || {}
      @retry_count = retry_count
      @attempt_number = retry_count + 1
      @lifespan = lifespan
      @parent_outputs = parent_outputs || {}
      @deps = deps || {}
      @priority = priority
      @filter_payload = filter_payload
      @worker_context = worker_context
      @exit_flag = false
      @cancelled = false
    end

    # Get the output of a parent task
    #
    # @param task_ref [Task, Symbol, String] Reference to the parent task
    # @return [Hash, nil] The parent task's output
    def task_output(task_ref)
      key = case task_ref
            when Symbol then task_ref.to_s
            when String then task_ref
            else task_ref.respond_to?(:name) ? task_ref.name.to_s : task_ref.to_s
            end

      @parent_outputs[key] || @parent_outputs[key.to_sym]
    end

    # Check if a parent task was skipped
    #
    # @param task_ref [Task, Symbol, String] Reference to the parent task
    # @return [Boolean] true if the task was skipped
    def was_skipped?(task_ref)
      task_output(task_ref).nil?
    end

    # Log a message via the Hatchet logging system.
    # Sends the log to the server via gRPC if an event client is available.
    #
    # @param message [String, Hash] The message to log
    def log(message)
      msg = message.is_a?(String) ? message : message.inspect

      # Send log to server via gRPC
      if @event_client && @step_run_id
        begin
          @event_client.put_log(step_run_id: @step_run_id, message: msg)
        rescue => e
          @client&.logger&.warn("Failed to send log to server: #{e.message}")
        end
      end

      @client&.logger&.info(msg) || puts(msg)
    end

    # Cancel the current workflow run
    def cancel
      @cancelled = true
      @exit_flag = true
      if @client && @workflow_run_id
        @client.runs.cancel(@workflow_run_id) rescue nil
      end
    end

    # Check if the task has been cancelled
    #
    # @return [Boolean] true if cancellation has been requested
    def exit_flag
      @exit_flag
    end

    # Refresh the execution timeout for this task.
    #
    # @param duration [Integer, String] New timeout in seconds, or a duration string
    def refresh_timeout(duration)
      return unless @dispatcher_client && @step_run_id

      @dispatcher_client.refresh_timeout(
        step_run_id: @step_run_id,
        timeout_seconds: duration
      )
    end

    # Release the worker slot before the task completes.
    # Useful for tasks that have a resource-intensive phase followed by a lighter phase.
    def release_slot
      return unless @dispatcher_client && @step_run_id

      @dispatcher_client.release_slot(step_run_id: @step_run_id)
    end

    # Put a stream chunk for real-time streaming output.
    #
    # @param data [String] The chunk data to stream
    def put_stream(data)
      return unless @event_client && @step_run_id

      @event_client.put_stream(step_run_id: @step_run_id, data: data)
    end

    # Get errors from upstream task runs (used in on_failure tasks)
    #
    # @return [Array<TaskRunError>] Task run errors
    def task_run_errors
      @action&.respond_to?(:task_run_errors) ? @action.task_run_errors : []
    end

    # Get the error from a specific upstream task (used in on_failure tasks)
    #
    # @param task_ref [Task, Symbol, String] Reference to the failed task
    # @return [TaskRunError, nil] The task run error, or nil
    def get_task_run_error(task_ref)
      key = case task_ref
            when Symbol then task_ref.to_s
            when String then task_ref
            else task_ref.respond_to?(:name) ? task_ref.name.to_s : task_ref.to_s
            end

      task_run_errors.find { |e| e.respond_to?(:task_name) && e.task_name == key }
    end

    # Access the worker context for worker-level operations
    #
    # @return [WorkerContext, nil]
    def worker
      @worker_context
    end
  end
end
