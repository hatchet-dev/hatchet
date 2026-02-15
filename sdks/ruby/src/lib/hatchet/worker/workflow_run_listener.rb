# frozen_string_literal: true

require "json"

module Hatchet
  # Thread-safe pooled gRPC listener for workflow run completion events.
  #
  # Maintains a single bidirectional gRPC stream to
  # `Dispatcher.SubscribeToWorkflowRuns`. Multiple callers share the stream;
  # each subscription sends a request on the outgoing side and blocks until the
  # matching `WorkflowRunEvent` arrives on the incoming side.
  #
  # Modeled on the Python SDK's `PooledWorkflowRunListener`.
  #
  # @example
  #   listener = WorkflowRunListener.new(config: config, channel: channel)
  #   result = listener.result("workflow-run-id-123")
  #   # => {"my_task" => {"value" => 42}}
  class WorkflowRunListener
    RETRY_INTERVAL = 3 # seconds between reconnect attempts
    MAX_RETRIES    = 5

    DEDUPE_MESSAGE = "DUPLICATE_WORKFLOW_RUN"

    def initialize(config:, channel:)
      @config  = config
      @channel = channel
      @logger  = config.logger

      # Outgoing request queue. The Enumerator fed to the bidi call
      # pulls from this queue.
      @request_queue = Queue.new

      # Subscription management (protected by @mu)
      @mu = Mutex.new
      # workflow_run_id -> Array<Queue>  (subscriber queues)
      @subscriptions = Hash.new { |h, k| h[k] = [] }

      # Background reader thread (lazy-started)
      @reader_thread = nil
      @started = false
    end

    # Subscribe to a workflow run and block until it finishes.
    #
    # Returns the result hash keyed by task readable_id, e.g.
    #   {"step1" => {"value" => 42}, "step2" => {...}}
    #
    # @param workflow_run_id [String]
    # @param timeout [Integer] Max seconds to wait (default 300)
    # @return [Hash] Task results keyed by task name
    # @raise [FailedRunError] if the run failed
    # @raise [DedupeViolationError] if a dedupe violation occurred
    # @raise [Timeout::Error] if the timeout expires
    def result(workflow_run_id, timeout: 300)
      event = subscribe(workflow_run_id, timeout: timeout)
      parse_event(event)
    end

    # Stop the listener and clean up resources.
    def shutdown
      @request_queue.close
      @reader_thread&.join(5)
    end

    private

    # Subscribe to a single workflow run and block until the event arrives.
    #
    # @param workflow_run_id [String]
    # @param timeout [Integer]
    # @return [WorkflowRunEvent]
    def subscribe(workflow_run_id, timeout: 300)
      subscriber_queue = Queue.new

      @mu.synchronize do
        @subscriptions[workflow_run_id] << subscriber_queue
      end

      ensure_reader_started!

      # Send the subscription request on the outgoing stream
      @request_queue << ::SubscribeToWorkflowRunsRequest.new(
        workflow_run_id: workflow_run_id,
      )

      # Block until we receive the event or timeout
      deadline = Time.now + timeout
      loop do
        remaining = deadline - Time.now
        raise Timeout::Error, "Timed out waiting for workflow run #{workflow_run_id}" if remaining <= 0

        # Use pop with timeout via a polling approach (Ruby Queue#pop doesn't
        # support timeout natively, so we use non_block + sleep)
        begin
          event = subscriber_queue.pop(true) # non-blocking
          return event
        rescue ThreadError
          # Queue is empty, sleep briefly and retry
          sleep(0.01)
        end
      end
    ensure
      # Clean up this subscription
      @mu.synchronize do
        if @subscriptions.key?(workflow_run_id)
          @subscriptions[workflow_run_id].delete(subscriber_queue)
          @subscriptions.delete(workflow_run_id) if @subscriptions[workflow_run_id].empty?
        end
      end
    end

    # Parse a WorkflowRunEvent into a result hash or raise on errors.
    #
    # @param event [WorkflowRunEvent]
    # @return [Hash] Task results keyed by task name
    def parse_event(event)
      errors = event.results.select { |r| r.respond_to?(:error) && r.error && !r.error.empty? }

      if errors.any?
        first_error = errors.first.error

        raise DedupeViolationError, first_error if first_error.include?(DEDUPE_MESSAGE)

        raise(FailedRunError, errors.map { |r| TaskRunError.new(r.error, task_run_external_id: r.task_run_external_id) })
      end

      # Build the result hash: { task_name => parsed_output }
      result = {}
      event.results.each do |step_result|
        next unless step_result.output && !step_result.output.empty?

        result[step_result.task_name] = JSON.parse(step_result.output)
      end
      result
    end

    # Ensure the background reader thread is running.
    def ensure_reader_started!
      @mu.synchronize do
        return if @started

        @started = true
        @reader_thread = Thread.new { reader_loop }
        @reader_thread.abort_on_exception = false
      end
    end

    # Background thread: maintains the bidi stream and routes events.
    def reader_loop
      retries = 0

      while retries < MAX_RETRIES
        begin
          retries += 1 if retries.positive?

          stub = ::Dispatcher::Stub.new(
            @config.host_port,
            nil,
            channel_override: @channel,
          )

          # Build the outgoing request enumerator.
          # On reconnect we re-subscribe all active workflow_run_ids first,
          # then continue pulling new requests from the queue.
          request_enum = build_request_enumerator

          # Open the bidi stream
          response_stream = stub.subscribe_to_workflow_runs(
            request_enum,
            metadata: @config.auth_metadata,
          )

          retries = 0 # connected successfully

          # Read events from the stream
          response_stream.each do |event|
            route_event(event)
          end

          # Stream ended normally (server closed). Reconnect.
          @logger.debug("WorkflowRunListener stream ended, reconnecting...")
        rescue ::GRPC::Unavailable => e
          @logger.warn("WorkflowRunListener gRPC unavailable: #{e.message}")
        rescue ::GRPC::Cancelled => e
          @logger.debug("WorkflowRunListener gRPC cancelled: #{e.message}")
          break # intentional shutdown
        rescue ::GRPC::Unknown => e
          @logger.warn("WorkflowRunListener gRPC unknown error: #{e.message}")
        rescue StopIteration
          break # queue closed
        rescue StandardError => e
          @logger.warn("WorkflowRunListener error: #{e.class}: #{e.message}")
        end

        if retries >= MAX_RETRIES
          @logger.error("WorkflowRunListener exhausted #{MAX_RETRIES} retries")
          break
        end

        sleep(RETRY_INTERVAL)
      end

      # Close all remaining subscriptions with an error
      @mu.synchronize do
        @started = false
      end
    end

    # Build an Enumerator that first re-subscribes existing workflow_run_ids,
    # then yields new requests from the queue.
    def build_request_enumerator
      Enumerator.new do |yielder|
        # Re-subscribe all currently active subscriptions
        active_ids = @mu.synchronize { @subscriptions.keys.dup }
        active_ids.each do |wfr_id|
          yielder << ::SubscribeToWorkflowRunsRequest.new(workflow_run_id: wfr_id)
        end

        # Then yield new requests as they arrive on the queue
        loop do
          request = @request_queue.pop # blocks until available; raises on close
          yielder << request
        end
      end
    end

    # Route an incoming WorkflowRunEvent to the correct subscriber queues.
    def route_event(event)
      wfr_id = event.workflow_run_id

      @mu.synchronize do
        queues = @subscriptions[wfr_id]
        return if queues.nil? || queues.empty?

        queues.each { |q| q << event }
      end
    end
  end
end
