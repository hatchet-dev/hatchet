# frozen_string_literal: true

module Hatchet
  # Worker label comparators for affinity scheduling
  module WorkerLabelComparator
    EQUAL = :equal
    NOT_EQUAL = :not_equal
    GREATER_THAN = :greater_than
    LESS_THAN = :less_than
    GREATER_THAN_OR_EQUAL = :greater_than_or_equal
    LESS_THAN_OR_EQUAL = :less_than_or_equal
  end

  # Defines a desired worker label for task scheduling affinity
  #
  # @example Prefer workers with a specific model
  #   Hatchet::DesiredWorkerLabel.new(value: "fancy-ai-model-v2", weight: 10)
  #
  # @example Require workers with enough memory
  #   Hatchet::DesiredWorkerLabel.new(value: 256, required: true, comparator: :less_than)
  class DesiredWorkerLabel
    # @return [String, Integer] The desired label value
    attr_reader :value

    # @return [Integer] Weight for soft scheduling preferences (higher = stronger preference)
    attr_reader :weight

    # @return [Boolean] Whether this label is required (hard constraint)
    attr_reader :required

    # @return [Symbol] Comparator for numeric values
    attr_reader :comparator

    # @param value [String, Integer] Desired label value
    # @param weight [Integer] Scheduling weight (default: 1)
    # @param required [Boolean] Hard requirement (default: false)
    # @param comparator [Symbol] Comparison operator (default: :equal)
    def initialize(value:, weight: 1, required: false, comparator: :equal)
      @value = value
      @weight = weight
      @required = required
      @comparator = comparator
    end

    # @return [Hash]
    def to_h
      {
        value: @value,
        weight: @weight,
        required: @required,
        comparator: @comparator.to_s.upcase
      }
    end
  end

  # Sticky scheduling strategies
  module StickyStrategy
    SOFT = :soft
    HARD = :hard
  end
end
