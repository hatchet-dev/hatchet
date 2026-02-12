# frozen_string_literal: true

require "hatchet-sdk"

HATCHET = Hatchet::Client.new(debug: true)

ON_SUCCESS_WORKFLOW = hatchet.workflow(name: "OnSuccessWorkflow")

FIRST_TASK = ON_SUCCESS_WORKFLOW.task(:first_task) do |input, ctx|
  puts "First task completed successfully"
end

SECOND_TASK = ON_SUCCESS_WORKFLOW.task(:second_task, parents: [FIRST_TASK]) do |input, ctx|
  puts "Second task completed successfully"
end

ON_SUCCESS_WORKFLOW.task(:third_task, parents: [FIRST_TASK, SECOND_TASK]) do |input, ctx|
  puts "Third task completed successfully"
end

ON_SUCCESS_WORKFLOW.task(:fourth_task) do |input, ctx|
  puts "Fourth task completed successfully"
end

ON_SUCCESS_WORKFLOW.on_success_task do |input, ctx|
  puts "On success task completed successfully"
end

def main
  worker = hatchet.worker("on-success-worker", workflows: [ON_SUCCESS_WORKFLOW])
  worker.start
end

main if __FILE__ == $PROGRAM_NAME
