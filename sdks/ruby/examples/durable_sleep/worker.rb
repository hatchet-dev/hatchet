# frozen_string_literal: true

require "hatchet-sdk"

HATCHET = Hatchet::Client.new(debug: true)

# > Durable Sleep
DURABLE_SLEEP_TASK = hatchet.durable_task(name: "DurableSleepTask") do |input, ctx|
  res = ctx.sleep_for(duration: 5)

  puts "got result #{res}"
end

def main
  worker = hatchet.worker("durable-sleep-worker", workflows: [DURABLE_SLEEP_TASK])
  worker.start
end

main if __FILE__ == $PROGRAM_NAME
