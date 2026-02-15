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
    # Sleep for a specified duration. The task is suspended and resumed
    # by the engine after the duration expires.
    #
    # @param duration [Integer, String] Duration in seconds, or a duration string (e.g. "60s")
    # @return [Hash, nil] Result from the sleep event
    def sleep_for(duration:)
      signal_key = "sleep_#{duration}"

      duration_str = duration.is_a?(String) ? duration : "#{duration}s"

      # Build the sleep condition
      sleep_condition = ::V1::SleepMatchCondition.new(
        base: ::V1::BaseMatchCondition.new(
          readable_data_key: signal_key,
          action: :QUEUE,
          or_group_id: SecureRandom.uuid,
        ),
        sleep_for: duration_str,
      )

      conditions = ::V1::DurableEventListenerConditions.new(
        sleep_conditions: [sleep_condition],
      )

      # Register the durable event
      register_request = ::V1::RegisterDurableEventRequest.new(
        task_id: @step_run_id,
        signal_key: signal_key,
        conditions: conditions,
      )

      v1_dispatcher_stub.register_durable_event(register_request, metadata: @client.config.auth_metadata)

      # Listen for the durable event via bidi stream
      listen_for_event(signal_key)
    end

    # Wait for a condition to be met (event or sleep).
    # The task is suspended and resumed when the condition is satisfied.
    #
    # @param key [String] A unique key for this wait operation
    # @param condition [Object] The condition to wait for (UserEventCondition, SleepCondition, Hash, etc.)
    # @return [Hash] Result from the wait, including which condition was satisfied
    def wait_for(key, condition)
      conditions = build_durable_conditions(key, condition)

      # Register the durable event
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

    private

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
        action: :QUEUE,
        sleep_conditions: sleep_conditions,
        user_event_conditions: user_event_conditions,
        or_group_id: or_group_id,
        readable_data_key: key,
        proto_method: :to_durable_proto,
        proto_arg: key,
      )
    end
  end
end
