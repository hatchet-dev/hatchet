# frozen_string_literal: true

require "securerandom"
require_relative "../spec_helper"
require_relative "worker"

RSpec.describe "BulkReplay" do
  it "replays failed runs in bulk" do
    test_run_id = SecureRandom.uuid
    n = 100

    # First batch -- all will fail on first attempt
    expect do
      BULK_REPLAY_TEST_1.run_many(
        (n + 1).times.map do
          BULK_REPLAY_TEST_1.create_bulk_run_item(
            options: Hatchet::TriggerWorkflowOptions.new(
              additional_metadata: { "test_run_id" => test_run_id }
            )
          )
        end
      )
    end.to raise_error

    expect do
      BULK_REPLAY_TEST_2.run_many(
        ((n / 2) - 1).times.map do
          BULK_REPLAY_TEST_2.create_bulk_run_item(
            options: Hatchet::TriggerWorkflowOptions.new(
              additional_metadata: { "test_run_id" => test_run_id }
            )
          )
        end
      )
    end.to raise_error

    expect do
      BULK_REPLAY_TEST_3.run_many(
        ((n / 2) - 2).times.map do
          BULK_REPLAY_TEST_3.create_bulk_run_item(
            options: Hatchet::TriggerWorkflowOptions.new(
              additional_metadata: { "test_run_id" => test_run_id }
            )
          )
        end
      )
    end.to raise_error

    workflow_ids = [BULK_REPLAY_TEST_1.id, BULK_REPLAY_TEST_2.id, BULK_REPLAY_TEST_3.id]

    # Should result in two batches of replays
    HATCHET.runs.bulk_replay(
      filters: {
        workflow_ids: workflow_ids,
        additional_metadata: { "test_run_id" => test_run_id }
      }
    )

    sleep 10

    runs = HATCHET.runs.list(
      workflow_ids: workflow_ids,
      additional_metadata: { "test_run_id" => test_run_id },
      limit: 1000
    )

    total_expected = (n + 1) + (n / 2 - 1) + (n / 2 - 2)
    expect(runs.rows.length).to eq(total_expected)

    runs.rows.each do |run|
      expect(run.status).to eq("COMPLETED")
    end
  end
end
