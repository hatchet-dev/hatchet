# frozen_string_literal: true

require "securerandom"
require_relative "../spec_helper"
require_relative "worker"

RSpec.describe "ConcurrencyCancelNewest" do
  # TODO-RUBY: fix this test
  xit "cancels newest runs when concurrency limit exceeded" do
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

    # Poll until the OLAP repo has caught up (replaces fixed sleep 5)
    all_runs = nil
    30.times do
      all_runs = HATCHET.runs.list(
        additional_metadata: { "test_run_id" => test_run_id },
        limit: 100
      ).rows
      break if all_runs.length >= 11

      sleep 0.5
    end

    successful_run = HATCHET.runs.get(to_run.workflow_run_id)
    expect(successful_run.status).to eq("COMPLETED")

    # Filter to workflow-level runs only
    workflow_runs = all_runs.reject { |r| r.respond_to?(:type) && r.type == "TASK" }

    other_runs = workflow_runs.reject { |r| r.metadata.id == to_run.workflow_run_id }
    expect(other_runs.all? { |r| r.status == "CANCELLED" }).to be true
  end
end
