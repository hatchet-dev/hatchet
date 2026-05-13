# frozen_string_literal: true

require "hatchet-sdk"

HATCHET = Hatchet::Client.new(debug: true) unless defined?(HATCHET)

# > Define a DAG
DAG_WORKFLOW = HATCHET.workflow(name: "DAGWorkflow")

# !!

# > First task
STEP1 = DAG_WORKFLOW.task(:step1, execution_timeout: 5) do |input, ctx|
  { "random_number" => rand(1..100) }
end

STEP2 = DAG_WORKFLOW.task(:step2, execution_timeout: 5) do |input, ctx|
  { "random_number" => rand(1..100) }
end

# !!

# > Task with parents
DAG_WORKFLOW.task(:step3, parents: [STEP1, STEP2]) do |input, ctx|
  one = ctx.task_output(STEP1)["random_number"]
  two = ctx.task_output(STEP2)["random_number"]

  { "sum" => one + two }
end

DAG_WORKFLOW.task(:step4, parents: [STEP1, :step3]) do |input, ctx|
  puts(
    "executed step4",
    Time.now.strftime("%H:%M:%S"),
    input.inspect,
    ctx.task_output(STEP1).inspect,
    ctx.task_output(:step3).inspect
  )

  { "step4" => "step4" }
end

# !!

# > Declare a worker
def main
  worker = HATCHET.worker("dag-worker", workflows: [DAG_WORKFLOW])
  worker.start
end

# !!

main if __FILE__ == $PROGRAM_NAME
