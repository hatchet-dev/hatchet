# frozen_string_literal: true

module Hatchet
  # Defines a default filter for event-triggered workflows
  #
  # @example Filter with scope and payload
  #   Hatchet::DefaultFilter.new(
  #     expression: "true",
  #     scope: "example-scope",
  #     payload: { "main_character" => "Anna" }
  #   )
  class DefaultFilter
    # @return [String] CEL expression to evaluate
    attr_reader :expression

    # @return [String, nil] Scope for filter matching
    attr_reader :scope

    # @return [Hash] Static payload for the filter
    attr_reader :payload

    # @param expression [String] CEL expression
    # @param scope [String, nil] Filter scope
    # @param payload [Hash] Static payload (default: {})
    def initialize(expression:, scope: nil, payload: {})
      @expression = expression
      @scope = scope
      @payload = payload
    end

    # @return [Hash]
    def to_h
      h = { expression: @expression, payload: @payload }
      h[:scope] = @scope if @scope
      h
    end

    # Convert to a V1::DefaultFilter protobuf message
    # @return [V1::DefaultFilter]
    def to_proto
      payload_bytes = JSON.generate(@payload || {}).encode("UTF-8")

      args = { expression: @expression, payload: payload_bytes }
      args[:scope] = @scope if @scope

      ::V1::DefaultFilter.new(**args)
    end
  end
end
