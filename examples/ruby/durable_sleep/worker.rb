# frozen_string_literal: true

require "hatchet-sdk"

HATCHET = Hatchet::Client.new(debug: true) unless defined?(HATCHET)

# > Durable Sleep
DURABLE_SLEEP_TASK = HATCHET.durable_task(name: "DurableSleepTask") do |input, ctx|
  res = ctx.sleep_for(duration: 5)

  puts "got result #{res}"
end


def main
  worker = HATCHET.worker("durable-sleep-worker", workflows: [DURABLE_SLEEP_TASK])
  worker.start
end

main if __FILE__ == $PROGRAM_NAME
