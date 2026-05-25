# frozen_string_literal: true

require_relative "../spec_helper"
require_relative "worker"

RSpec.describe "NonRetryableWorkflow" do
  it "does not retry non-retryable exceptions" do
    ref = NON_RETRYABLE_WORKFLOW.run_no_wait

    expect { ref.result }.to raise_error(Hatchet::FailedRunError)

    # Poll until all task events have been recorded.
    # Rescue 404 errors: the run record may not be visible immediately after result raises.
    run_details = nil
    60.times do
      begin
        run_details = HATCHET.runs.get_details(ref.workflow_run_id)
        failed_events = run_details.task_events.select { |e| e.event_type == "FAILED" }
        break if failed_events.length >= 3
      rescue HatchetSdkRest::ApiError => e
        raise unless e.code == 404
      end

      sleep 0.5
    end

    # Only the task with the wrong exception type should have retrying events
    retrying_events = run_details.task_events.select { |e| e.event_type == "RETRYING" }
    expect(retrying_events.length).to eq(1)

    # Three failed events: two failing initial runs + one retry failure
    failed_events = run_details.task_events.select { |e| e.event_type == "FAILED" }
    expect(failed_events.length).to eq(3)
  end
end
