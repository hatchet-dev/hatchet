# frozen_string_literal: true

require_relative "../spec_helper"
require_relative "worker"

RSpec.describe "CancelWorkflow" do
  it "cancels a workflow run" do
    ref = CANCELLATION_WORKFLOW.run_no_wait

    # Sleep for a long time since we only need cancellation to happen eventually
    sleep 10

    30.times do
      run = hatchet.runs.get(ref.workflow_run_id)

      if run.status == "RUNNING"
        sleep 1
        next
      end

      expect(run.status).to eq("CANCELLED")
      expect(run.output).to be_nil
      break
    end
  end
end
