frontend/docs/pages/v1/task-eviction.mdx# frozen_string_literal: true

require "hatchet-sdk"

HATCHET = Hatchet::Client.new(debug: true) unless defined?(HATCHET)

EVICTION_TTL_SECONDS = 5
LONG_SLEEP_SECONDS = 15
CAPACITY_SLEEP_SECONDS = 20
EVENT_KEY = "durable-eviction:event"

# > Eviction Policy
EVICTION_POLICY = Hatchet::EvictionPolicy.new(
  ttl: EVICTION_TTL_SECONDS,
  allow_capacity_eviction: true,
  priority: 0,
)
# !!

CAPACITY_EVICTION_POLICY = Hatchet::EvictionPolicy.new(
  ttl: nil,
  allow_capacity_eviction: true,
  priority: 0,
)

NON_EVICTABLE_POLICY = Hatchet::EvictionPolicy.new(
  ttl: nil,
  allow_capacity_eviction: false,
  priority: 0,
)

CHILD_TASK = HATCHET.task(name: "child_task", execution_timeout: 60) do |_input, _ctx|
  sleep LONG_SLEEP_SECONDS
  { "child_status" => "completed" }
end

BULK_CHILD_TASK = HATCHET.task(name: "bulk_child_task", execution_timeout: 60) do |input, _ctx|
  sleep_for = (input["sleep_for"] || 0).to_i
  sleep sleep_for
  { "sleep_for" => sleep_for, "status" => "completed" }
end

# > Evictable Sleep
EVICTABLE_SLEEP = HATCHET.durable_task(
  name: "evictable_sleep",
  execution_timeout: 300,
  eviction_policy: EVICTION_POLICY,
) do |_input, ctx|
  ctx.sleep_for(duration: LONG_SLEEP_SECONDS)
  { "status" => "completed" }
end
# !!

EVICTABLE_WAIT_FOR_EVENT = HATCHET.durable_task(
  name: "evictable_wait_for_event",
  execution_timeout: 300,
  eviction_policy: EVICTION_POLICY,
) do |_input, ctx|
  ctx.wait_for(
    EVENT_KEY,
    Hatchet::UserEventCondition.new(event_key: EVENT_KEY, expression: "true"),
  )
  { "status" => "completed" }
end

EVICTABLE_CHILD_SPAWN = HATCHET.durable_task(
  name: "evictable_child_spawn",
  execution_timeout: 300,
  eviction_policy: EVICTION_POLICY,
) do |_input, _ctx|
  child_result = CHILD_TASK.run
  { "child" => child_result, "status" => "completed" }
end

EVICTABLE_CHILD_BULK_SPAWN = HATCHET.durable_task(
  name: "evictable_child_bulk_spawn",
  execution_timeout: 300,
  eviction_policy: EVICTION_POLICY,
) do |_input, _ctx|
  items = Array.new(3) do |i|
    BULK_CHILD_TASK.create_bulk_run_item(
      input: { "sleep_for" => (EVICTION_TTL_SECONDS + 5) * (i + 1) },
      key: "child#{i}",
    )
  end
  child_results = BULK_CHILD_TASK.run_many(items)
  { "child_results" => child_results }
end

MULTIPLE_EVICTION = HATCHET.durable_task(
  name: "multiple_eviction",
  execution_timeout: 300,
  eviction_policy: EVICTION_POLICY,
) do |_input, ctx|
  ctx.sleep_for(duration: LONG_SLEEP_SECONDS)
  ctx.sleep_for(duration: LONG_SLEEP_SECONDS)
  { "status" => "completed" }
end

CAPACITY_EVICTABLE_SLEEP = HATCHET.durable_task(
  name: "capacity_evictable_sleep",
  execution_timeout: 300,
  eviction_policy: CAPACITY_EVICTION_POLICY,
) do |_input, ctx|
  ctx.sleep_for(duration: CAPACITY_SLEEP_SECONDS)
  { "status" => "completed" }
end

# > Non Evictable Sleep
NON_EVICTABLE_SLEEP = HATCHET.durable_task(
  name: "non_evictable_sleep",
  execution_timeout: 300,
  eviction_policy: NON_EVICTABLE_POLICY,
) do |_input, ctx|
  ctx.sleep_for(duration: 10)
  { "status" => "completed" }
end
# !!

def main
  worker = HATCHET.worker(
    "eviction-worker",
    workflows: [
      EVICTABLE_SLEEP,
      EVICTABLE_WAIT_FOR_EVENT,
      EVICTABLE_CHILD_SPAWN,
      EVICTABLE_CHILD_BULK_SPAWN,
      MULTIPLE_EVICTION,
      CAPACITY_EVICTABLE_SLEEP,
      NON_EVICTABLE_SLEEP,
      CHILD_TASK,
      BULK_CHILD_TASK,
    ],
  )
  worker.start
end

main if __FILE__ == $PROGRAM_NAME
