# frozen_string_literal: true

require "concurrent"
require "json"

module Hatchet
  module WorkerRuntime
    # Executes task actions in a thread pool, managing concurrency slots
    # and context variable propagation.
    #
    # The runner receives actions from the action listener, looks up the
    # corresponding task block, sets up context variables, executes the task,
    # and sends the result back to the dispatcher.
    #
    # @example
    #   runner = Runner.new(
    #     workflows: [my_workflow],
    #     slots: 10,
    #     dispatcher_client: dispatcher_grpc,
    #     event_client: event_grpc,
    #     logger: logger,
    #     client: hatchet_client
    #   )
    #   runner.execute(action)
    class Runner
      # @param workflows [Array<Workflow, Task>] Registered workflows
      # @param slots [Integer] Maximum concurrent task slots
      # @param dispatcher_client [Hatchet::Clients::Grpc::Dispatcher] gRPC dispatcher client
      # @param event_client [Hatchet::Clients::Grpc::EventClient] gRPC event client
      # @param logger [Logger] Logger instance
      # @param client [Hatchet::Client] The Hatchet client
      # @param lifespan_data [Object, nil] Shared lifespan data
      def initialize(workflows:, slots:, dispatcher_client:, event_client:, logger:, client:, lifespan_data: nil)
        @workflows = workflows
        @slots = slots
        @dispatcher_client = dispatcher_client
        @event_client = event_client
        @logger = logger
        @client = client
        @lifespan_data = lifespan_data

        # Thread pool with semaphore for slot management
        @pool = Concurrent::FixedThreadPool.new(slots)
        @semaphore = Concurrent::Semaphore.new(slots)

        # Build task lookup table
        @task_map = build_task_map
      end

      # Execute an action (task assignment) in the thread pool.
      #
      # @param action [AssignedAction] The action from the dispatcher
      def execute(action)
        @semaphore.acquire

        @pool.post do
          execute_task(action)
        ensure
          @semaphore.release
        end
      end

      # Gracefully shutdown the runner.
      #
      # @param timeout [Integer] Seconds to wait for in-progress tasks
      def shutdown(timeout: 30)
        @pool.shutdown
        @pool.wait_for_termination(timeout)
      end

      private

      def build_task_map
        map = {}

        @workflows.each do |wf|
          if wf.is_a?(Workflow)
            service_name = @client.config.apply_namespace(wf.name.downcase)

            wf.tasks.each do |name, task|
              # TODO: is this what we do across sdks...
              key = "#{service_name}:#{name}".downcase
              map[key] = task
            end

            if wf.on_failure
              map["#{service_name}:on_failure"] = wf.on_failure
            end

            if wf.on_success
              map["#{service_name}:on_success"] = wf.on_success
            end
          elsif wf.is_a?(Task)
            # Standalone task -- the workflow wrapper has the same name
            workflow = wf.workflow
            if workflow
              service_name = @client.config.apply_namespace(workflow.name.downcase)
              map["#{service_name}:#{wf.name}".downcase] = wf
            end
          end
        end

        map
      end

      def execute_task(action)
        # Set context vars BEFORE executing the task
        ContextVars.set(
          workflow_run_id: action.workflow_run_id,
          step_run_id: action.task_run_external_id,
          worker_id: @dispatcher_client.respond_to?(:worker_id) ? @dispatcher_client.worker_id : "",
          action_key: action.action_id,
          additional_metadata: parse_metadata(action),
          retry_count: action.retry_count
        )

        # Send STARTED event
        send_started(action)

        # Look up the task by action_id (service_name:task_name), case-insensitive
        task_key = action.action_id.downcase
        task = @task_map[task_key]

        unless task
          @logger.error("No task found for action: #{task_key}")
          send_failure(action, StandardError.new("No task found for action: #{task_key}"), retryable: false)
          return
        end

        # Parse parent outputs from the action payload
        parent_outputs = parse_parent_outputs(action)

        # Build the context
        ctx_class = task.durable ? DurableContext : Context
        ctx = ctx_class.new(
          workflow_run_id: action.workflow_run_id,
          step_run_id: action.task_run_external_id,
          action: action,
          client: @client,
          dispatcher_client: @dispatcher_client,
          event_client: @event_client,
          additional_metadata: ContextVars.additional_metadata,
          retry_count: action.retry_count,
          lifespan: @lifespan_data,
          parent_outputs: parent_outputs
        )

        # Parse input from action payload
        input = parse_input(action)

        # Resolve dependencies if the task has any
        if task.deps && !task.deps.empty?
          ctx.deps = resolve_dependencies(task.deps, input, ctx)
        end

        # Execute the task
        result = task.call(input, ctx)

        # Send result back to dispatcher
        send_result(action, result)
      rescue NonRetryableError => e
        @logger.error("Non-retryable error in task #{action.action_id}: #{e.message}")
        send_failure(action, e, retryable: false)
      rescue => e
        @logger.error("Error in task #{action.action_id}: #{e.message}")
        send_failure(action, e, retryable: true)
      ensure
        # CRITICAL: Clean up context vars to prevent leaking to next task
        ContextVars.clear
      end

      # Send a STARTED event to the dispatcher.
      def send_started(action)
        @dispatcher_client.send_step_action_event(
          action: action,
          event_type: :STEP_EVENT_TYPE_STARTED,
          payload: "{}"
        )
      rescue => e
        @logger.warn("Failed to send STARTED event: #{e.message}")
      end

      # Send a COMPLETED event with the task result.
      def send_result(action, result)
        payload = result.nil? ? "{}" : JSON.generate(result)

        @dispatcher_client.send_step_action_event(
          action: action,
          event_type: :STEP_EVENT_TYPE_COMPLETED,
          payload: payload,
          retry_count: action.retry_count
        )
      end

      # Send a FAILED event with error details.
      def send_failure(action, error, retryable:)
        payload = JSON.generate({ "error" => error.message })

        @dispatcher_client.send_step_action_event(
          action: action,
          event_type: :STEP_EVENT_TYPE_FAILED,
          payload: payload,
          retry_count: action.retry_count,
          should_not_retry: !retryable
        )
      end

      # Resolve task dependencies in two passes:
      # 1. Simple deps (2-arg lambdas: input, ctx)
      # 2. Composite deps (3-arg lambdas: input, ctx, resolved_deps)
      def resolve_dependencies(deps_hash, input, ctx)
        resolved = {}
        deferred = {}

        deps_hash.each do |name, dep_fn|
          if dep_fn.arity.abs <= 2
            resolved[name] = dep_fn.call(input, ctx)
          else
            deferred[name] = dep_fn
          end
        end

        deferred.each do |name, dep_fn|
          resolved[name] = dep_fn.call(input, ctx, resolved)
        end

        resolved
      end

      # Parse additional metadata from the action.
      def parse_metadata(action)
        raw = action.respond_to?(:additional_metadata) ? action.additional_metadata : nil
        return {} if raw.nil? || raw.to_s.empty?

        JSON.parse(raw)
      rescue JSON::ParserError
        {}
      end

      # Parse parent task outputs from the action payload.
      def parse_parent_outputs(action)
        raw = action.respond_to?(:action_payload) ? action.action_payload : nil
        return {} if raw.nil? || raw.to_s.empty?

        parsed = JSON.parse(raw)
        parsed.is_a?(Hash) && parsed.key?("parents") ? parsed["parents"] : {}
      rescue JSON::ParserError
        {}
      end

      # Parse task input from the action payload.
      def parse_input(action)
        raw = action.respond_to?(:action_payload) ? action.action_payload : nil
        return {} if raw.nil? || raw.to_s.empty?

        parsed = JSON.parse(raw)
        # The input is typically stored under the "input" key in the payload
        parsed.is_a?(Hash) && parsed.key?("input") ? parsed["input"] : parsed
      rescue JSON::ParserError
        {}
      end
    end
  end
end
