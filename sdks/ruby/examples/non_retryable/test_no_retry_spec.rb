# frozen_string_literal: true

require_relative "../spec_helper"
require_relative "worker"

RSpec.describe "NonRetryableWorkflow" do
  it "does not retry non-retryable exceptions" do
    ref = NON_RETRYABLE_WORKFLOW.run_no_wait

    expect { ref.result }.to raise_error(Hatchet::FailedRunError)

    sleep 3

    run_details = HATCHET.runs.get_details(ref.workflow_run_id)

    # Only the task with the wrong exception type should have retrying events
    retrying_events = run_details.task_events.select { |e| e.event_type == "RETRYING" }
    expect(retrying_events.length).to eq(1)

    # Three failed events: two failing initial runs + one retry failure
    failed_events = run_details.task_events.select { |e| e.event_type == "FAILED" }
    expect(failed_events.length).to eq(3)
  end
end
