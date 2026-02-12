# frozen_string_literal: true

require "hatchet-sdk"

HATCHET = Hatchet::Client.new(debug: true)

SCHEDULED_WORKFLOW = HATCHET.workflow(name: "ScheduledWorkflow")

SCHEDULED_WORKFLOW.task(:scheduled_task) do |input, ctx|
  puts "Scheduled task executed at #{Time.now}"
  { "status" => "success" }
end

# > Programmatic Schedule
def schedule_workflow
  future_time = Time.now + 60 # 1 minute from now
  SCHEDULED_WORKFLOW.schedule(future_time, input: { "message" => "scheduled run" })
end

# !!

def main
  worker = HATCHET.worker("scheduled-worker", workflows: [SCHEDULED_WORKFLOW])
  worker.start
end

main if __FILE__ == $PROGRAM_NAME
