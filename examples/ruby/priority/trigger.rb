# frozen_string_literal: true

require_relative "worker"

# > Runtime priority
PRIORITY_WORKFLOW.run_no_wait(
  {},
  options: Hatchet::TriggerWorkflowOptions.new(
    priority: 1,
    additional_metadata: { "priority" => "low", "key" => 1 },
  ),
)

PRIORITY_WORKFLOW.run_no_wait(
  {},
  options: Hatchet::TriggerWorkflowOptions.new(
    priority: 3,
    additional_metadata: { "priority" => "high", "key" => 1 },
  ),
)

# > Scheduled priority
PRIORITY_WORKFLOW.schedule(
  Time.now + 60,
  options: Hatchet::TriggerWorkflowOptions.new(priority: 3),
)

PRIORITY_WORKFLOW.create_cron(
  "my-scheduled-cron",
  "0 * * * *",
  input: {},
)
