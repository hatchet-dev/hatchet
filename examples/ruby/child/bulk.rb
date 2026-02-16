# frozen_string_literal: true

require_relative "worker"

# > Bulk run a task
greetings = ["Hello, World!", "Hello, Moon!", "Hello, Mars!"]

results = CHILD_TASK_WF.run_many(
  greetings.map do |greeting|
    CHILD_TASK_WF.create_bulk_run_item(
      input: { "message" => greeting }
    )
  end
)

puts results
