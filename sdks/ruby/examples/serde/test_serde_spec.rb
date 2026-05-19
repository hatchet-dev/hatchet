# frozen_string_literal: true

require_relative "../spec_helper"
require_relative "worker"

RSpec.describe "SerdeWorkflow" do
  it "compresses and decompresses via custom serde" do
    result = SERDE_WORKFLOW.run

    # The generate_result output should be compressed (not equal to the raw value)
    expect(result["generate_result"]["result"]).not_to eq("my_result")

    # The read_result step should decompress and return the original value
    expect(result["read_result"]["final_result"]).to eq("my_result")
  end
end
