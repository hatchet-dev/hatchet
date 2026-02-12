# frozen_string_literal: true

require "hatchet-sdk"

HATCHET = Hatchet::Client.new(debug: true)

# > Cron Workflow Definition
CRON_WORKFLOW = hatchet.workflow(
  name: "CronWorkflow",
  on_crons: ["*/5 * * * *"]
)

CRON_WORKFLOW.task(:cron_task) do |input, ctx|
  puts "Cron task executed at #{Time.now}"
  { "status" => "success" }
end

# > Programmatic Cron Creation
def create_cron
  hatchet.cron.create(
    workflow_name: "CronWorkflow",
    cron_name: "my-programmatic-cron",
    expression: "*/10 * * * *",
    input: { "message" => "hello from cron" }
  )
end

def main
  worker = hatchet.worker("cron-worker", workflows: [CRON_WORKFLOW])
  worker.start
end

main if __FILE__ == $PROGRAM_NAME
