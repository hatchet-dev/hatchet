# frozen_string_literal: true

RSpec.describe Hatchet::Features::CEL do
  let(:valid_token) { "eyJhbGciOiJIUzI1NiJ9.eyJzdWIiOiJ0ZXN0LXRlbmFudCJ9.signature" }
  let(:config) { Hatchet::Config.new(token: valid_token) }
  let(:rest_client) { instance_double("ApiClient") }
  let(:cel_api) { instance_double("HatchetSdkRest::CELApi") }
  let(:cel_client) { described_class.new(rest_client, config) }

  before do
    allow(HatchetSdkRest::CELApi).to receive(:new).with(rest_client).and_return(cel_api)
  end

  around do |example|
    original_env = ENV.select { |k, _| k.start_with?("HATCHET_CLIENT_") }
    ENV.keys.select { |k| k.start_with?("HATCHET_CLIENT_") }.each { |k| ENV.delete(k) }
    example.run
    ENV.keys.select { |k| k.start_with?("HATCHET_CLIENT_") }.each { |k| ENV.delete(k) }
    original_env.each { |k, v| ENV[k] = v }
  end

  describe "#initialize" do
    it "creates a new CEL client with required dependencies" do
      expect(cel_client).to be_a(described_class)
      expect(cel_client.instance_variable_get(:@config)).to eq(config)
      expect(cel_client.instance_variable_get(:@rest_client)).to eq(rest_client)
    end

    it "initializes CEL API client" do
      described_class.new(rest_client, config)
      expect(HatchetSdkRest::CELApi).to have_received(:new).with(rest_client)
    end
  end

  describe "#debug" do
    let(:expression) { "input.value > 10" }
    let(:input) { { value: 15 } }
    let(:debug_request) { instance_double("HatchetSdkRest::V1CELDebugRequest") }

    before do
      allow(HatchetSdkRest::V1CELDebugRequest).to receive(:new).and_return(debug_request)
    end

    it "returns a success result when expression evaluates successfully" do
      result_obj = double("result", status: "SUCCESS", output: true, error: nil)
      allow(cel_api).to receive(:v1_cel_debug).with("test-tenant", debug_request).and_return(result_obj)

      result = cel_client.debug(expression: expression, input: input)

      expect(result).to be_a(Hatchet::Features::CELEvaluationResult)
      expect(result.result).to be_a(Hatchet::Features::CELSuccess)
      expect(result.result.status).to eq("success")
      expect(result.result.output).to eq(true)
    end

    it "returns a failure result when expression has an error" do
      result_obj = double("result", status: "ERROR", output: nil, error: "invalid expression")
      allow(cel_api).to receive(:v1_cel_debug).with("test-tenant", debug_request).and_return(result_obj)

      result = cel_client.debug(expression: expression, input: input)

      expect(result).to be_a(Hatchet::Features::CELEvaluationResult)
      expect(result.result).to be_a(Hatchet::Features::CELFailure)
      expect(result.result.status).to eq("failure")
      expect(result.result.error).to eq("invalid expression")
    end

    it "raises error when error status but no error message" do
      result_obj = double("result", status: "ERROR", output: nil, error: nil)
      allow(cel_api).to receive(:v1_cel_debug).with("test-tenant", debug_request).and_return(result_obj)

      expect { cel_client.debug(expression: expression, input: input) }
        .to raise_error(RuntimeError, "No error message received from CEL debug API.")
    end

    it "raises error when success status but no output" do
      result_obj = double("result", status: "SUCCESS", output: nil, error: nil)
      allow(cel_api).to receive(:v1_cel_debug).with("test-tenant", debug_request).and_return(result_obj)

      expect { cel_client.debug(expression: expression, input: input) }
        .to raise_error(RuntimeError, "No output received from CEL debug API.")
    end

    it "passes all parameters to the API" do
      result_obj = double("result", status: "SUCCESS", output: false, error: nil)
      allow(cel_api).to receive(:v1_cel_debug).and_return(result_obj)

      cel_client.debug(
        expression: expression,
        input: input,
        additional_metadata: { env: "prod" },
        filter_payload: { threshold: 5 },
      )

      expect(HatchetSdkRest::V1CELDebugRequest).to have_received(:new).with(
        expression: expression,
        input: input,
        additional_metadata: { env: "prod" },
        filter_payload: { threshold: 5 },
      )
    end
  end
end

RSpec.describe Hatchet::Features::CELSuccess do
  it "creates a success result" do
    result = described_class.new(output: true)
    expect(result.status).to eq("success")
    expect(result.output).to eq(true)
  end
end

RSpec.describe Hatchet::Features::CELFailure do
  it "creates a failure result" do
    result = described_class.new(error: "bad expression")
    expect(result.status).to eq("failure")
    expect(result.error).to eq("bad expression")
  end
end

RSpec.describe Hatchet::Features::CELEvaluationResult do
  it "wraps a success result" do
    success = Hatchet::Features::CELSuccess.new(output: true)
    result = described_class.new(result: success)
    expect(result.result).to eq(success)
  end

  it "wraps a failure result" do
    failure = Hatchet::Features::CELFailure.new(error: "error")
    result = described_class.new(result: failure)
    expect(result.result).to eq(failure)
  end
end
