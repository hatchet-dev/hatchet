# frozen_string_literal: true

require_relative "../spec_helper"
require_relative "worker"

RSpec.describe "UnitTesting" do
  let(:input) { { "key" => "test_key", "number" => 42 } }
  let(:additional_metadata) { { "meta_key" => "meta_value" } }
  let(:lifespan) { { "mock_db_url" => "sqlite:///:memory:" } }
  let(:retry_count) { 1 }

  let(:expected_output) do
    {
      "key" => input["key"],
      "number" => input["number"],
      "additional_metadata" => additional_metadata,
      "retry_count" => retry_count,
      "mock_db_url" => lifespan["mock_db_url"]
    }
  end

  [
    :SYNC_STANDALONE,
    :DURABLE_SYNC_STANDALONE
  ].each do |const|
    it "unit tests #{const}" do
      task = Object.const_get(const)
      result = task.mock_run(
        input: input,
        additional_metadata: additional_metadata,
        lifespan: lifespan,
        retry_count: retry_count
      )

      expect(result).to eq(expected_output)
    end
  end

  it "unit tests complex workflow with parent outputs" do
    task = COMPLEX_UNIT_TEST_WORKFLOW
    parent_output = expected_output

    result = task.tasks[:sync_complex_workflow].mock_run(
      input: input,
      additional_metadata: additional_metadata,
      lifespan: lifespan,
      retry_count: retry_count,
      parent_outputs: { "start" => parent_output }
    )

    expect(result).to eq(parent_output)
  end
end
