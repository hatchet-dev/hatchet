# frozen_string_literal: true

require_relative "workflows/first_task"

# > Run a task
result = FIRST_TASK.run({ "message" => "Hello World!" })
puts "Finished running task: #{result['transformed_message']}"
# !!
