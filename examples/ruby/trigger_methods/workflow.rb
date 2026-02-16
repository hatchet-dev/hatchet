# frozen_string_literal: true

require "hatchet-sdk"

hatchet = Hatchet::Client.new

# > Define a task
SAY_HELLO = hatchet.task(name: "say_hello") do |input, ctx|
  { "greeting" => "Hello, #{input['name']}!" }
end

# > Sync
ref = SAY_HELLO.run_no_wait({ "name" => "World" })

# > Async
# In Ruby, run_no_wait is the equivalent of async enqueuing
ref = SAY_HELLO.run_no_wait({ "name" => "World" })

# > Result sync
result = ref.result

# > Result async
# In Ruby, result is synchronous - use poll for async-like behavior
result = ref.result
