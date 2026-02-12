# frozen_string_literal: true

# > Lifespan

require "hatchet-sdk"

HATCHET = Hatchet::Client.new(debug: true)

# Ruby lifespan uses a block with yield for setup/teardown
LIFESPAN_PROC = proc do
  { foo: "bar", pi: 3.14 }
end

LIFESPAN_TASK = HATCHET.task(name: "LifespanWorkflow") do |input, ctx|
  ctx.lifespan
end

def main
  worker = HATCHET.worker(
    "test-worker", slots: 1, workflows: [LIFESPAN_TASK], lifespan: LIFESPAN_PROC
  )
  worker.start
end

main if __FILE__ == $PROGRAM_NAME
