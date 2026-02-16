# frozen_string_literal: true

require_relative "worker"

# > Running a task
result = CHILD_TASK_WF.run({ "message" => "Hello, World!" })
# !!

# > Running a task aio
# In Ruby, run is synchronous
result = CHILD_TASK_WF.run({ "message" => "Hello, World!" })
# !!

# > Running multiple tasks
results = CHILD_TASK_WF.run_many(
  [
    CHILD_TASK_WF.create_bulk_run_item(input: { "message" => "Hello, World!" }),
    CHILD_TASK_WF.create_bulk_run_item(input: { "message" => "Hello, Moon!" })
  ]
)
puts results
# !!
