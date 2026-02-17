# frozen_string_literal: true

require "hatchet-sdk"

hatchet = Hatchet::Client.new

# > Create
scheduled_run = hatchet.scheduled.create(
  workflow_name: "simple-workflow",
  trigger_at: Time.now + 10,
  input: { "data" => "simple-workflow-data" },
  additional_metadata: { "customer_id" => "customer-a" }
)

id = scheduled_run.metadata.id

# > Reschedule
hatchet.scheduled.update(
  scheduled_run.metadata.id,
  trigger_at: Time.now + 3600
)

# > Delete
hatchet.scheduled.delete(scheduled_run.metadata.id)

# > List
scheduled_runs = hatchet.scheduled.list

# > Bulk delete
hatchet.scheduled.bulk_delete(scheduled_ids: [id])

# > Bulk reschedule
hatchet.scheduled.bulk_update(
  [[id, Time.now + 7200]]
)
