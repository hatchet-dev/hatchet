# frozen_string_literal: true

require "json"
require "securerandom"

module Hatchet
  # Extended context for durable tasks that supports sleep and event-waiting
  # across task suspensions.
  #
  # Durable tasks can be suspended and resumed by the Hatchet engine,
  # allowing long-running workflows that survive process restarts.
  #
  # Uses V1::V1Dispatcher for registering and listening for durable events
  # via bidirectional gRPC streaming.
  #
  # @example Sleep for a duration
  #   hatchet.durable_task(name: "my_task") do |input, ctx|
  #     ctx.sleep_for(duration: 60) # sleep for 60 seconds
  #   end
  #
  # @example Wait for an event
  #   hatchet.durable_task(name: "my_task") do |input, ctx|
  #     result = ctx.wait_for("event", Hatchet::UserEventCondition.new(event_key: "user:update"))
  #   end
  class DurableContext < Context
    # @return [Hatchet::WorkerRuntime::DurableEviction::DurableEvictionManager, nil]
    attr_accessor :eviction_manager

    # @return [String, nil] The action key used by the eviction manager to
    #   identify this run invocation.
    attr_accessor :action_key

    # @return [Hatchet::WorkerRuntime::DurableEventListener, nil] New-style bidi
    #   listener. When set the context delegates through it instead of the
    #   legacy RegisterDurableEvent/ListenForDurableEvent path.
    attr_accessor :durable_event_listener

    # @return [Integer] Durable-task invocation count (>= 1).
    attr_accessor :invocation_count

    # @return [String, nil] Engine version string advertised via GetVersion.
    attr_accessor :engine_version

    # Sleep for a specified duration. The task is suspended and resumed
    # by the engine after the duration expires.
    #
    # Delegates to {#wait_for} with a {Hatchet::SleepCondition} so that both
    # sleeps and event waits share a single registration / eviction path.
    #
    # @param duration [Integer, String] Duration in seconds, or a duration string (e.g. "60s")
    # @param label [String, nil] Optional wait label shown in durable event logs.
    # @return [Hash, nil] Result from the sleep event
    def sleep_for(duration:, label: nil)
      duration_str = duration.is_a?(String) ? duration : "#{duration}s"
      duration_value = duration.is_a?(String) ? duration : duration.to_i
      wait_index = increment_wait_index
      signal_key = "sleep:#{duration_str}-#{wait_index}"

      wait_for(signal_key, Hatchet::SleepCondition.new(duration_value), label: label)
    end

    # Wait for a condition to be met (event or sleep).
    # The task is suspended and resumed when the condition is satisfied.
    #
    # Register the durable wait with ``send_event`` first, then start eviction
    # tracking only while blocked on ``wait_for_callback``.
    #
    # @param key [String] A unique key for this wait operation
    # @param condition [Object] The condition to wait for (UserEventCondition, SleepCondition, Hash, etc.)
    # @param label [String, nil] Optional wait label shown in durable event logs.
    # @return [Hash] Result from the wait, including which condition was satisfied
    def wait_for(key, condition, label: nil)
      conditions = build_durable_conditions(key, condition)

      if supports_durable_eviction?
        invocation = @invocation_count || 1

        event = Hatchet::WorkerRuntime::DurableEventListener::WaitForEvent.new(
          wait_for_conditions: conditions,
          label: label,
        )

        ack = @durable_event_listener.send_event(@step_run_id, invocation, event)

        with_eviction_wait(wait_kind: "wait_for", resource_id: key) do
          result = @durable_event_listener.wait_for_callback(
            @step_run_id,
            invocation,
            ack[:branch_id],
            ack[:node_id],
          )

          result[:payload] || {}
        end
      else
        with_eviction_wait(wait_kind: "wait_for", resource_id: key) do
          legacy_wait_for(key, conditions)
        end
      end
    end

    private

    # Monotonically increasing per-context wait index, used to disambiguate
    # multiple sleep/wait calls with otherwise identical resource ids.
    def increment_wait_index
      @wait_index ||= 0
      index = @wait_index
      @wait_index += 1
      index
    end

    # Wrap a block in mark_waiting/mark_active calls on the eviction manager
    # when one is attached. Safe no-op when not.
    def with_eviction_wait(wait_kind:, resource_id:)
      mgr = @eviction_manager
      key = @action_key

      mgr.mark_waiting(key, wait_kind: wait_kind, resource_id: resource_id) if mgr && key

      begin
        yield
      ensure
        mgr.mark_active(key) if mgr && key
      end
    end

    # True when this context can use the durable-eviction bidi protocol.
    #
    # Matches the Python SDK's feature-gate naming more closely than
    # ``use_new_listener?`` while preserving the same behavior here.
    def supports_durable_eviction?
      return false unless @durable_event_listener
      return false unless @engine_version

      !Hatchet::EngineVersion.semver_less_than?(
        @engine_version,
        Hatchet::MinEngineVersion::DURABLE_EVICTION,
      )
    end

    def legacy_wait_for(key, conditions)
      register_request = ::V1::RegisterDurableEventRequest.new(
        task_id: @step_run_id,
        signal_key: key,
        conditions: conditions,
      )

      v1_dispatcher_stub.register_durable_event(register_request, metadata: @client.config.auth_metadata)

      # Listen for the durable event via bidi stream
      result = listen_for_event(key)

      # Parse the result data
      if result.respond_to?(:data) && !result.data.to_s.empty?
        begin
          JSON.parse(result.data)
        rescue JSON::ParserError
          { "data" => result.data.to_s }
        end
      else
        {}
      end
    end

    # Get or create the V1::V1Dispatcher::Stub for durable events.
    def v1_dispatcher_stub
      @v1_dispatcher_stub ||= ::V1::V1Dispatcher::Stub.new(
        @client.config.host_port,
        nil,
        channel_override: @client.channel,
      )
    end

    # Listen for a durable event using bidirectional streaming.
    #
    # In Ruby's grpc gem, bidi streams use an Enumerator for requests
    # and return an Enumerator for responses.
    #
    # @param signal_key [String] The signal key to listen for
    # @return [V1::DurableEvent, nil] The received durable event
    def listen_for_event(signal_key)
      # Create a request enumerator for the bidi stream
      request_queue = Queue.new
      request_enum = Enumerator.new do |yielder|
        # Send initial request
        yielder << ::V1::ListenForDurableEventRequest.new(
          task_id: @step_run_id,
          signal_key: signal_key,
        )

        # Keep the stream alive until we get a response
        loop do
          msg = request_queue.pop
          break if msg == :done

          yielder << msg
        end
      end

      # Start the bidi stream
      response_stream = v1_dispatcher_stub.listen_for_durable_event(
        request_enum,
        metadata: @client.config.auth_metadata,
      )

      # Wait for the first matching response
      result = nil
      response_stream.each do |event|
        if event.signal_key == signal_key
          result = event
          break
        end
      end

      # Signal the request stream to close
      request_queue << :done

      result
    rescue StandardError => e
      begin
        request_queue << :done
      rescue StandardError
        nil
      end
      raise e
    end

    # Build DurableEventListenerConditions from a condition object.
    #
    # @param key [String] The signal key
    # @param condition [Object] The condition (UserEventCondition, SleepCondition, OrCondition, Hash, etc.)
    # @return [V1::DurableEventListenerConditions]
    def build_durable_conditions(key, condition)
      sleep_conditions = []
      user_event_conditions = []

      if condition.is_a?(Hatchet::OrCondition)
        # All conditions in an OR group share the same or_group_id
        or_group_id = SecureRandom.uuid
        condition.conditions.each do |cond|
          process_durable_condition(key, cond, or_group_id, sleep_conditions, user_event_conditions)
        end
      else
        process_durable_condition(key, condition, SecureRandom.uuid, sleep_conditions, user_event_conditions)
      end

      ::V1::DurableEventListenerConditions.new(
        sleep_conditions: sleep_conditions,
        user_event_conditions: user_event_conditions,
      )
    end

    # Process a single condition into the appropriate proto lists.
    # Delegates to ConditionConverter for shared logic.
    #
    # @param key [String] The signal key
    # @param condition [Object] The condition to process
    # @param or_group_id [String] The OR group ID for this condition
    # @param sleep_conditions [Array] Accumulator for sleep conditions
    # @param user_event_conditions [Array] Accumulator for user event conditions
    def process_durable_condition(key, condition, or_group_id, sleep_conditions, user_event_conditions)
      ConditionConverter.convert_condition(
        condition,
        # Do not force base.action. Leaving it unset keeps protobuf default semantics on the server path.
        action: nil,
        sleep_conditions: sleep_conditions,
        user_event_conditions: user_event_conditions,
        or_group_id: or_group_id,
        readable_data_key: key,
        proto_method: :to_durable_proto,
        proto_arg: key,
        config: @client&.config,
      )
    end
  end
end
