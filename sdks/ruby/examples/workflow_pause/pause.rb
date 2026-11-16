# frozen_string_literal: true

require "hatchet-sdk"

hatchet = Hatchet::Client.new

pausable_workflow = hatchet.workflow(name: "PausableWorkflow")

# > Pause a workflow
pausable_workflow.pause(
  queue_ttl: "24h",
  paused_workflow_cron_run_queue_behavior: "DROP",
  paused_workflow_scheduled_run_queue_behavior: "QUEUE"
)
# !!

# > Unpause a workflow
pausable_workflow.unpause
# !!
