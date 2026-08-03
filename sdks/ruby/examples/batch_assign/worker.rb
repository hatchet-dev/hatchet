# frozen_string_literal: true

require 'hatchet-sdk'
require 'securerandom'

HATCHET = Hatchet::Client.new(debug: true) unless defined?(HATCHET)

# Preview: batch tasks are in beta and may change in future releases.

# > Declaring a batch task
BATCH_SIMPLE = HATCHET.batch_task(
  name: 'ruby-e2e-batch-simple',
  batch: Hatchet::BatchTaskConfig.new(max_size: 3, max_interval_ms: 200)
) do |inputs, _ctx|
  inputs.transform_values { |input| { 'transformed_message' => input['message'].upcase } }
end
# !!

# > Declaring a keyed batch task
BATCH_KEYED = HATCHET.batch_task(
  name: 'ruby-e2e-batch-keyed',
  batch: Hatchet::BatchTaskConfig.new(max_size: 2, max_interval_ms: 200, group_key: 'input.group')
) do |inputs, _ctx|
  unique_keys = inputs.values.map { |i| i['group'] }.uniq.length
  batch_size = inputs.length

  inputs.transform_values do |input|
    {
      'batch_key' => input['group'],
      'batch_size' => batch_size,
      'unique_keys' => unique_keys,
      'uppercase' => input['message'].upcase,
    }
  end
end
# !!

BATCH_KEYED_FAILABLE = HATCHET.batch_task(
  name: 'ruby-e2e-batch-keyed-failable',
  batch: Hatchet::BatchTaskConfig.new(max_size: 2, max_interval_ms: 200, group_key: 'input.group')
) do |inputs, _ctx|
  inputs.transform_values { |input| { 'uppercase' => input['message'].upcase } }
end

BATCH_KEYED_INTERVAL = HATCHET.batch_task(
  name: 'ruby-e2e-batch-keyed-interval',
  batch: Hatchet::BatchTaskConfig.new(max_size: 3, max_interval_ms: 150, group_key: 'input.group')
) do |inputs, _ctx|
  unique_keys = inputs.values.map { |i| i['group'] }.uniq.length
  batch_size = inputs.length

  inputs.transform_values do |input|
    {
      'batch_key' => input['group'],
      'batch_size' => batch_size,
      'unique_keys' => unique_keys,
      'uppercase' => input['message'].upcase,
    }
  end
end

BATCH_LARGE = HATCHET.batch_task(
  name: 'ruby-e2e-batch-large',
  batch: Hatchet::BatchTaskConfig.new(max_size: 100, max_interval_ms: 10_000)
) do |inputs, _ctx|
  batch_id = SecureRandom.uuid
  batch_size = inputs.length

  inputs.transform_values do |input|
    {
      'batch_id' => batch_id,
      'received' => true,
      'batch_size' => batch_size,
      'data_length' => input['data'].length,
    }
  end
end

BATCH_SINGLE = HATCHET.batch_task(
  name: 'ruby-e2e-batch-single',
  batch: Hatchet::BatchTaskConfig.new(max_size: 1, max_interval_ms: 100)
) do |inputs, _ctx|
  batch_size = inputs.length
  inputs.transform_values { |input| { 'original' => input['message'], 'batch_size' => batch_size } }
end

BATCH_ORDERED = HATCHET.batch_task(
  name: 'ruby-e2e-batch-ordered',
  batch: Hatchet::BatchTaskConfig.new(max_size: 20, max_interval_ms: 2_000)
) do |inputs, _ctx|
  inputs.transform_values { |input| { 'index' => input['index'] } }
end

# > Declaring a broadcast batch task
BATCH_BROADCAST = HATCHET.batch_task(
  name: 'ruby-e2e-batch-broadcast',
  batch: Hatchet::BatchTaskConfig.new(max_size: 10, max_interval_ms: 2_000, broadcast_output: true)
) do |inputs, _ctx|
  { 'sum' => inputs.values.sum { |i| i['message'].length } }
end
# !!

BATCH_CANCEL = HATCHET.batch_task(
  name: 'ruby-e2e-batch-cancel',
  batch: Hatchet::BatchTaskConfig.new(max_size: 10, max_interval_ms: 2_000, broadcast_output: true)
) do |_inputs, ctx|
  ctx.cancel
  {}
end

CHILD = HATCHET.task(name: 'ruby-e2e-batch-child') do |input, _ctx|
  { 'message_len' => input['message'].length }
end

CHILD_BATCH = HATCHET.batch_task(
  name: 'ruby-e2e-batch-child-batch',
  batch: Hatchet::BatchTaskConfig.new(max_size: 10, max_interval_ms: 60_000, broadcast_output: true)
) do |inputs, _ctx|
  { 'out' => inputs }
end

BATCH_CHILD_SPAWN = HATCHET.batch_task(
  name: 'ruby-e2e-batch-child-spawn',
  batch: Hatchet::BatchTaskConfig.new(max_size: 10, max_interval_ms: 60_000),
  execution_timeout: 60
) do |inputs, _ctx|
  inputs.each_with_object({}) do |(id, _input), out|
    out[id] = CHILD.run({ 'message' => 'blahblah' })
  end
end

BATCH_CHILD_BATCH_SPAWN = HATCHET.batch_task(
  name: 'ruby-e2e-batch-child-batch-spawn',
  batch: Hatchet::BatchTaskConfig.new(max_size: 10, max_interval_ms: 60_000),
  execution_timeout: 60
) do |inputs, _ctx|
  results = {}
  mutex = Mutex.new

  threads = inputs.keys.map do |id|
    Thread.new do
      result = CHILD_BATCH.run({ 'message' => 'hello' })
      mutex.synchronize { results[id] = result }
    end
  end
  threads.each(&:join)

  results
end

def main
  worker = HATCHET.worker(
    'batch-assign-worker',
    workflows: [
      BATCH_SIMPLE,
      BATCH_KEYED,
      BATCH_KEYED_FAILABLE,
      BATCH_KEYED_INTERVAL,
      BATCH_LARGE,
      BATCH_SINGLE,
      BATCH_ORDERED,
      BATCH_BROADCAST,
      BATCH_CANCEL,
      CHILD,
      CHILD_BATCH,
      BATCH_CHILD_SPAWN,
      BATCH_CHILD_BATCH_SPAWN,
    ],
    slots: 25
  )
  worker.start
end

main if __FILE__ == $PROGRAM_NAME
