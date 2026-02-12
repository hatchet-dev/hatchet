# frozen_string_literal: true

require "hatchet-sdk"

HATCHET = Hatchet::Client.new(debug: true)

RUN_DETAIL_TEST_WORKFLOW = HATCHET.workflow(name: "RunDetailTest")

DETAIL_STEP1 = RUN_DETAIL_TEST_WORKFLOW.task(:step1) do |input, ctx|
  { "random_number" => rand(1..100) }
end

RUN_DETAIL_TEST_WORKFLOW.task(:cancel_step) do |input, ctx|
  ctx.cancel
  10.times { sleep 1 }
end

RUN_DETAIL_TEST_WORKFLOW.task(:fail_step) do |input, ctx|
  raise "Intentional Failure"
end

DETAIL_STEP2 = RUN_DETAIL_TEST_WORKFLOW.task(:step2) do |input, ctx|
  sleep 5
  { "random_number" => rand(1..100) }
end

RUN_DETAIL_TEST_WORKFLOW.task(:step3, parents: [DETAIL_STEP1, DETAIL_STEP2]) do |input, ctx|
  one = ctx.task_output(DETAIL_STEP1)["random_number"]
  two = ctx.task_output(DETAIL_STEP2)["random_number"]

  { "sum" => one + two }
end

RUN_DETAIL_TEST_WORKFLOW.task(:step4, parents: [DETAIL_STEP1, :step3]) do |input, ctx|
  puts(
    "executed step4",
    Time.now.strftime("%H:%M:%S"),
    input.inspect,
    ctx.task_output(DETAIL_STEP1).inspect,
    ctx.task_output(:step3).inspect
  )

  { "step4" => "step4" }
end

def main
  worker = HATCHET.worker("run-detail-worker", workflows: [RUN_DETAIL_TEST_WORKFLOW])
  worker.start
end

main if __FILE__ == $PROGRAM_NAME
