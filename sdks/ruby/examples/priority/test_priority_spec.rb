# frozen_string_literal: true

require "securerandom"
require_relative "../spec_helper"
require_relative "worker"

RSpec.describe "PriorityWorkflow" do
  def priority_to_int(priority)
    case priority
    when "high" then 3
    when "medium" then 2
    when "low" then 1
    when "default" then DEFAULT_PRIORITY
    else raise "Invalid priority: #{priority}"
    end
  end

  it "executes runs in priority order" do
    test_run_id = SecureRandom.uuid
    choices = %w[low medium high default]
    n = 30

    run_refs = PRIORITY_WORKFLOW.run_many_no_wait(
      n.times.map do |ix|
        priority = choices.sample
        PRIORITY_WORKFLOW.create_bulk_run_item(
          options: Hatchet::TriggerWorkflowOptions.new(
            priority: priority_to_int(priority),
            additional_metadata: {
              "priority" => priority,
              "key" => ix,
              "test_run_id" => test_run_id
            }
          )
        )
      end
    )

    # Wait for all runs to complete
    run_refs.each(&:result)

    workflows = hatchet.workflows.list(workflow_name: PRIORITY_WORKFLOW.name)
    expect(workflows.rows).not_to be_empty

    workflow = workflows.rows.find { |w| w.name == PRIORITY_WORKFLOW.name }
    expect(workflow).not_to be_nil

    runs = hatchet.runs.list(
      workflow_ids: [workflow.metadata.id],
      additional_metadata: { "test_run_id" => test_run_id },
      limit: 1000
    )

    sorted_runs = runs.rows.sort_by(&:started_at)
    expect(sorted_runs.length).to eq(n)

    sorted_runs.each_cons(2) do |curr, nxt|
      curr_priority = (curr.additional_metadata || {})["priority"] || "low"
      nxt_priority = (nxt.additional_metadata || {})["priority"] || "low"

      # Run start times should be in order of priority
      expect(priority_to_int(curr_priority)).to be >= priority_to_int(nxt_priority)
    end
  end
end
