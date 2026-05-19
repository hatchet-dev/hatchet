# frozen_string_literal: true

require "hatchet-sdk"

# > Setup
hatchet = Hatchet::Client.new

workflows = hatchet.workflows.list

workflow = workflows.rows.first
# !!

# > List runs
workflow_runs = hatchet.runs.list(workflow_ids: [workflow.metadata.id])
# !!

# > Cancel by run ids
workflow_run_ids = workflow_runs.rows.map { |run| run.metadata.id }

hatchet.runs.bulk_cancel(ids: workflow_run_ids)
# !!

# > Cancel by filters
hatchet.runs.bulk_cancel(
  since: Time.now - 86_400,
  until_time: Time.now,
  statuses: ["RUNNING"],
  workflow_ids: [workflow.metadata.id],
  additional_metadata: { "key" => "value" }
)
# !!
