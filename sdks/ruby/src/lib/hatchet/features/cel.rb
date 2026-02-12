# frozen_string_literal: true

module Hatchet
  module Features
    # Result types for CEL expression evaluation
    CELSuccess = Struct.new(:status, :output, keyword_init: true) do
      def initialize(output:)
        super(status: 'success', output: output)
      end
    end

    CELFailure = Struct.new(:status, :error, keyword_init: true) do
      def initialize(error:)
        super(status: 'failure', error: error)
      end
    end

    CELEvaluationResult = Struct.new(:result, keyword_init: true)

    # CEL client for debugging CEL expressions within Hatchet
    #
    # This class provides a high-level interface for testing and debugging
    # CEL (Common Expression Language) expressions used in filters and conditions.
    #
    # @example Debugging a CEL expression
    #   result = cel_client.debug(
    #     expression: 'input.value > 10',
    #     input: { value: 15 }
    #   )
    #   if result.result.status == 'success'
    #     puts "Output: #{result.result.output}"
    #   else
    #     puts "Error: #{result.result.error}"
    #   end
    #
    # @since 0.1.0
    class CEL
      # Initializes a new CEL client instance
      #
      # @param rest_client [Object] The configured REST client for API communication
      # @param config [Hatchet::Config] The Hatchet configuration containing tenant_id and other settings
      # @return [void]
      # @since 0.1.0
      def initialize(rest_client, config)
        @rest_client = rest_client
        @config = config
        @cel_api = HatchetSdkRest::CELApi.new(rest_client)
      end

      # Debug a CEL expression with provided input and optional metadata
      #
      # Useful for testing and validating CEL expressions and debugging issues in production.
      #
      # @param expression [String] The CEL expression to debug
      # @param input [Hash] The input, which simulates the workflow run input
      # @param additional_metadata [Hash, nil] Additional metadata simulating metadata sent with an event or workflow run
      # @param filter_payload [Hash, nil] The filter payload simulating a payload set on a previously-created filter
      # @return [CELEvaluationResult] A result containing either a CELSuccess or CELFailure
      # @raise [RuntimeError] If no response is received from the CEL debug API
      # @raise [HatchetSdkRest::ApiError] If the API request fails
      # @example
      #   result = cel_client.debug(
      #     expression: 'input.count > 5 && metadata.env == "prod"',
      #     input: { count: 10 },
      #     additional_metadata: { env: 'prod' }
      #   )
      def debug(expression:, input:, additional_metadata: nil, filter_payload: nil)
        request = HatchetSdkRest::V1CELDebugRequest.new(
          expression: expression,
          input: input,
          additional_metadata: additional_metadata,
          filter_payload: filter_payload
        )

        result = @cel_api.v1_cel_debug(@config.tenant_id, request)

        if result.status == 'ERROR' || result.status == :ERROR
          raise 'No error message received from CEL debug API.' if result.error.nil?

          return CELEvaluationResult.new(result: CELFailure.new(error: result.error))
        end

        raise 'No output received from CEL debug API.' if result.output.nil?

        CELEvaluationResult.new(result: CELSuccess.new(output: result.output))
      end
    end
  end
end
