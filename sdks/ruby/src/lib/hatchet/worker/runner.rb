# frozen_string_literal: true

require "concurrent"
require "json"
require "monitor"

module Hatchet
  module WorkerRuntime
    # Executes task actions in a thread pool, managing concurrency slots
    # and context variable propagation.
    #
    # The runner receives actions from the action listener, looks up the
    # corresponding task block, sets up context variables, executes the task,
    # and sends the result back to the dispatcher. For durable tasks, it wires
    # the :class:`DurableContext` up to the shared
    # :class:`DurableEventListener` and per-run
    # :class:`DurableEviction::DurableEvictionManager` when the engine supports
    # eviction.
    #
    # @example
    #   runner = Runner.new(
    #     workflows: [my_workflow],
    #     slots: 10,
    #     dispatcher_client: dispatcher_grpc,
    #     event_client: event_grpc,
    #     logger: logger,
    #     client: hatchet_client,
    #     engine_version: "v0.80.0",
    #     durable_slots: 10,
    #   )
    #   runner.execute(action)
    class Runner
      STARTED_EVENT_RETRY_COUNT = 5
      STARTED_EVENT_STOP = Object.new

      # @param workflows [Array<Workflow, Task>] Registered workflows
      # @param slots [Integer] Maximum concurrent task slots
      # @param dispatcher_client [Hatchet::Clients::Grpc::Dispatcher] gRPC dispatcher client
      # @param event_client [Hatchet::Clients::Grpc::EventClient] gRPC event client
      # @param logger [Logger] Logger instance
      # @param client [Hatchet::Client] The Hatchet client
      # @param engine_version [String, nil] Engine semantic version (from GetVersion)
      # @param durable_slots [Integer, nil] Separate slot count for durable tasks; defaults to ``slots``.
      # @param worker_id [String, nil] Worker ID from registration; stamped onto durable calls that need it.
      def initialize(
        workflows:,
        slots:,
        dispatcher_client:,
        event_client:,
        logger:,
        client:,
        engine_version: nil,
        durable_slots: nil,
        worker_id: nil
      )
        @workflows = workflows
        @slots = slots
        @durable_slots = durable_slots || slots
        @dispatcher_client = dispatcher_client
        @event_client = event_client
        @logger = logger
        @client = client
        @engine_version = engine_version
        @worker_id = worker_id

        @pool = Concurrent::FixedThreadPool.new(slots)
        @semaphore = Concurrent::Semaphore.new(slots)

        @batch_mu = Mutex.new
        @batch_states = {}
        @batch_map = {}

        @task_map = build_task_map

        @contexts_mu = Monitor.new
        @contexts = {}
        @task_threads = {}
        @step_action_event_queue = Queue.new
        @step_action_event_thread = Thread.new { process_step_action_events }

        @has_durable_tasks = @task_map.values.any?(&:durable)
        @supports_durable_eviction = supports_durable_eviction?

        @durable_event_listener = build_durable_event_listener
        @eviction_manager = nil
        @eviction_manager_mu = Mutex.new
      end

      # @return [WorkerRuntime::DurableEviction::DurableEvictionManager, nil]
      attr_reader :eviction_manager

      # @return [WorkerRuntime::DurableEventListener, nil]
      attr_reader :durable_event_listener

      # Execute an action (task assignment) in the thread pool.
      #
      # @param action [AssignedAction] The action from the dispatcher
      def execute(action)
        # START_BATCH is a lightweight control signal — no slot consumed
        if action.respond_to?(:action_type) && action.action_type == :START_BATCH
          handle_start_batch_action(action)
          return
        end

        # Batch items must all be simultaneously in-flight so the batch state
        # machine can accumulate every peer item before flushing. If they went
        # through the semaphore the first batch would fill all slots while
        # blocking on the result queue, preventing the remaining items from
        # starting — deadlock. Raw threads let all items wait concurrently.
        has_batch_id = action.respond_to?(:batchId) && action.batchId && !action.batchId.empty?
        if has_batch_id && @batch_map.key?(action.action_id.downcase)
          t = Thread.new { execute_batch_item(action) }
          t.abort_on_exception = false
          return
        end

        ensure_eviction_manager_started(action)

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
        if @eviction_manager
          begin
            @eviction_manager.evict_all_waiting
          rescue StandardError => e
            @logger.warn("Runner: failed to evict waiting durable runs during shutdown: #{e.class}: #{e.message}")
          end
        end

        @pool.shutdown
        @pool.wait_for_termination(timeout)
        stop_step_action_event_thread

        @durable_event_listener&.stop
      end

      private

      def supports_durable_eviction?
        return false unless @engine_version

        !Hatchet::EngineVersion.semver_less_than?(
          @engine_version,
          Hatchet::MinEngineVersion::DURABLE_EVICTION,
        )
      end

      def build_durable_event_listener
        return nil unless @has_durable_tasks && @supports_durable_eviction

        DurableEventListener.new(
          config: @client.config,
          channel: @client.channel,
          logger: @logger,
          on_server_evict: method(:handle_server_evict),
        )
      end

      def ensure_eviction_manager_started(_action)
        return unless @has_durable_tasks
        return unless @supports_durable_eviction

        @durable_event_listener&.ensure_started(@worker_id) if @worker_id
        return if @eviction_manager

        @eviction_manager_mu.synchronize do
          return if @eviction_manager

          @eviction_manager = DurableEviction::DurableEvictionManager.new(
            durable_slots: @durable_slots,
            cancel_local: method(:eviction_cancel_local),
            request_eviction_with_ack: method(:eviction_request_with_ack),
            logger: @logger,
          )
          @eviction_manager.start
        end
      end

      def handle_server_evict(durable_task_external_id, invocation_count)
        return unless @eviction_manager

        @eviction_manager.handle_server_eviction(durable_task_external_id, invocation_count)
      end

      def eviction_cancel_local(action_key)
        thread, ctx = @contexts_mu.synchronize do
          [@task_threads[action_key], @contexts[action_key]]
        end

        if @durable_event_listener && ctx.is_a?(DurableContext)
          @durable_event_listener.cleanup_task_state(ctx.step_run_id, ctx.invocation_count || 1)
        end

        thread&.raise(Hatchet::DurableTaskEvictedError.new)
      end

      def eviction_request_with_ack(action_key, rec)
        return unless @durable_event_listener

        invocation_count = 1
        @contexts_mu.synchronize do
          ctx = @contexts[action_key]
          invocation_count = ctx.invocation_count if ctx.is_a?(DurableContext) && ctx.invocation_count
        end

        @durable_event_listener.send_evict_invocation(
          rec.step_run_id,
          invocation_count,
          reason: rec.eviction_reason,
        )
      end

      # Handle a START_BATCH control action: mark the batch as started and flush if ready.
      def handle_start_batch_action(action)
        action_id = action.action_id.downcase
        batch_id = action.batchId
        expected_size = action.respond_to?(:batchStart) && action.batchStart ? action.batchStart.expectedSize.to_i : 0

        @logger.debug("Runner: START_BATCH action_id=#{action_id} batch_id=#{batch_id} expected_size=#{expected_size}")

        flush_now = false
        flush_task = nil
        flush_state = nil

        @batch_mu.synchronize do
          states = (@batch_states[action_id] ||= {})
          state = (states[batch_id] ||= { expected_size: expected_size, items: {}, started: false })

          state[:started] = true
          state[:expected_size] = expected_size if expected_size.positive?

          if state[:expected_size].positive? && state[:items].size >= state[:expected_size]
            flush_now = true
            flush_task = @batch_map[action_id]
            flush_state = states.delete(batch_id)
          end
        end

        return unless flush_now && flush_task && flush_state

        flush_batch(flush_task, flush_state)
      rescue StandardError => e
        @logger.error("Runner: error in handle_start_batch_action: #{e.class}: #{e.message}")
      end

      # Execute a single batch item. Runs in a raw thread (outside the semaphore-protected pool)
      # so batch items can all wait simultaneously without exhausting the slot pool.
      def execute_batch_item(action)
        action_key = action_key_for(action)
        task_key = action.action_id.downcase
        task = @task_map[task_key]

        unless task
          @logger.error("Runner: no task found for batch action: #{task_key}")
          return
        end

        prepare_action_execution(action)
        ctx = build_context(action, task)
        input = parse_input(action)

        batch_id = action.batchId
        expected_size = action.respond_to?(:batchSize) && action.batchSize ? action.batchSize.to_i : 0
        batch_index = action.respond_to?(:batchIndex) && action.batchIndex ? action.batchIndex.to_i : 0

        result_queue = SizedQueue.new(1)
        flush_now = false
        flush_state = nil

        @batch_mu.synchronize do
          states = (@batch_states[task_key] ||= {})
          state = (states[batch_id] ||= { expected_size: expected_size, items: {}, started: false })

          state[:expected_size] = expected_size if expected_size.positive? && state[:expected_size].zero?

          if state[:items].key?(batch_index)
            @logger.error("Runner: duplicate batch index #{batch_index} for batch #{batch_id}")
            result_queue.push([:error, Error.new("Duplicate batch index #{batch_index}")])
          else
            state[:items][batch_index] = { input: input, ctx: ctx, result_queue: result_queue }

            if state[:started] && state[:expected_size].positive? && state[:items].size >= state[:expected_size]
              flush_now = true
              flush_state = states.delete(batch_id)
            end
          end
        end

        flush_batch(task, flush_state) if flush_now

        result_type, value = result_queue.pop

        if result_type == :ok
          send_result(action, value)
        else
          send_failure(action, value, retryable: true)
        end
      rescue NonRetryableError => e
        @logger.error("Runner: non-retryable error in batch item #{action_key}: #{e.message}")
        send_failure(action, e, retryable: false)
      rescue StandardError => e
        @logger.error("Runner: error in batch item #{action_key}: #{e.message}")
        send_failure(action, e, retryable: true)
      ensure
        ContextVars.clear
      end

      # Execute the batch fn for all collected items and distribute results.
      def flush_batch(task, state)
        sorted_items = state[:items].sort_by { |idx, _| idx }.map { |_, entry| entry }
        tasks = sorted_items.map { |e| [e[:input], e[:ctx]] }

        begin
          results = task.batch_fn.call(tasks)

          unless results.is_a?(Array) && results.length == sorted_items.length
            raise Error, "Batch fn returned #{results.is_a?(Array) ? results.length : results.class} " \
                         "outputs for #{sorted_items.length} inputs"
          end

          sorted_items.each_with_index do |entry, i|
            entry[:result_queue].push([:ok, results[i]])
          end
        rescue StandardError => e
          @logger.error("Runner: batch fn failed: #{e.class}: #{e.message}")
          sorted_items.each { |entry| entry[:result_queue].push([:error, e]) }
        end
      end

      def build_task_map
        map = {}

        @workflows.each do |wf|
          if wf.is_a?(Workflow)
            service_name = @client.config.apply_namespace(wf.name.downcase)

            wf.tasks.each do |name, task|
              key = "#{service_name}:#{name}".downcase
              map[key] = task
              @batch_map[key] = task if task.batch?
            end

            map["#{service_name}:on_failure"] = wf.on_failure if wf.on_failure

            map["#{service_name}:on_success"] = wf.on_success if wf.on_success
          elsif wf.is_a?(Task)
            workflow = wf.workflow
            if workflow
              service_name = @client.config.apply_namespace(workflow.name.downcase)
              key = "#{service_name}:#{wf.name}".downcase
              map[key] = wf
              @batch_map[key] = wf if wf.batch?
            end
          end
        end

        map
      end

      def execute_task(action)
        action_key = nil
        prepare_action_execution(action)

        task = find_task(action)
        return unless task

        ctx = build_context(action, task)
        action_key = action_key_for(action)
        configure_durable_context(task, ctx, action, action_key)
        track_action_context(action_key, ctx)
        run_task(action, task, ctx)
      rescue Hatchet::DurableTaskEvictedError => e
        @logger.info("Durable task evicted: #{action.action_id}: #{e.message}")
      rescue NonRetryableError => e
        @logger.error("Non-retryable error in task #{action.action_id}: #{e.message}")
        send_failure(action, e, retryable: false)
      rescue StandardError => e
        @logger.error("Error in task #{action.action_id}: #{e.message}")
        send_failure(action, e, retryable: true)
      ensure
        # CRITICAL: Clean up context vars to prevent leaking to next task
        cleanup_action(action_key) if action_key
        ContextVars.clear
      end

      def prepare_action_execution(action)
        @logger.debug(
          "Runner: received action action_id=#{action.action_id} step_run_id=#{action.task_run_external_id} " \
          "retry_count=#{action.retry_count} durable_invocation_count=#{extract_invocation_count(action)}",
        )
        ContextVars.set(
          workflow_run_id: action.workflow_run_id,
          step_run_id: action.task_run_external_id,
          worker_id: worker_id,
          action_key: action.action_id,
          additional_metadata: parse_metadata(action),
          retry_count: action.retry_count,
        )
        send_started(action)
      end

      def find_task(action)
        task_key = action.action_id.downcase
        task = @task_map[task_key]
        return task if task

        @logger.error("No task found for action: #{task_key}")
        send_failure(action, StandardError.new("No task found for action: #{task_key}"), retryable: false)
        nil
      end

      def build_context(action, task)
        ctx_class = task.durable ? DurableContext : Context
        ctx_class.new(
          workflow_run_id: action.workflow_run_id,
          step_run_id: action.task_run_external_id,
          action: action,
          client: @client,
          dispatcher_client: @dispatcher_client,
          event_client: @event_client,
          additional_metadata: ContextVars.additional_metadata,
          retry_count: action.retry_count,
          parent_outputs: parse_parent_outputs(action),
          worker_id: ContextVars.worker_id,
        )
      end

      def configure_durable_context(task, ctx, action, action_key)
        return unless task.durable && ctx.is_a?(DurableContext)

        ctx.eviction_manager = @eviction_manager
        ctx.action_key = action_key
        ctx.durable_event_listener = @durable_event_listener
        ctx.invocation_count = extract_invocation_count(action)
        ctx.engine_version = @engine_version
        register_durable_run(task, action, action_key, ctx)
      end

      def register_durable_run(task, action, action_key, ctx)
        return unless @eviction_manager && task.eviction_policy

        @eviction_manager.register_run(
          action_key,
          step_run_id: action.task_run_external_id,
          invocation_count: ctx.invocation_count,
          eviction_policy: task.eviction_policy,
        )
      end

      def track_action_context(action_key, ctx)
        @contexts_mu.synchronize do
          @contexts[action_key] = ctx
          @task_threads[action_key] = Thread.current
        end
      end

      def run_task(action, task, ctx)
        input = parse_input(action)
        ctx.deps = resolve_dependencies(task.deps, input, ctx) if task.deps && !task.deps.empty?
        result = task.call(input, ctx)
        send_result(action, result)
      end

      def action_key_for(action)
        "#{action.task_run_external_id}/#{action.retry_count}"
      end

      def extract_invocation_count(action)
        value = action.respond_to?(:durable_task_invocation_count) ? action.durable_task_invocation_count : nil
        value.nil? || value.zero? ? 1 : value
      end

      def worker_id
        @dispatcher_client.respond_to?(:worker_id) ? @dispatcher_client.worker_id : ""
      end

      def cleanup_action(action_key)
        @contexts_mu.synchronize do
          @contexts.delete(action_key)
          @task_threads.delete(action_key)
        end

        @eviction_manager&.unregister_run(action_key)
      end

      def send_started(action)
        @step_action_event_queue << action
      end

      def process_step_action_events
        loop do
          action = @step_action_event_queue.pop
          break if action.equal?(STARTED_EVENT_STOP)

          send_started_with_retry(action)
        end
      rescue ClosedQueueError
        nil
      end

      def send_started_with_retry(action)
        attempt = 1

        loop do
          @dispatcher_client.send_step_action_event(
            action: action,
            event_type: :STEP_EVENT_TYPE_STARTED,
            payload: "{}",
            retry_count: action.retry_count,
          )
          return
        rescue StandardError => e
          @logger.warn(
            "Failed to send STARTED event (#{attempt}/#{STARTED_EVENT_RETRY_COUNT}): #{e.message}",
          )
          raise e if attempt >= STARTED_EVENT_RETRY_COUNT

          sleep started_event_backoff_seconds(attempt)
          attempt += 1
        end
      end

      def started_event_backoff_seconds(attempt)
        base = 0.1
        jitter = rand * base
        [((base * (2**attempt)) + jitter), 1.0].min
      end

      def stop_step_action_event_thread
        return unless @step_action_event_thread

        @step_action_event_queue << STARTED_EVENT_STOP
        @step_action_event_thread.join(5)
      rescue StandardError
        nil
      end

      def send_result(action, result)
        payload = result.nil? ? "{}" : JSON.generate(result)

        @dispatcher_client.send_step_action_event(
          action: action,
          event_type: :STEP_EVENT_TYPE_COMPLETED,
          payload: payload,
          retry_count: action.retry_count,
        )
      end

      def send_failure(action, error, retryable:)
        payload = JSON.generate({ "error" => error.message })

        @dispatcher_client.send_step_action_event(
          action: action,
          event_type: :STEP_EVENT_TYPE_FAILED,
          payload: payload,
          retry_count: action.retry_count,
          should_not_retry: !retryable,
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

      def parse_metadata(action)
        raw = action.respond_to?(:additional_metadata) ? action.additional_metadata : nil
        return {} if raw.nil? || raw.to_s.empty?

        JSON.parse(raw)
      rescue JSON::ParserError
        {}
      end

      def parse_parent_outputs(action)
        raw = action.respond_to?(:action_payload) ? action.action_payload : nil
        return {} if raw.nil? || raw.to_s.empty?

        parsed = JSON.parse(raw)
        parsed.is_a?(Hash) && parsed.key?("parents") ? parsed["parents"] : {}
      rescue JSON::ParserError
        {}
      end

      def parse_input(action)
        raw = action.respond_to?(:action_payload) ? action.action_payload : nil
        return {} if raw.nil? || raw.to_s.empty?

        parsed = JSON.parse(raw)
        parsed.is_a?(Hash) && parsed.key?("input") ? parsed["input"] : parsed
      rescue JSON::ParserError
        {}
      end
    end
  end
end
