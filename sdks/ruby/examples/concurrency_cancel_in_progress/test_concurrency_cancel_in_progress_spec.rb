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

    # Wait for the OLAP repo to catch up
    sleep 5

    runs = HATCHET.runs.list(additional_metadata: { "test_run_id" => test_run_id }).rows
    runs.sort_by! { |r| (r.additional_metadata || {})["i"].to_i }

    expect(runs.length).to eq(10)
    expect((runs.last.additional_metadata || {})["i"]).to eq("9")
    expect(runs.last.status).to eq("COMPLETED")
    expect(runs[0..-2].all? { |r| r.status == "CANCELLED" }).to be true
  end
end
