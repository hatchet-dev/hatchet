# frozen_string_literal: true

require "hatchet-sdk"

HATCHET = Hatchet::Client.new(debug: true) unless defined?(HATCHET)

# > BulkFanoutParent
BULK_PARENT_WF = HATCHET.workflow(name: "BulkFanoutParent")
BULK_CHILD_WF = HATCHET.workflow(name: "BulkFanoutChild")

BULK_PARENT_WF.task(:spawn, execution_timeout: 300) do |input, ctx|
  n = input["n"] || 100

  # Create each workflow run to spawn
  child_workflow_runs = n.times.map do |i|
    BULK_CHILD_WF.create_bulk_run_item(
      input: { "a" => i.to_s },
      key: "child#{i}",
      options: Hatchet::TriggerWorkflowOptions.new(
        additional_metadata: { "hello" => "earth" }
      )
    )
  end

  # Run workflows in bulk to improve performance
  spawn_results = BULK_CHILD_WF.run_many(child_workflow_runs)

  { "results" => spawn_results }
end

BULK_CHILD_WF.task(:process) do |input, ctx|
  puts "child process #{input['a']}"
  { "status" => "success #{input['a']}" }
end

BULK_CHILD_WF.task(:process2) do |input, ctx|
  puts "child process2"
  { "status2" => "success" }
end

# !!

def main
  worker = HATCHET.worker(
    "fanout-worker", slots: 40, workflows: [BULK_PARENT_WF, BULK_CHILD_WF]
  )
  worker.start
end

main if __FILE__ == $PROGRAM_NAME
