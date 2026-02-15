# frozen_string_literal: true

# > Simple

require "hatchet-sdk"

HATCHET = Hatchet::Client.new(debug: true) unless defined?(HATCHET)

CHILD_TASK_WF = HATCHET.workflow(name: "SimpleWorkflow")

CHILD_TASK_WF.task(:step1) do |input, ctx|
  puts "executed step1: #{input['message']}"
  { "transformed_message" => input["message"].upcase }
end

# !!

def main
  worker = HATCHET.worker("test-worker", slots: 1, workflows: [CHILD_TASK_WF])
  worker.start
end

main if __FILE__ == $PROGRAM_NAME
