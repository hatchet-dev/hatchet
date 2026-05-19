# frozen_string_literal: true

module Hatchet
  # A sleep condition that causes a task to wait for a specified duration
  #
  # @example Wait for 10 seconds
  #   Hatchet::SleepCondition.new(10)
  class SleepCondition
    # @return [Integer] Duration in seconds to sleep
    attr_reader :duration

    # @param duration [Integer] Duration in seconds
    def initialize(duration)
      @duration = duration
    end

    # @return [Hash]
    def to_h
      { type: "sleep", duration: "#{@duration}s" }
    end
  end

  # A condition that waits for a user event with a specific key
  #
  # @example Wait for a user:update event
  #   Hatchet::UserEventCondition.new(event_key: "user:update")
  #
  # @example With filter expression
  #   Hatchet::UserEventCondition.new(event_key: "user:update", expression: "input.user_id == '1234'")
  class UserEventCondition
    # @return [String] The event key to listen for
    attr_reader :event_key

    # @return [String, nil] Optional CEL expression to filter events
    attr_reader :expression

    # @param event_key [String] The event key to wait for
    # @param expression [String, nil] Optional CEL filter expression
    def initialize(event_key:, expression: nil)
      @event_key = event_key
      @expression = expression
    end

    # @return [Hash]
    def to_h
      h = { type: "user_event", event_key: @event_key }
      h[:expression] = @expression if @expression
      h
    end
  end

  # A condition based on parent task output
  #
  # @example Skip if parent output meets condition
  #   Hatchet::ParentCondition.new(parent: start_task, expression: "output.random_number > 50")
  class ParentCondition
    # @return [Hatchet::Task, Symbol, String] Reference to the parent task
    attr_reader :parent

    # @return [String] CEL expression evaluated against the parent's output
    attr_reader :expression

    # @param parent [Hatchet::Task, Symbol, String] The parent task reference
    # @param expression [String] CEL expression to evaluate
    def initialize(parent:, expression:)
      @parent = parent
      @expression = expression
    end

    # @return [Hash]
    def to_h
      parent_name = case @parent
                    when Symbol then @parent.to_s
                    when String then @parent
                    else @parent.respond_to?(:name) ? @parent.name.to_s : @parent.to_s
                    end
      { type: "parent_condition", parent: parent_name, expression: @expression }
    end
  end

  # Represents an OR group of conditions. The task proceeds when ANY condition is met.
  #
  # @example Wait for either a sleep or event
  #   Hatchet.or_(
  #     Hatchet::SleepCondition.new(60),
  #     Hatchet::UserEventCondition.new(event_key: "start")
  #   )
  class OrCondition
    # @return [Array] The conditions in this OR group
    attr_reader :conditions

    # @param conditions [Array] Conditions to OR together
    def initialize(*conditions)
      @conditions = conditions
    end

    # @return [Hash]
    def to_h
      { type: "or", conditions: @conditions.map(&:to_h) }
    end
  end

  # Create an OR condition group
  #
  # @param conditions [Array] Conditions to OR together
  # @return [OrCondition]
  def self.or_(*conditions)
    OrCondition.new(*conditions)
  end
end
