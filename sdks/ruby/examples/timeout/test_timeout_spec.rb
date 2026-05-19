# frozen_string_literal: true

require_relative "../spec_helper"
require_relative "worker"

RSpec.describe "TimeoutWorkflow" do
  it "times out on execution timeout" do
    ref = TIMEOUT_WF.run_no_wait

    expect { ref.result }.to raise_error(
      /Task exceeded timeout|TIMED_OUT|Workflow run .* failed with multiple errors/
    )
  end

  it "succeeds when timeout is refreshed" do
    result = REFRESH_TIMEOUT_WF.run

    expect(result["refresh_task"]["status"]).to eq("success")
  end
end
