# frozen_string_literal: true

require_relative "../spec_helper"
require_relative "worker"

RSpec.describe "NonRetryableWorkflow" do
  it "does not retry non-retryable exceptions" do
    ref = NON_RETRYABLE_WORKFLOW.run_no_wait

    expect { ref.result }.to raise_error(Hatchet::FailedRunError)

    # Task events reach the OLAP store asynchronously and are not guaranteed to
    # land in emission order, so the RETRYING event can become visible after the
    # final FAILED event. Poll until both counts are in place instead of reading
    # the event list once.
    # Rescue 404 errors: the run record may not be visible immediately after result raises.
    retrying_events = []
    failed_events = []
    60.times do
      begin
        run_details = HATCHET.runs.get_details(ref.workflow_run_id)
        retrying_events = run_details.task_events.select { |e| e.event_type == "RETRYING" }
        failed_events = run_details.task_events.select { |e| e.event_type == "FAILED" }
        break if retrying_events.length >= 1 && failed_events.length >= 3
      rescue HatchetSdkRest::ApiError => e
        raise unless e.code == 404
      end

      sleep 0.5
    end

    # Only the task with the wrong exception type should have retrying events
    expect(retrying_events.length).to eq(1)

    # Three failed events: two failing initial runs + one retry failure
    expect(failed_events.length).to eq(3)
  end
end
