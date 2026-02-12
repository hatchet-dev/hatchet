# frozen_string_literal: true

require_relative "../spec_helper"
require_relative "worker"

RSpec.describe "OnFailureWorkflow" do
  it "runs the on_failure task after workflow failure" do
    ref = ON_FAILURE_WF.run_no_wait

    expect { ref.result }.to raise_error(/step1 failed/)

    sleep 5 # Wait for the on_failure job to finish

    details = HATCHET.runs.get(ref.workflow_run_id)

    expect(details.tasks.length).to eq(2)

    completed_count = details.tasks.count { |t| t.status == "COMPLETED" }
    failed_count = details.tasks.count { |t| t.status == "FAILED" }

    expect(completed_count).to eq(1)
    expect(failed_count).to eq(1)

    completed_task = details.tasks.find { |t| t.status == "COMPLETED" }
    failed_task = details.tasks.find { |t| t.status == "FAILED" }

    expect(completed_task.display_name).to include("on_failure")
    expect(failed_task.display_name).to include("step1")
  end
end
