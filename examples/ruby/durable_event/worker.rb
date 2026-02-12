# frozen_string_literal: true

require "hatchet-sdk"

HATCHET = Hatchet::Client.new(debug: true)

DURABLE_EVENT_TASK_KEY = "user:update"

# > Durable Event
DURABLE_EVENT_TASK = HATCHET.durable_task(name: "DurableEventTask") do |input, ctx|
  res = ctx.wait_for(
    "event",
    Hatchet::UserEventCondition.new(event_key: "user:update")
  )

  puts "got event #{res}"
end

DURABLE_EVENT_TASK_WITH_FILTER = HATCHET.durable_task(name: "DurableEventWithFilterTask") do |input, ctx|

  # > Durable Event With Filter
  res = ctx.wait_for(
    "event",
    Hatchet::UserEventCondition.new(
      event_key: "user:update",
      expression: "input.user_id == '1234'"
    )
  )

  puts "got event #{res}"
end


def main
  worker = HATCHET.worker(
    "durable-event-worker",
    workflows: [DURABLE_EVENT_TASK, DURABLE_EVENT_TASK_WITH_FILTER]
  )
  worker.start
end

main if __FILE__ == $PROGRAM_NAME
