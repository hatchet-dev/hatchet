# frozen_string_literal: true

require "hatchet-sdk"

HATCHET = Hatchet::Client.new(debug: true)

# > FanoutParent
FANOUT_PARENT_WF = HATCHET.workflow(name: "FanoutParent")
FANOUT_CHILD_WF = HATCHET.workflow(name: "FanoutChild")

FANOUT_PARENT_WF.task(:spawn, execution_timeout: 300) do |input, ctx|
  puts "spawning child"
  n = input["n"] || 100

  result = FANOUT_CHILD_WF.run_many(
    n.times.map do |i|
      FANOUT_CHILD_WF.create_bulk_run_item(
        input: { "a" => i.to_s },
        options: Hatchet::TriggerWorkflowOptions.new(
          additional_metadata: { "hello" => "earth" },
          key: "child#{i}"
        )
      )
    end
  )

  puts "results #{result}"
  { "results" => result }
end

# > FanoutChild
FANOUT_CHILD_PROCESS = FANOUT_CHILD_WF.task(:process) do |input, ctx|
  puts "child process #{input['a']}"
  { "status" => input["a"] }
end

FANOUT_CHILD_WF.task(:process2, parents: [FANOUT_CHILD_PROCESS]) do |input, ctx|
  process_output = ctx.task_output(FANOUT_CHILD_PROCESS)
  a = process_output["status"]
  { "status2" => "#{a}2" }
end

def main
  worker = HATCHET.worker("fanout-worker", slots: 40, workflows: [FANOUT_PARENT_WF, FANOUT_CHILD_WF])
  worker.start
end

main if __FILE__ == $PROGRAM_NAME
