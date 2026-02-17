# frozen_string_literal: true

require "hatchet-sdk"

HATCHET = Hatchet::Client.new(debug: true) unless defined?(HATCHET)

# > ScheduleTimeout
TIMEOUT_WF = HATCHET.workflow(
  name: "TimeoutWorkflow",
  task_defaults: { execution_timeout: 120 } # 2 minutes
)


# > ExecutionTimeout
# Specify an execution timeout on a task
TIMEOUT_WF.task(:timeout_task, execution_timeout: 5, schedule_timeout: 600) do |input, ctx|
  sleep 30
  { "status" => "success" }
end

REFRESH_TIMEOUT_WF = HATCHET.workflow(name: "RefreshTimeoutWorkflow")


# > RefreshTimeout
REFRESH_TIMEOUT_WF.task(:refresh_task, execution_timeout: 4) do |input, ctx|
  ctx.refresh_timeout(10)
  sleep 5

  { "status" => "success" }
end


def main
  worker = HATCHET.worker(
    "timeout-worker", slots: 4, workflows: [TIMEOUT_WF, REFRESH_TIMEOUT_WF]
  )
  worker.start
end

main if __FILE__ == $PROGRAM_NAME
