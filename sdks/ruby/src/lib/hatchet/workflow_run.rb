# frozen_string_literal: true

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
    # @param client [Hatchet::Client] The Hatchet client for fetching results
    def initialize(workflow_run_id:, client: nil)
      @workflow_run_id = workflow_run_id
      @client = client
      @result = nil
      @resolved = false
    end

    # Block until the workflow run completes and return the result
    #
    # @param timeout [Integer] Maximum seconds to wait (default: 300)
    # @return [Hash] The workflow run output
    # @raise [Hatchet::FailedRunError] if the workflow run failed
    # @raise [Timeout::Error] if the timeout is exceeded
    def result(timeout: 300)
      return @result if @resolved

      deadline = Time.now + timeout

      loop do
        raise Timeout::Error, "Timed out waiting for workflow run #{@workflow_run_id}" if Time.now > deadline

        run = @client.runs.get(@workflow_run_id)
        status = run.respond_to?(:run) ? run.run.status : run.status

        case status.to_s
        when "COMPLETED"
          @result = run.respond_to?(:run) ? run.run.output : run.output
          @resolved = true
          return @result
        when "FAILED"
          error_msg = run.respond_to?(:run) ? run.run.error : (run.respond_to?(:error) ? run.error : "Workflow run failed")
          raise FailedRunError.new([TaskRunError.new(error_msg.to_s)])
        when "CANCELLED"
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
