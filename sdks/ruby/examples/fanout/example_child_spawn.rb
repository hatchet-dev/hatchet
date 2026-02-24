# frozen_string_literal: true

require_relative "worker"

# > Child spawn
FANOUT_CHILD_WF.run({ "a" => "b" })
# !!

# > Error handling
begin
  FANOUT_CHILD_WF.run({ "a" => "b" })
rescue StandardError => e
  puts "Child workflow failed: #{e.message}"
end
# !!
