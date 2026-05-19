# frozen_string_literal: true

require "hatchet-sdk"

HATCHET = Hatchet::Client.new unless defined?(HATCHET)

# > Event trigger
EVENT_KEY = "user:create"
SECONDARY_KEY = "foobarbaz"
WILDCARD_KEY = "subscription:*"

EVENT_WORKFLOW = HATCHET.workflow(
  name: "EventWorkflow",
  on_events: [EVENT_KEY, SECONDARY_KEY, WILDCARD_KEY]
)


# > Event trigger with filter
EVENT_WORKFLOW_WITH_FILTER = HATCHET.workflow(
  name: "EventWorkflow",
  on_events: [EVENT_KEY, SECONDARY_KEY, WILDCARD_KEY],
  default_filters: [
    Hatchet::DefaultFilter.new(
      expression: "true",
      scope: "example-scope",
      payload: {
        "main_character" => "Anna",
        "supporting_character" => "Stiva",
        "location" => "Moscow"
      }
    )
  ]
)

EVENT_WORKFLOW.task(:task) do |input, ctx|
  puts "event received"
  ctx.filter_payload
end


# > Accessing the filter payload
EVENT_WORKFLOW_WITH_FILTER.task(:filtered_task) do |input, ctx|
  puts ctx.filter_payload.inspect
end


def main
  worker = HATCHET.worker(name: "EventWorker", workflows: [EVENT_WORKFLOW])
  worker.start
end

main if __FILE__ == $PROGRAM_NAME
