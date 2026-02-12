# frozen_string_literal: true

require_relative "../spec_helper"
require_relative "worker"

RSpec.describe "DurableWorkflow" do
  it "completes a durable sleep then waits for event" do
    ref = DURABLE_WORKFLOW.run_no_wait

    # Wait for the sleep to complete
    sleep(DURABLE_SLEEP_TIME + 2)

    # Push the event to unblock the durable task
    hatchet.events.create(key: DURABLE_EVENT_KEY, data: { "test" => true })

    result = ref.result
    expect(result["durable_task"]["status"]).to eq("success")
  end

  it "handles multi-sleep in durable tasks" do
    result = WAIT_FOR_SLEEP_TWICE.run

    expect(result["runtime"]).to be >= DURABLE_SLEEP_TIME
  end
end
