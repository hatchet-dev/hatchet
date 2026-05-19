# frozen_string_literal: true

require "hatchet-sdk"

hatchet = Hatchet::Client.new

dynamic_cron_workflow = hatchet.workflow(name: "DynamicCronWorkflow")

# > Create
cron_trigger = dynamic_cron_workflow.create_cron(
  "customer-a-daily-report",
  "0 12 * * *",
  input: { "name" => "John Doe" }
)

id = cron_trigger.metadata.id

# > List
cron_triggers = hatchet.cron.list

# > Delete
hatchet.cron.delete(cron_trigger.metadata.id)
