# frozen_string_literal: true

require "hatchet-sdk"

HATCHET = Hatchet::Client.new(debug: true) unless defined?(HATCHET)

# > Shared Concurrency Strategy
# A tenant-scoped strategy is shared across workflows: every task declaring the same
# name consumes the same concurrency limit. Re-registering the name updates it in place.
SHARED_LIMIT = Hatchet::ConcurrencyExpression.new(
  expression: "input.group",
  max_runs: 1,
  limit_strategy: :group_round_robin,
  name: "example-shared-limit",
  is_tenant_scoped: true
)

# two different workflows consuming one limit
SYNC_WORKFLOW = HATCHET.workflow(
  name: "SyncCrm",
  concurrency: SHARED_LIMIT
)

SYNC_WORKFLOW.task(:sync) do |input, ctx|
  puts "syncing crm"
  sleep 2
end

REPORT_WORKFLOW = HATCHET.workflow(
  name: "GenerateReport",
  concurrency: SHARED_LIMIT
)

REPORT_WORKFLOW.task(:report) do |input, ctx|
  puts "generating report"
  sleep 2
end


def main
  worker = HATCHET.worker(
    "concurrency-shared-worker",
    slots: 10,
    workflows: [SYNC_WORKFLOW, REPORT_WORKFLOW]
  )
  worker.start
end

main if __FILE__ == $PROGRAM_NAME
