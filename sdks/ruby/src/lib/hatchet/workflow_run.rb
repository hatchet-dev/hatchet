# frozen_string_literal: true

require "json"

module Hatchet
  # Reference to a running workflow, returned by `Workflow#run_no_wait`
  #
  # The result is the full workflow-level output keyed by task readable_id,
  # e.g. `{"step1" => {...}, "step2" => {...}}`.
  #
  # @example Get result from a run reference
  #   ref = workflow.run_no_wait(input)
  #   result = ref.result  # blocks until complete
  class WorkflowRunRef
    # @return [String] The workflow run ID
    attr_reader :workflow_run_id

    # @param workflow_run_id [String] The workflow run ID
    # @param client [Hatchet::Client] The Hatchet client (used as fallback)
    # @param listener [Hatchet::WorkflowRunListener, nil] Pooled gRPC listener
    def initialize(workflow_run_id:, client: nil, listener: nil)
      @workflow_run_id = workflow_run_id
      @client = client
      @listener = listener
      @result = nil
      @resolved = false
    end

    # Block until the workflow run completes and return the result.
    #
    # Uses the pooled gRPC `SubscribeToWorkflowRuns` listener when available.
    # Falls back to gRPC `GetRunDetails` polling otherwise.
    #
    # @param timeout [Integer] Maximum seconds to wait (default: 300)
    # @return [Hash] The workflow run output keyed by task readable_id
    # @raise [Hatchet::FailedRunError] if the workflow run failed
    # @raise [Timeout::Error] if the timeout is exceeded
    def result(timeout: 300)
      return @result if @resolved

      @result = if @listener
                  @listener.result(@workflow_run_id, timeout: timeout)
                else
                  poll_result_via_grpc(timeout)
                end

      @resolved = true
      @result
    end

    private

    # Fallback: poll via gRPC GetRunDetails (like Python SDK's sync path).
    def poll_result_via_grpc(timeout)
      deadline = Time.now + timeout
      retries = 0

      loop do
        raise Timeout::Error, "Timed out waiting for workflow run #{@workflow_run_id}" if Time.now > deadline

        begin
          response = @client.admin_grpc.get_run_details(external_id: @workflow_run_id)
        rescue StandardError
          retries += 1
          raise if retries > 10

          sleep 1
          next
        end

        status = response.status

        if !response.done && status != :COMPLETED && status != :FAILED && status != :CANCELLED
          sleep 1
          next
        end

        case status
        when :COMPLETED
          # Build result hash keyed by task readable_id (matching listener format)
          result = {}
          response.task_runs.each do |readable_id, detail|
            result[readable_id] = JSON.parse(detail.output) if detail.output && !detail.output.empty?
          end
          return result
        when :FAILED
          errors = response.task_runs
            .select { |_, d| d.respond_to?(:error) && d.error && !d.error.empty? }
            .map { |_, d| TaskRunError.new(d.error, task_run_external_id: d.external_id) }
          raise FailedRunError, errors.empty? ? [TaskRunError.new("Workflow run failed")] : errors
        when :CANCELLED
          raise Error, "Workflow run #{@workflow_run_id} was cancelled"
        end

        sleep 1
      end
    end
  end

  # Reference to a running standalone task, returned by `Task#run_no_wait`.
  #
  # Wraps a {WorkflowRunRef} and automatically extracts the task-specific
  # output from the workflow-level result. For a task named "my_task", calling
  # +result+ returns the task output directly (e.g. `{"value" => 42}`) instead
  # of the full keyed output (`{"my_task" => {"value" => 42}}`).
  #
  # @example
  #   ref = my_task.run_no_wait(input)
  #   output = ref.result  # => {"value" => 42}
  class TaskRunRef
    # @return [String] The workflow run ID
    attr_reader :workflow_run_id

    # @param workflow_run_ref [WorkflowRunRef] The underlying workflow run reference
    # @param task_name [Symbol, String] The task name used to extract output
    def initialize(workflow_run_ref:, task_name:)
      @workflow_run_ref = workflow_run_ref
      @task_name = task_name.to_s
      @workflow_run_id = workflow_run_ref.workflow_run_id
    end

    # Block until the task completes and return the extracted task output
    #
    # @param timeout [Integer] Maximum seconds to wait (default: 300)
    # @return [Hash] The task output
    # @raise [Hatchet::FailedRunError] if the workflow run failed
    # @raise [Timeout::Error] if the timeout is exceeded
    def result(timeout: 300)
      full_result = @workflow_run_ref.result(timeout: timeout)
      return full_result unless full_result.is_a?(Hash)

      full_result[@task_name] || full_result
    end
  end
end
