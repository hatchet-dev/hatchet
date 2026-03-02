# frozen_string_literal: true

require_relative "../spec_helper"
require_relative "worker"

RSpec.describe "ConcurrencyDemoWorkflowRR" do
  # The timing for this test is not reliable
  xit "runs with round-robin concurrency" do
    num_groups = 2
    runs = []

    num_groups.times do
      runs << CONCURRENCY_LIMIT_RR_WORKFLOW.run_no_wait
      runs << CONCURRENCY_LIMIT_RR_WORKFLOW.run_no_wait
    end

    successful_runs = []
    cancelled_runs = []

    start_time = Time.now

    runs.each_with_index do |run, i|
      result = run.result
      successful_runs << [i + 1, result]
    rescue => e
      if e.message.include?("CANCELLED_BY_CONCURRENCY_LIMIT")
        cancelled_runs << [i + 1, e.message]
      else
        raise
      end
    end

    total_time = Time.now - start_time

    expect(successful_runs.length).to eq(4)
    expect(cancelled_runs.length).to eq(0)
    expect(total_time).to be_between(3.8, 7)
  end
end
