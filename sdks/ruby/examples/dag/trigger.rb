# frozen_string_literal: true

require_relative "worker"

# > Trigger the DAG
result = DAG_WORKFLOW.run
puts result
# !!
