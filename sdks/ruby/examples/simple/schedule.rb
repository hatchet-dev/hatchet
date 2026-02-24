# frozen_string_literal: true

require_relative "worker"

# > Schedule a task
schedule = SIMPLE.schedule(Time.now + 86_400, input: { "message" => "Hello, World!" })

## do something with the id
puts schedule.metadata.id
# !!
