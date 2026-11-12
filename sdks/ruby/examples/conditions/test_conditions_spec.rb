# frozen_string_literal: true

require_relative "../spec_helper"
require_relative "worker"

# Poll until the task reaches COMPLETED in the run's details.
def wait_for_task_to_complete(client, run_id, task_name, timeout: 60, interval: 0.5)
  deadline = Time.now + timeout

  while Time.now < deadline
    begin
      details = client.runs.get_details(run_id)
      # a task's display name is "<step readable id>-<unix timestamp>"
      return if details.tasks&.any? { |t| t.status == "COMPLETED" && t.display_name.start_with?("#{task_name}-") }
    rescue HatchetSdkRest::ApiError => e
      raise unless e.code == 404
    end
    sleep interval
  end

  raise "#{task_name} did not complete within #{timeout}s for run #{run_id}"
end

RSpec.describe "TaskConditionWorkflow" do
  it "runs the condition workflow with event triggers" do
    ref = TASK_CONDITION_WORKFLOW.run_no_wait

    # wait_for_event's event match is registered when start completes, so anchor on that.
    wait_for_task_to_complete(HATCHET, ref.workflow_run_id, "start")

    # Push twice: the read model can show start COMPLETED just before the engine
    # registers the match, and an unmatched event is dropped silently.
    2.times do
      HATCHET.events.create(key: "wait_for_event:start", data: {})
      sleep 2
    end

    result = ref.result

    expect(result["skip_with_multiple_parents"]).to eq({ "skipped" => true })
    expect([result["left_branch"], result["right_branch"]]).to include({ "skipped" => true })

    branch_random_number = result["left_branch"]["random_number"] || result["right_branch"]["random_number"]

    expect(result["sum"]["sum"]).to eq(
      result["start"]["random_number"] +
      result["wait_for_event"]["random_number"] +
      result["wait_for_sleep"]["random_number"] +
      result["skip_on_event"]["random_number"] +
      branch_random_number
    )
  end
end
