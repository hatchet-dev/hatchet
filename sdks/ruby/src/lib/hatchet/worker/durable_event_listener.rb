# frozen_string_literal: true

require "json"
require "monitor"
require "timeout"

module Hatchet
  module WorkerRuntime
    # Thread-safe multiplexer over the ``V1Dispatcher.DurableTask`` bidirectional
    # gRPC stream.
    #
    # A single stream is shared across all durable task invocations running on
    # the worker; callers send ``send_event`` / ``wait_for_callback``
    # / ``send_evict_invocation`` requests and block on per-call Queues until the
    # response-dispatch thread routes the matching ``DurableTaskResponse`` back.
    #
    # @example
    #   listener = DurableEventListener.new(config: config, channel: channel, logger: logger)
    #   listener.start("worker-id-123")
    #   ack = listener.send_event(task_id, invocation_count, wait_for_event)
    #   result = listener.wait_for_callback(task_id, invocation_count, branch_id, node_id)
    class DurableEventListener
      DEFAULT_RECONNECT_INTERVAL = 3 # seconds
      EVICTION_ACK_TIMEOUT_SECONDS = 30.0
      REGISTER_WORKER_ACK_TIMEOUT_SECONDS = 10.0

      # Outgoing event sent via ``send_event``.
      #
      # @!attribute [r] wait_for_conditions
      #   @return [V1::DurableEventListenerConditions]
      # @!attribute [r] label
      #   @return [String, nil]
      WaitForEvent = Struct.new(:wait_for_conditions, :label, keyword_init: true)

      # Memo event with a ``bytes`` key and an optional already-computed result.
      MemoEvent = Struct.new(:memo_key, :result, keyword_init: true)

      # @return [String, nil]
      attr_reader :worker_id

      # @param config [Hatchet::Config]
      # @param channel [GRPC::Core::Channel]
      # @param logger [Logger]
      # @param on_server_evict [Proc, nil] Called with (durable_task_external_id, invocation_count)
      #   when the server notifies about a stale invocation.
      def initialize(config:, channel:, logger:, on_server_evict: nil)
        @config = config
        @channel = channel
        @logger = logger
        @on_server_evict = on_server_evict

        @worker_id = nil
        @stub = nil
        @request_queue = nil

        @mu = Monitor.new

        # (task_external_id, invocation_count) => Queue (push [:ok, ack] or [:err, exc])
        @pending_event_acks = {}
        # (task_external_id, invocation_count) => Queue (push [:ok, nil] or [:err, exc])
        @pending_eviction_acks = {}
        # (task_external_id, invocation_count, branch_id, node_id) => Queue
        @pending_callbacks = {}
        # key -> [inserted_at, result] (rudimentary TTL cache)
        @buffered_completions = {}

        @running = false
        @start_mu = Mutex.new
        @registration_mu = Mutex.new
        @registration_cv = ConditionVariable.new
        @worker_registered = false

        @receive_thread = nil
        @send_thread = nil
      end

      # Start the listener if not already running. Idempotent.
      #
      # @param worker_id [String]
      def start(worker_id)
        @start_mu.synchronize do
          return if @running

          @worker_id = worker_id
          @running = true
          @registration_mu.synchronize { @worker_registered = false }

          connect

          @receive_thread = Thread.new { receive_loop }
          @send_thread = Thread.new { send_loop }
          wait_for_register_worker_ack
        end
      end

      # Start the listener if not already running.
      def ensure_started(worker_id)
        start(worker_id) unless @running
      end

      # Stop the listener and release resources.
      def stop
        @running = false

        fail_all_pending(Hatchet::Error.new("DurableListener stopped"))

        @request_queue&.close
        rescue_thread(@receive_thread)
        rescue_thread(@send_thread)
      end

      # Send a ``DurableTask`` message and block for its ack.
      #
      # @param durable_task_external_id [String]
      # @param invocation_count [Integer]
      # @param event [WaitForEvent, MemoEvent] The event to send
      # @return [Object] The parsed ack body (a simple Hash describing the ack)
      # @raise [Hatchet::Error] on server-reported errors or listener disconnection
      def send_event(durable_task_external_id, invocation_count, event)
        raise Hatchet::Error, "DurableEventListener not started" unless @request_queue

        key = [durable_task_external_id, invocation_count]
        queue = Queue.new

        @mu.synchronize { @pending_event_acks[key] = queue }

        request = build_event_request(durable_task_external_id, invocation_count, event)
        @logger&.debug(
          "durable event listener send_event: task=#{durable_task_external_id} " \
          "invocation=#{invocation_count} event=#{event.class}",
        )
        @request_queue << request

        ack = await_queue(queue)
        @logger&.debug(
          "durable event listener send_event ack: task=#{durable_task_external_id} " \
          "invocation=#{invocation_count} ack_type=#{ack[:ack_type]} " \
          "branch_id=#{ack[:branch_id]} node_id=#{ack[:node_id]}",
        )
        ack
      end

      # Block until the server delivers an ``entry_completed`` (or error) for
      # this durable task / invocation / branch / node id tuple.
      #
      # @return [Hash] ``{ durable_task_external_id:, node_id:, payload: }``
      def wait_for_callback(durable_task_external_id, invocation_count, branch_id, node_id)
        key = [durable_task_external_id, invocation_count, branch_id, node_id]

        buffered = @mu.synchronize { @buffered_completions.delete(key) }
        if buffered
          @logger&.debug(
            "durable event listener wait_for_callback: buffered completion hit " \
            "task=#{durable_task_external_id} invocation=#{invocation_count} " \
            "branch_id=#{branch_id} node_id=#{node_id}",
          )
          return buffered[1]
        end

        queue = @mu.synchronize do
          @pending_callbacks[key] ||= Queue.new
        end

        @logger&.debug(
          "durable event listener wait_for_callback: waiting " \
          "task=#{durable_task_external_id} invocation=#{invocation_count} " \
          "branch_id=#{branch_id} node_id=#{node_id}",
        )
        poll_worker_status

        result = await_queue(queue)
        @logger&.debug(
          "durable event listener wait_for_callback: completed " \
          "task=#{durable_task_external_id} invocation=#{invocation_count} " \
          "branch_id=#{branch_id} node_id=#{node_id}",
        )
        result
      end

      # Request eviction of a stale invocation from the server and block until ack.
      #
      # @param durable_task_external_id [String]
      # @param invocation_count [Integer]
      # @param reason [String, nil] Optional human-readable reason.
      # @raise [Hatchet::Error] on timeout or listener disconnection
      def send_evict_invocation(durable_task_external_id, invocation_count, reason: nil)
        raise Hatchet::Error, "DurableEventListener not started" unless @request_queue

        key = [durable_task_external_id, invocation_count]
        queue = Queue.new
        @mu.synchronize { @pending_eviction_acks[key] = queue }

        args = {
          durable_task_external_id: durable_task_external_id,
          invocation_count: invocation_count,
        }
        args[:reason] = reason if reason
        req = ::V1::DurableTaskEvictInvocationRequest.new(**args)

        @logger&.debug(
          "durable event listener send_evict_invocation: task=#{durable_task_external_id} " \
          "invocation=#{invocation_count} reason=#{reason}",
        )
        @request_queue << ::V1::DurableTaskRequest.new(evict_invocation: req)

        await_queue(queue, timeout: EVICTION_ACK_TIMEOUT_SECONDS)
        @logger&.debug(
          "durable event listener send_evict_invocation ack: task=#{durable_task_external_id} " \
          "invocation=#{invocation_count}",
        )
      rescue Timeout::Error
        @mu.synchronize { @pending_eviction_acks.delete(key) }
        raise Hatchet::Error,
              "Eviction ack timed out after #{EVICTION_ACK_TIMEOUT_SECONDS.to_i}s " \
              "for task #{durable_task_external_id} invocation #{invocation_count}"
      end

      # Fire-and-forget ``complete_memo`` notification.
      def send_memo_completed_notification(durable_task_external_id:, node_id:, branch_id:, invocation_count:, memo_key:,
                                           memo_result_payload:)
        raise Hatchet::Error, "DurableEventListener not started" unless @request_queue

        ref = ::V1::DurableEventLogEntryRef.new(
          durable_task_external_id: durable_task_external_id,
          node_id: node_id,
          invocation_count: invocation_count,
          branch_id: branch_id,
        )
        complete = ::V1::DurableTaskCompleteMemoRequest.new(
          ref: ref,
          memo_key: memo_key,
          payload: memo_result_payload,
        )
        @request_queue << ::V1::DurableTaskRequest.new(complete_memo: complete)
      end

      # Drop pending callbacks / acks / buffered completions whose invocation
      # count is ``<= invocation_count`` for the given task id.
      def cleanup_task_state(durable_task_external_id, invocation_count)
        @mu.synchronize do
          @pending_callbacks.each_key do |k|
            next unless k[0] == durable_task_external_id && k[1] <= invocation_count

            @pending_callbacks.delete(k)&.close
          end

          @pending_event_acks.each_key do |k|
            next unless k[0] == durable_task_external_id && k[1] <= invocation_count

            @pending_event_acks.delete(k)&.close
          end

          @buffered_completions.each_key do |k|
            next unless k[0] == durable_task_external_id && k[1] <= invocation_count

            @buffered_completions.delete(k)
          end
        end
      end

      # Hook for tests: handle a single response message (bypassing the network).
      def handle_response_for_test(response)
        handle_response(response)
      end

      private

      def build_event_request(durable_task_external_id, invocation_count, event)
        case event
        when WaitForEvent
          if event.wait_for_conditions
            sleep_conditions = event.wait_for_conditions.sleep_conditions || []
            user_event_conditions = event.wait_for_conditions.user_event_conditions || []
            first_sleep = sleep_conditions.first
            if first_sleep&.base
              @logger&.debug(
                "durable event listener wait_for payload: task=#{durable_task_external_id} " \
                "invocation=#{invocation_count} sleep_count=#{sleep_conditions.length} " \
                "event_count=#{user_event_conditions.length} " \
                "first_sleep_readable_key=#{first_sleep.base.readable_data_key} " \
                "first_sleep_for=#{first_sleep.sleep_for} " \
                "first_sleep_action=#{first_sleep.base.action} " \
                "first_sleep_or_group_id=#{first_sleep.base.or_group_id}",
              )
            else
              @logger&.debug(
                "durable event listener wait_for payload: task=#{durable_task_external_id} " \
                "invocation=#{invocation_count} sleep_count=#{sleep_conditions.length} " \
                "event_count=#{user_event_conditions.length}",
              )
            end
          else
            @logger&.debug(
              "durable event listener wait_for payload: task=#{durable_task_external_id} " \
              "invocation=#{invocation_count} wait_for_conditions=nil",
            )
          end

          wait_req = ::V1::DurableTaskWaitForRequest.new(
            durable_task_external_id: durable_task_external_id,
            invocation_count: invocation_count,
            wait_for_conditions: event.wait_for_conditions,
            label: event.label,
          )
          ::V1::DurableTaskRequest.new(wait_for: wait_req)
        when MemoEvent
          memo_req = ::V1::DurableTaskMemoRequest.new(
            durable_task_external_id: durable_task_external_id,
            invocation_count: invocation_count,
            key: event.memo_key,
          )
          memo_req.payload = event.result.to_s if event.result
          ::V1::DurableTaskRequest.new(memo: memo_req)
        else
          raise ArgumentError, "Unknown durable task send event: #{event.class}"
        end
      end

      def await_queue(queue, timeout: nil)
        msg = if timeout
                deadline = Time.now + timeout
                loop do
                  break queue.pop(true)
                rescue ThreadError
                  raise Timeout::Error, "timed out waiting for queue" if Time.now >= deadline

                  sleep 0.05
                end
              else
                queue.pop
              end

        raise Hatchet::Error, "listener closed" if msg.nil?

        kind, payload = msg
        raise payload if kind == :err

        payload
      end

      def connect
        @request_queue = Queue.new

        stub = ::V1::V1Dispatcher::Stub.new(
          @config.host_port,
          nil,
          channel_override: @channel,
        )
        @stub = stub

        @request_enum = build_request_enumerator

        @logger&.info("durable event listener connecting...")

        @stream = stub.durable_task(@request_enum, metadata: @config.auth_metadata)

        register_worker
        poll_worker_status

        @logger&.info("durable event listener connected")
      end

      def mark_stream_unavailable(error)
        old_queue = @request_queue
        @request_queue = nil
        @stream = nil

        begin
          old_queue&.close
        rescue StandardError
          nil
        end

        fail_pending_acks(error)
      end

      def wait_for_register_worker_ack
        timeout_at = Time.now + REGISTER_WORKER_ACK_TIMEOUT_SECONDS
        @registration_mu.synchronize do
          until @worker_registered
            remaining = timeout_at - Time.now
            break if remaining <= 0

            @registration_cv.wait(@registration_mu, remaining)
          end
        end

        return if @registration_mu.synchronize { @worker_registered }

        raise Hatchet::Error,
              "durable event listener did not receive register_worker ack " \
              "within #{REGISTER_WORKER_ACK_TIMEOUT_SECONDS.to_i}s"
      end

      def build_request_enumerator
        queue = @request_queue
        Enumerator.new do |yielder|
          loop do
            begin
              req = queue.pop
            rescue ClosedQueueError
              break
            end

            break if req.nil?

            request_kind =
              if req.respond_to?(:register_worker) && req.register_worker
                "register_worker"
              elsif req.respond_to?(:wait_for) && req.wait_for
                "wait_for"
              elsif req.respond_to?(:memo) && req.memo
                "memo"
              elsif req.respond_to?(:trigger_runs) && req.trigger_runs
                "trigger_runs"
              elsif req.respond_to?(:evict_invocation) && req.evict_invocation
                "evict_invocation"
              elsif req.respond_to?(:worker_status) && req.worker_status
                "worker_status"
              elsif req.respond_to?(:complete_memo) && req.complete_memo
                "complete_memo"
              else
                "unknown"
              end
            @logger&.debug("durable event listener stream write: kind=#{request_kind}")
            yielder << req
          end
        end
      end

      def register_worker
        raise Hatchet::Error, "Client not started" if @worker_id.nil?

        @request_queue << ::V1::DurableTaskRequest.new(
          register_worker: ::V1::DurableTaskRequestRegisterWorker.new(worker_id: @worker_id),
        )
      end

      def poll_worker_status
        return if @request_queue.nil? || @worker_id.nil?

        pending = @mu.synchronize { @pending_callbacks.keys.dup }
        return if pending.empty?

        waiting = pending.map do |(task_ext_id, inv_count, branch_id, node_id)|
          ::V1::DurableTaskAwaitedCompletedEntry.new(
            durable_task_external_id: task_ext_id,
            invocation_count: inv_count,
            node_id: node_id,
            branch_id: branch_id,
          )
        end

        @request_queue << ::V1::DurableTaskRequest.new(
          worker_status: ::V1::DurableTaskWorkerStatusRequest.new(
            worker_id: @worker_id,
            waiting_entries: waiting,
          ),
        )
      end

      def send_loop
        while @running
          sleep 1
          begin
            poll_worker_status
          rescue StandardError => e
            @logger&.error("durable event listener send_loop error: #{e.class}: #{e.message}")
          end
        end
      end

      def receive_loop
        while @running
          unless @stream
            sleep DEFAULT_RECONNECT_INTERVAL
            next
          end

          begin
            @stream.each { |response| handle_response(response) }

            if @running
              @logger&.warn(
                "durable event listener disconnected (EOF), reconnecting in #{DEFAULT_RECONNECT_INTERVAL}s...",
              )
              mark_stream_unavailable(Hatchet::Error.new("durable stream disconnected"))
              sleep DEFAULT_RECONNECT_INTERVAL
              safe_reconnect
            end
          rescue ::GRPC::Cancelled
            break
          rescue ::GRPC::BadStatus => e
            @logger&.warn(
              "durable event listener disconnected: code=#{e.code}, " \
              "details=#{e.details}, reconnecting in #{DEFAULT_RECONNECT_INTERVAL}s...",
            )
            if @running
              mark_stream_unavailable(Hatchet::Error.new("durable stream error: #{e.code} #{e.details}"))
              sleep DEFAULT_RECONNECT_INTERVAL
              safe_reconnect
            end
          rescue StandardError => e
            @logger&.error("unexpected error in durable event listener: #{e.class}: #{e.message}")
            if @running
              mark_stream_unavailable(e)
              sleep DEFAULT_RECONNECT_INTERVAL
              safe_reconnect
            end
          end
        end
      end

      def safe_reconnect
        connect
      rescue StandardError => e
        @logger&.error("failed to reconnect durable event listener: #{e.class}: #{e.message}")
      end

      def handle_response(response)
        @logger&.debug("durable event listener stream read: kind=#{response_kind(response)}")

        return handle_register_worker if response.has_register_worker?
        return handle_trigger_runs_ack(response.trigger_runs_ack) if response.has_trigger_runs_ack?
        return handle_memo_ack(response.memo_ack) if response.has_memo_ack?
        return handle_wait_for_ack(response.wait_for_ack) if response.has_wait_for_ack?
        return handle_entry_completed(response.entry_completed) if response.has_entry_completed?
        return handle_eviction_ack(response.eviction_ack) if response.has_eviction_ack?
        return handle_server_evict(response.server_evict) if response.has_server_evict?

        handle_error_response(response.error) if response.has_error?
      end

      def response_kind(response)
        return "register_worker" if response.has_register_worker?
        return "trigger_runs_ack" if response.has_trigger_runs_ack?
        return "memo_ack" if response.has_memo_ack?
        return "wait_for_ack" if response.has_wait_for_ack?
        return "entry_completed" if response.has_entry_completed?
        return "eviction_ack" if response.has_eviction_ack?
        return "server_evict" if response.has_server_evict?
        return "error" if response.has_error?

        "unknown"
      end

      def handle_register_worker
        @registration_mu.synchronize do
          @worker_registered = true
          @registration_cv.broadcast
        end
      end

      def handle_trigger_runs_ack(ack)
        deliver_event_ack(
          [ack.durable_task_external_id, ack.invocation_count],
          {
            ack_type: :run,
            invocation_count: ack.invocation_count,
            durable_task_external_id: ack.durable_task_external_id,
            run_entries: ack.run_entries.map do |entry|
              {
                node_id: entry.node_id,
                branch_id: entry.branch_id,
                workflow_run_external_id: entry.workflow_run_external_id,
              }
            end,
          },
        )
      end

      def handle_memo_ack(ack)
        deliver_event_ack(
          [ack.ref.durable_task_external_id, ack.ref.invocation_count],
          {
            ack_type: :memo,
            invocation_count: ack.ref.invocation_count,
            durable_task_external_id: ack.ref.durable_task_external_id,
            node_id: ack.ref.node_id,
            branch_id: ack.ref.branch_id,
            memo_already_existed: ack.memo_already_existed,
            memo_result_payload: ack.memo_result_payload,
          },
        )
      end

      def handle_wait_for_ack(ack)
        @logger&.debug(
          "durable event listener recv wait_for_ack: task=#{ack.ref.durable_task_external_id} " \
          "invocation=#{ack.ref.invocation_count} branch_id=#{ack.ref.branch_id} node_id=#{ack.ref.node_id}",
        )
        deliver_event_ack(
          [ack.ref.durable_task_external_id, ack.ref.invocation_count],
          {
            ack_type: :wait,
            invocation_count: ack.ref.invocation_count,
            durable_task_external_id: ack.ref.durable_task_external_id,
            node_id: ack.ref.node_id,
            branch_id: ack.ref.branch_id,
          },
        )
      end

      def handle_entry_completed(completed)
        @logger&.debug(
          "durable event listener recv entry_completed: task=#{completed.ref.durable_task_external_id} " \
          "invocation=#{completed.ref.invocation_count} branch_id=#{completed.ref.branch_id} node_id=#{completed.ref.node_id}",
        )
        key = callback_key_for(completed.ref)
        result = parse_entry_completed(completed)

        @mu.synchronize do
          queue = @pending_callbacks.delete(key)
          if queue
            queue << [:ok, result]
          else
            @buffered_completions[key] = [Time.now, result]
          end
        end
      end

      def handle_eviction_ack(ack)
        key = [ack.durable_task_external_id, ack.invocation_count]

        @mu.synchronize do
          queue = @pending_eviction_acks.delete(key)
          queue&.<<([:ok, nil])
        end
      end

      def handle_server_evict(evict)
        @logger&.info(
          "received server eviction notification for task #{evict.durable_task_external_id} " \
          "invocation #{evict.invocation_count}: #{evict.reason}",
        )
        cleanup_task_state(evict.durable_task_external_id, evict.invocation_count)
        @on_server_evict&.call(evict.durable_task_external_id, evict.invocation_count)
      end

      def callback_key_for(ref)
        [
          ref.durable_task_external_id,
          ref.invocation_count,
          ref.branch_id,
          ref.node_id,
        ]
      end

      def handle_error_response(error)
        exc = if error.error_type == :DURABLE_TASK_ERROR_TYPE_NONDETERMINISM
                Hatchet::NonDeterminismError.new(
                  error.error_message,
                  task_external_id: error.ref.durable_task_external_id,
                  invocation_count: error.ref.invocation_count,
                  node_id: error.ref.node_id,
                )
              else
                Hatchet::Error.new(
                  "Unspecified durable task error: #{error.error_message} (type: #{error.error_type})",
                )
              end

        event_key = [error.ref.durable_task_external_id, error.ref.invocation_count]
        callback_key = [
          error.ref.durable_task_external_id,
          error.ref.invocation_count,
          error.ref.branch_id,
          error.ref.node_id,
        ]

        @mu.synchronize do
          queue = @pending_event_acks.delete(event_key)
          queue&.<<([:err, exc])

          queue = @pending_callbacks.delete(callback_key)
          queue&.<<([:err, exc])

          queue = @pending_eviction_acks.delete(event_key)
          queue&.<<([:err, exc])
        end
      end

      def deliver_event_ack(key, payload)
        @mu.synchronize do
          queue = @pending_event_acks.delete(key)
          queue&.<<([:ok, payload])
        end
      end

      def parse_entry_completed(completed)
        payload = nil
        if completed.payload && !completed.payload.empty?
          begin
            payload_json = completed.payload.dup.force_encoding("UTF-8")
            payload = JSON.parse(payload_json)
          rescue JSON::ParserError
            payload = nil
          end
        end

        {
          durable_task_external_id: completed.ref.durable_task_external_id,
          node_id: completed.ref.node_id,
          payload: payload,
        }
      end

      def fail_pending_acks(exc)
        @mu.synchronize do
          @pending_event_acks.each_value { |q| q << [:err, exc] }
          @pending_event_acks.clear
          @pending_eviction_acks.each_value { |q| q << [:err, exc] }
          @pending_eviction_acks.clear
        end
      end

      def fail_all_pending(exc)
        fail_pending_acks(exc)
        @mu.synchronize do
          @pending_callbacks.each_value { |q| q << [:err, exc] }
          @pending_callbacks.clear
          @buffered_completions.clear
        end
      end

      def rescue_thread(thread)
        return unless thread

        thread.join(5)
      rescue StandardError
        nil
      end
    end
  end
end
