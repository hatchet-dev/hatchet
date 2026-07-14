# frozen_string_literal: true

require "securerandom"
require_relative "../spec_helper"
require_relative "../worker_fixture"
require_relative "worker"

RSpec.describe "Idempotency" do
  around do |example|
    HatchetWorkerFixture.with_worker(
      ["bundle", "exec", "ruby", File.expand_path("worker.rb", __dir__)],
      healthcheck_port: 8005
    ) do
      example.run
    end
  end

  def wait_for_runs(test_run_id, min_runs:)
    runs = nil

    15.times do
      runs = hatchet.runs.list(
        since: Time.now - (5 * 60),
        additional_metadata: { "test_run_id" => test_run_id },
        limit: 20
      )

      break if runs.rows.length >= min_runs

      sleep 1
    end

    runs
  end

  it "prevents duplicate direct triggers" do
    test_run_id = SecureRandom.uuid
    ref1 = IDEMPOTENT_TASK.run_no_wait(
      { "id" => test_run_id },
      options: Hatchet::TriggerWorkflowOptions.new(
        additional_metadata: { "test_run_id" => test_run_id }
      )
    )

    collision = nil

    expect do
      IDEMPOTENT_TASK.run_no_wait({ "id" => test_run_id })
    end.to raise_error(Hatchet::IdempotencyCollisionError) { |error| collision = error }

    expect(collision&.existing_run_external_id).to eq(ref1.workflow_run_id)

    runs = wait_for_runs(test_run_id, min_runs: 1)

    expect(runs).not_to be_nil
    expect(runs.rows.length).to eq(1)
    expect(runs.rows[0].metadata.id).to eq(ref1.workflow_run_id)
  end

  it 'prevents duplicate bulk triggers' do
    test_run_id = SecureRandom.uuid

    collision = nil

    expect do
      IDEMPOTENT_TASK.run_many_no_wait(
        [
          IDEMPOTENT_TASK.create_bulk_run_item(
            input: { 'id' => test_run_id },
            options: Hatchet::TriggerWorkflowOptions.new(additional_metadata: { 'test_run_id' => test_run_id })
          ),
          IDEMPOTENT_TASK.create_bulk_run_item(input: { 'id' => test_run_id })
        ]
      )
    end.to raise_error(Hatchet::BulkTriggerIdempotencyCollisionError) { |error| collision = error }

    expect(collision&.successful_workflow_run_external_ids&.length).to eq(1)
    expect(collision&.collisions&.length).to eq(1)
    expect(collision&.collisions&.first).to be_a(Hatchet::IdempotencyCollisionError)
  end

  it "allows reruns after the short idempotency window expires" do
    test_run_id = SecureRandom.uuid

    4.times do |i|
      if i == 1
        expect do
          IDEMPOTENT_TASK_SHORT_WINDOW.run_no_wait(
            { "id" => test_run_id },
            options: Hatchet::TriggerWorkflowOptions.new(
              additional_metadata: { "test_run_id" => test_run_id }
            )
          )
        end.to raise_error(Hatchet::IdempotencyCollisionError) { |error| expect(error.existing_run_external_id).not_to be_nil }
      else
        IDEMPOTENT_TASK_SHORT_WINDOW.run_no_wait(
          { "id" => test_run_id },
          options: Hatchet::TriggerWorkflowOptions.new(
            additional_metadata: { "test_run_id" => test_run_id }
          )
        )
      end

      sleep(i + 1.5) unless i == 3
    end

    runs = wait_for_runs(test_run_id, min_runs: 3)

    expect(runs.rows.length).to eq(3)
  end

  it "deduplicates event-triggered runs" do
    test_run_id = SecureRandom.uuid
    e1 = hatchet.events.push(
      EVENT_KEY,
      { "id" => test_run_id },
      additional_metadata: { "test_run_id" => test_run_id }
    )
    e2 = hatchet.events.push(
      EVENT_KEY,
      { "id" => test_run_id },
      additional_metadata: { "test_run_id" => test_run_id }
    )

    runs = wait_for_runs(test_run_id, min_runs: 1)

    expect(runs).not_to be_nil
    expect(runs.rows.length).to eq(1)

    details = nil
    15.times do
      details = hatchet.events.list(event_ids: [e1.event_id, e2.event_id])
      break if details.rows.length == 2

      sleep 1
    end

    expect(details).not_to be_nil
    expect(details.rows.length).to eq(2)

    all_triggered_runs = details.rows.flat_map { |row| row.triggered_runs || [] }

    expect(all_triggered_runs.length).to eq(1)

    run = nil
    15.times do
      run = hatchet.runs.get(all_triggered_runs[0].workflow_run_id)
      break unless %w[QUEUED RUNNING].include?(run.status)

      sleep 1
    end

    expect(run.status).to eq("COMPLETED")
  end
end
