# frozen_string_literal: true

require "hatchet-sdk"

HATCHET = Hatchet::Client.new(debug: true) unless defined?(HATCHET)

EVICTION_TTL_SECONDS = 5
LONG_SLEEP_SECONDS = 15
EVENT_KEY = "durable-eviction:event"

# > Eviction Policy
EVICTION_POLICY = Hatchet::EvictionPolicy.new(
  ttl: EVICTION_TTL_SECONDS,
  allow_capacity_eviction: true,
  priority: 0,
)
# !!

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

# > Non Evictable Sleep
NON_EVICTABLE_SLEEP = HATCHET.durable_task(
  name: "non_evictable_sleep",
  execution_timeout: 300,
  eviction_policy: Hatchet::EvictionPolicy.new(
    ttl: nil,
    allow_capacity_eviction: false,
    priority: 0,
  ),
) do |_input, ctx|
  ctx.sleep_for(duration: 10)
  { "status" => "completed" }
end
# !!

def main
  worker = HATCHET.worker(
    "eviction-worker",
    workflows: [EVICTABLE_SLEEP, NON_EVICTABLE_SLEEP],
  )
  worker.start
end

main if __FILE__ == $PROGRAM_NAME
