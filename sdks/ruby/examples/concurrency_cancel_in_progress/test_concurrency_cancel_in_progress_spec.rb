# frozen_string_literal: true

require "securerandom"
require_relative "../spec_helper"
require_relative "worker"

RSpec.describe "ConcurrencyCancelInProgress" do
  it "cancels in-progress runs when concurrency limit exceeded" do
    test_run_id = SecureRandom.uuid
    refs = []

    10.times do |i|
      ref = CONCURRENCY_CANCEL_IN_PROGRESS_WORKFLOW.run_no_wait(
        { "group" => "A" },
        options: Hatchet::TriggerWorkflowOptions.new(
          additional_metadata: { "test_run_id" => test_run_id, "i" => i.to_s }
        )
      )
      refs << ref
      sleep 1
    end

    refs.each do |ref|
      puts "Waiting for run #{ref.workflow_run_id} to complete"
      ref.result rescue nil
    end

    # Poll until the OLAP repo has caught up (replaces fixed sleep 5)
    all_rows = nil
    30.times do
      all_rows = HATCHET.runs.list(additional_metadata: { "test_run_id" => test_run_id }, limit: 100).rows
      break if all_rows.length >= 10

      sleep 0.5
    end
    # Filter to workflow-level runs only (exclude individual task runs)
    runs = all_rows.reject { |r| r.respond_to?(:type) && r.type == "TASK" }
    runs.sort_by! { |r| (r.additional_metadata || {})["i"].to_i }

    expect(runs.length).to eq(10)
    expect((runs.last.additional_metadata || {})["i"]).to eq("9")
    expect(runs.last.status).to eq("COMPLETED")
    expect(runs[0..-2].all? { |r| r.status == "CANCELLED" }).to be true
  end
end
