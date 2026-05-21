# frozen_string_literal: true

require_relative "../spec_helper"
require_relative "worker"

RSpec.describe "TaskConditionWorkflow" do
  it "runs the condition workflow with event triggers" do
    ref = TASK_CONDITION_WORKFLOW.run_no_wait

    wait_for_running_status(HATCHET, ref.workflow_run_id)
    sleep 5

    HATCHET.events.create(key: "wait_for_event:start", data: {})

    result = ref.result
    expect(result["sum"]["sum"]).to be_a(Integer)
  end
end
