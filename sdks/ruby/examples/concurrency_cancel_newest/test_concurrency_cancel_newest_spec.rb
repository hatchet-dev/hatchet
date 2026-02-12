# frozen_string_literal: true

require "securerandom"
require_relative "../spec_helper"
require_relative "worker"

RSpec.describe "ConcurrencyCancelNewest" do
  it "cancels newest runs when concurrency limit exceeded" do
    test_run_id = SecureRandom.uuid

    to_run = CONCURRENCY_CANCEL_NEWEST_WORKFLOW.run_no_wait(
      { "group" => "A" },
      options: Hatchet::TriggerWorkflowOptions.new(
        additional_metadata: { "test_run_id" => test_run_id }
      )
    )

    sleep 1

    to_cancel = CONCURRENCY_CANCEL_NEWEST_WORKFLOW.run_many_no_wait(
      10.times.map do
        CONCURRENCY_CANCEL_NEWEST_WORKFLOW.create_bulk_run_item(
          input: { "group" => "A" },
          options: Hatchet::TriggerWorkflowOptions.new(
            additional_metadata: { "test_run_id" => test_run_id }
          )
        )
      end
    )

    to_run.result
    to_cancel.each { |ref| ref.result rescue nil }

    # Wait for the OLAP repo to catch up
    sleep 5

    successful_run = hatchet.runs.get(to_run.workflow_run_id)
    expect(successful_run.run.status).to eq("COMPLETED")

    all_runs = hatchet.runs.list(
      additional_metadata: { "test_run_id" => test_run_id }
    ).rows

    other_runs = all_runs.reject { |r| r.metadata.id == to_run.workflow_run_id }
    expect(other_runs.all? { |r| r.status == "CANCELLED" }).to be true
  end
end
