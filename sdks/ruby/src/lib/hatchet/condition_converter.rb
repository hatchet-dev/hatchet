# frozen_string_literal: true

require "securerandom"

module Hatchet
  # Shared logic for converting condition objects into protobuf messages.
  #
  # Used by both Task (for workflow registration) and DurableContext (for
  # runtime durable event registration) to avoid duplicating the condition
  # processing logic.
  module ConditionConverter
    module_function

    # Convert a single condition into the appropriate protobuf message and
    # append it to the provided accumulator arrays.
    #
    # @param cond [Object] The condition (SleepCondition, UserEventCondition, Hash, or duck-typed)
    # @param action [Symbol] Proto action (:QUEUE or :SKIP)
    # @param sleep_conditions [Array] Accumulator for sleep condition protos
    # @param user_event_conditions [Array] Accumulator for user event condition protos
    # @param or_group_id [String, nil] Shared OR group ID (defaults to a new UUID)
    # @param readable_data_key [String, nil] Override for Hash readable_data_key (used by durable context)
    # @param proto_method [Symbol] Method to check for self-converting conditions (:to_proto or :to_durable_proto)
    # @param proto_arg [Object, nil] Argument to pass to proto_method (action for :to_proto, key for :to_durable_proto)
    def convert_condition(cond, action:, sleep_conditions:, user_event_conditions:,
                          or_group_id: nil, readable_data_key: nil,
                          proto_method: :to_proto, proto_arg: nil, config: nil)
      or_group_id ||= SecureRandom.uuid

      if cond.respond_to?(proto_method)
        # Condition object knows how to convert itself
        proto = cond.public_send(proto_method, proto_arg || action)
        case proto
        when ::V1::SleepMatchCondition
          sleep_conditions << proto
        when ::V1::UserEventMatchCondition
          user_event_conditions << proto
        end
      elsif cond.is_a?(Hatchet::SleepCondition)
        convert_sleep_condition(cond.duration, action: action, or_group_id: or_group_id,
                                               sleep_conditions: sleep_conditions,)
      elsif cond.is_a?(Hatchet::UserEventCondition)
        convert_user_event_condition(cond.event_key, action: action, or_group_id: or_group_id,
                                                     expression: cond.expression || "",
                                                     user_event_conditions: user_event_conditions,
                                                     config: config,)
      elsif cond.is_a?(Hash)
        convert_hash_condition(cond, action: action, or_group_id: or_group_id,
                                     readable_data_key: readable_data_key,
                                     sleep_conditions: sleep_conditions,
                                     user_event_conditions: user_event_conditions,
                                     config: config,)
      elsif cond.respond_to?(:event_key) && cond.event_key
        expression = cond.respond_to?(:expression) ? (cond.expression || "") : ""
        convert_user_event_condition(cond.event_key, action: action, or_group_id: or_group_id,
                                                     expression: expression,
                                                     user_event_conditions: user_event_conditions,
                                                     config: config,)
      elsif cond.respond_to?(:duration) && cond.duration
        convert_sleep_condition(cond.duration, action: action, or_group_id: or_group_id,
                                               sleep_conditions: sleep_conditions,)
      end
    end

    # @param duration [Integer, String] Sleep duration
    def convert_sleep_condition(duration, action:, or_group_id:, sleep_conditions:)
      base = ::V1::BaseMatchCondition.new(
        readable_data_key: "sleep_#{duration}",
        action: action,
        or_group_id: or_group_id,
      )
      sleep_conditions << ::V1::SleepMatchCondition.new(
        base: base,
        sleep_for: "#{duration}s",
      )
    end

    # @param event_key [String] Event key
    # @param config [Hatchet::Config, nil] Config for namespace resolution
    def convert_user_event_condition(event_key, action:, or_group_id:, expression:, user_event_conditions:, config: nil)
      namespaced_key = config ? config.apply_namespace(event_key) : event_key

      base = ::V1::BaseMatchCondition.new(
        readable_data_key: namespaced_key,
        action: action,
        or_group_id: or_group_id,
        expression: expression,
      )
      user_event_conditions << ::V1::UserEventMatchCondition.new(
        base: base,
        user_event_key: namespaced_key,
      )
    end

    # @param cond [Hash] Hash-based condition
    # @param config [Hatchet::Config, nil] Config for namespace resolution
    def convert_hash_condition(cond, action:, or_group_id:, readable_data_key:,
                               sleep_conditions:, user_event_conditions:, config: nil)
      base_key = readable_data_key || cond[:readable_data_key] || cond[:key] || ""

      base = ::V1::BaseMatchCondition.new(
        readable_data_key: base_key,
        action: action,
        or_group_id: cond[:or_group_id] || or_group_id,
        expression: cond[:expression] || "",
      )

      if cond[:sleep_for]
        sleep_conditions << ::V1::SleepMatchCondition.new(
          base: base,
          sleep_for: cond[:sleep_for].to_s,
        )
      elsif cond[:event_key]
        namespaced_key = config ? config.apply_namespace(cond[:event_key]) : cond[:event_key]
        user_event_conditions << ::V1::UserEventMatchCondition.new(
          base: base,
          user_event_key: namespaced_key,
        )
      end
    end
  end
end
