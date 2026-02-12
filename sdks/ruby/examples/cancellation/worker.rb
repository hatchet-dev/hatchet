# frozen_string_literal: true

require "hatchet-sdk"

HATCHET = Hatchet::Client.new(debug: true)

CANCELLATION_WORKFLOW = hatchet.workflow(name: "CancelWorkflow")

# > Self-cancelling task
CANCELLATION_WORKFLOW.task(:self_cancel) do |input, ctx|
  sleep 2

  ## Cancel the task
  ctx.cancel

  sleep 10

  { "error" => "Task should have been cancelled" }
end

# > Checking exit flag
CANCELLATION_WORKFLOW.task(:check_flag) do |input, ctx|
  3.times do
    sleep 1

    # Note: Checking the status of the exit flag is mostly useful for cancelling
    # sync tasks without needing to forcibly kill the thread they're running on.
    if ctx.exit_flag
      puts "Task has been cancelled"
      raise "Task has been cancelled"
    end
  end

  { "error" => "Task should have been cancelled" }
end

def main
  worker = hatchet.worker("cancellation-worker", workflows: [CANCELLATION_WORKFLOW])
  worker.start
end

main if __FILE__ == $PROGRAM_NAME
