# frozen_string_literal: true

require_relative "../spec_helper"
require_relative "worker"

RSpec.describe "LoggingWorkflow" do
  it "runs the logging workflow" do
    result = LOGGING_WORKFLOW.run

    expect(result["root_logger"]["status"]).to eq("success")
  end
end
