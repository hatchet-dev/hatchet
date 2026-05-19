# frozen_string_literal: true

require 'hatchet-sdk'

HATCHET = Hatchet::Client.new(debug: true) unless defined?(HATCHET)

# > Step 02 Schedule One Time
# Schedule a one-time run at a specific time.
run_at = Time.now + 3600
HATCHET.scheduled.create(workflow_name: 'ScheduledWorkflow', trigger_at: run_at, input: {})
