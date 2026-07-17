# frozen_string_literal: true

require 'hatchet-sdk'
require_relative 'worker'

HATCHET = Hatchet::Client.new(debug: true) unless defined?(HATCHET)

# > trigger
first_ref = IDEMPOTENT_TASK.run_no_wait({ 'id' => '123' })

second_run_id = begin
  second_ref = IDEMPOTENT_TASK.run_no_wait({ 'id' => '123' })
  second_ref.workflow_run_id
rescue Hatchet::IdempotencyCollisionError => e
  puts "Run #{e.existing_run_external_id} already exists for this idempotency key"
  e.existing_run_external_id
end
# !!

puts "First run: #{first_ref.workflow_run_id}"
puts "Second run (or existing): #{second_run_id}"
