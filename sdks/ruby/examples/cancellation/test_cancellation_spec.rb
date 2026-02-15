# frozen_string_literal: true

require_relative "../spec_helper"
require_relative "worker"

RSpec.describe "CancelWorkflow" do
  it "cancels a workflow run" do
    ref = CANCELLATION_WORKFLOW.run_no_wait

    # Poll until the run reaches a terminal state (replaces fixed sleep 10 + polling loop)
    run = HATCHET.runs.poll(ref.workflow_run_id, interval: 1.0, timeout: 60)

    expect(run.status).to eq("CANCELLED")
  end
end
