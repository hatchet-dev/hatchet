# frozen_string_literal: true

require "hatchet-sdk"

HATCHET = Hatchet::Client.new(debug: true) unless defined?(HATCHET)

# > Step 01 Define Cron Task
CRON_WF = HATCHET.workflow(name: "ScheduledWorkflow", on_crons: ["0 * * * *"])

CRON_WF.task(:run_scheduled_job) do |_input, _ctx|
  { "status" => "completed", "job" => "maintenance" }
end


def main
  # > Step 03 Run Worker
  worker = HATCHET.worker("scheduled-worker", workflows: [CRON_WF])
  worker.start
end

main if __FILE__ == $PROGRAM_NAME
