# frozen_string_literal: true

require_relative "../spec_helper"
require_relative "worker"

RSpec.describe "DAGWorkflow" do
  it "runs the DAG workflow" do
    result = DAG_WORKFLOW.run

    one = result["step1"]["random_number"]
    two = result["step2"]["random_number"]
    expect(result["step3"]["sum"]).to eq(one + two)
    expect(result["step4"]["step4"]).to eq("step4")
  end
end
