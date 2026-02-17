# frozen_string_literal: true

require_relative "worker"

# > Run a task
result = SIMPLE.run({ "message" => "Hello, World!" })
puts result
