# frozen_string_literal: true

require "hatchet-sdk"

HATCHET = Hatchet::Client.new(debug: false)

ERROR_TEXT = "step1 failed"

# > OnFailure Step
# This workflow will fail because the step will throw an error
# we define an onFailure step to handle this case

ON_FAILURE_WF = HATCHET.workflow(name: "OnFailureWorkflow")

ON_FAILURE_WF.task(:step1, execution_timeout: 1) do |input, ctx|
  # This step will always raise an exception
  raise ERROR_TEXT
end

# After the workflow fails, this special step will run
ON_FAILURE_WF.on_failure_task do |input, ctx|
  # We can do things like perform cleanup logic
  # or notify a user here

  # Fetch the errors from upstream step runs from the context
  puts ctx.task_run_errors.inspect

  { "status" => "success" }
end


# > OnFailure With Details
# We can access the failure details in the onFailure step
# via the context method

ON_FAILURE_WF_WITH_DETAILS = HATCHET.workflow(name: "OnFailureWorkflowWithDetails")

DETAILS_STEP1 = ON_FAILURE_WF_WITH_DETAILS.task(:details_step1, execution_timeout: 1) do |input, ctx|
  raise ERROR_TEXT
end

# After the workflow fails, this special step will run
ON_FAILURE_WF_WITH_DETAILS.on_failure_task do |input, ctx|
  error = ctx.get_task_run_error(DETAILS_STEP1)

  unless error
    next { "status" => "unexpected success" }
  end

  # We can access the failure details here
  raise "Expected Hatchet::TaskRunError" unless error.is_a?(Hatchet::TaskRunError)

  if error.message.include?("step1 failed")
    next {
      "status" => "success",
      "failed_run_external_id" => error.task_run_external_id
    }
  end

  raise "unexpected failure"
end


def main
  worker = HATCHET.worker(
    "on-failure-worker",
    slots: 4,
    workflows: [ON_FAILURE_WF, ON_FAILURE_WF_WITH_DETAILS]
  )
  worker.start
end

main if __FILE__ == $PROGRAM_NAME
