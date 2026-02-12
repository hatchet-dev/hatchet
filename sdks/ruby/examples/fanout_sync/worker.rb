# frozen_string_literal: true

require "hatchet-sdk"

HATCHET = Hatchet::Client.new(debug: true)

SYNC_FANOUT_PARENT = hatchet.workflow(name: "SyncFanoutParent")
SYNC_FANOUT_CHILD = hatchet.workflow(name: "SyncFanoutChild")

SYNC_FANOUT_PARENT.task(:spawn, execution_timeout: 300) do |input, ctx|
  puts "spawning child"
  n = input["n"] || 5

  results = SYNC_FANOUT_CHILD.run_many(
    n.times.map do |i|
      SYNC_FANOUT_CHILD.create_bulk_run_item(
        input: { "a" => i.to_s },
        key: "child#{i}",
        options: Hatchet::TriggerWorkflowOptions.new(
          additional_metadata: { "hello" => "earth" }
        )
      )
    end
  )

  puts "results #{results}"
  { "results" => results }
end

SYNC_PROCESS = SYNC_FANOUT_CHILD.task(:process) do |input, ctx|
  { "status" => "success #{input['a']}" }
end

SYNC_FANOUT_CHILD.task(:process2, parents: [SYNC_PROCESS]) do |input, ctx|
  process_output = ctx.task_output(SYNC_PROCESS)
  a = process_output["status"]
  { "status2" => "#{a}2" }
end

def main
  worker = hatchet.worker(
    "sync-fanout-worker",
    slots: 40,
    workflows: [SYNC_FANOUT_PARENT, SYNC_FANOUT_CHILD]
  )
  worker.start
end

main if __FILE__ == $PROGRAM_NAME
