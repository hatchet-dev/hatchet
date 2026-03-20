# frozen_string_literal: true

require_relative "worker"

# > Cancelling a run
ref = CANCELLATION_WORKFLOW.run_no_wait

sleep 5

HATCHET.runs.cancel(ref.workflow_run_id)
# !!
