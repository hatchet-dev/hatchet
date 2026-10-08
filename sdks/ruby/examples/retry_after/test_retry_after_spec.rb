# frozen_string_literal: true

require_relative "../spec_helper"
require_relative "worker"

RSpec.describe "RetryAfterError" do
  it "waits the requested delay before retrying" do
    start = Process.clock_gettime(Process::CLOCK_MONOTONIC)

    result = RETRY_AFTER_UPSTREAM_DELAY.run({ "failing_attempts" => 2 })

    expect(result["attempt"]).to eq(2)
    expect(Process.clock_gettime(Process::CLOCK_MONOTONIC) - start).to be >= 4
  end

  it "fails once the retries are exhausted" do
    ref = RETRY_AFTER_EXPONENTIAL_BACKOFF.run_no_wait({ "failing_attempts" => 100 })

    expect { ref.result }.to raise_error(Hatchet::FailedRunError)

    task_run = nil
    60.times do
      begin
        task_run = HATCHET.runs.get_task_run(ref.workflow_run_id)
        break if task_run.retry_count == 3
      rescue HatchetSdkRest::ApiError => e
        raise unless e.code == 404
      end

      sleep 0.5
    end

    expect(task_run.retry_count).to eq(3)
  end
end
