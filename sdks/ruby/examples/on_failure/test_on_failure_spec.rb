# frozen_string_literal: true

require_relative "../spec_helper"
require_relative "worker"

RSpec.describe "OnFailureWorkflow" do
  it "runs the on_failure task after workflow failure" do
    ref = ON_FAILURE_WF.run_no_wait

    expect { ref.result }.to raise_error(/step1 failed/)

    # Poll until both tasks are in a terminal state (replaces fixed sleep 5)
    details = nil
    30.times do
      details = HATCHET.runs.get_details(ref.workflow_run_id)
      break if details.tasks.length >= 2 && details.tasks.all? { |t| %w[COMPLETED FAILED].include?(t.status) }

      sleep 0.5
    end

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
