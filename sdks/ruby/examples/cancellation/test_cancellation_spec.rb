# frozen_string_literal: true

require_relative "../spec_helper"
require_relative "worker"

RSpec.describe "CancelWorkflow" do
  it "cancels a workflow run" do
    ref = CANCELLATION_WORKFLOW.run_no_wait

    # Wait for the cancellation to happen
    sleep 10

    # Poll until the run reaches a terminal state
    run = HATCHET.runs.poll(ref.workflow_run_id, interval: 1.0, timeout: 60)

    expect(run.status).to eq("CANCELLED")
  end
end
