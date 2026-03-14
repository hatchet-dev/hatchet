# frozen_string_literal: true

require "hatchet-sdk"

HATCHET = Hatchet::Client.new(debug: true) unless defined?(HATCHET)

ON_SUCCESS_WORKFLOW = HATCHET.workflow(name: "OnSuccessWorkflow")

FIRST_TASK = ON_SUCCESS_WORKFLOW.task(:first_task) do |_input, _ctx|
  puts "First task completed successfully"
end

SECOND_TASK = ON_SUCCESS_WORKFLOW.task(:second_task, parents: [FIRST_TASK]) do |_input, _ctx|
  puts "Second task completed successfully"
end

ON_SUCCESS_WORKFLOW.task(:third_task, parents: [FIRST_TASK, SECOND_TASK]) do |_input, _ctx|
  puts "Third task completed successfully"
end

ON_SUCCESS_WORKFLOW.task(:fourth_task) do |_input, _ctx|
  puts "Fourth task completed successfully"
end

ON_SUCCESS_WORKFLOW.on_success_task do |_input, _ctx|
  puts "On success task completed successfully"
end

def main
  worker = HATCHET.worker("on-success-worker", workflows: [ON_SUCCESS_WORKFLOW])
  worker.start
end

main if __FILE__ == $PROGRAM_NAME
