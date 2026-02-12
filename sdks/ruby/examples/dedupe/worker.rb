# frozen_string_literal: true

require "hatchet-sdk"

HATCHET = Hatchet::Client.new(debug: true)

DEDUPE_PARENT_WF = hatchet.workflow(name: "DedupeParent")
DEDUPE_CHILD_WF = hatchet.workflow(name: "DedupeChild")

DEDUPE_PARENT_WF.task(:spawn, execution_timeout: 60) do |input, ctx|
  puts "spawning child"

  results = []

  2.times do |i|
    begin
      results << DEDUPE_CHILD_WF.run(
        options: Hatchet::TriggerWorkflowOptions.new(
          additional_metadata: { "dedupe" => "test" },
          key: "child#{i}"
        )
      )
    rescue Hatchet::DedupeViolationError => e
      puts "dedupe violation #{e}"
      next
    end
  end

  puts "results #{results}"
  { "results" => results }
end

DEDUPE_CHILD_WF.task(:process) do |input, ctx|
  sleep 3
  puts "child process"
  { "status" => "success" }
end

DEDUPE_CHILD_WF.task(:process2) do |input, ctx|
  puts "child process2"
  { "status2" => "success" }
end

def main
  worker = hatchet.worker(
    "fanout-worker", slots: 100, workflows: [DEDUPE_PARENT_WF, DEDUPE_CHILD_WF]
  )
  worker.start
end

main if __FILE__ == $PROGRAM_NAME
