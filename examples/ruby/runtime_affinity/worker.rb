# frozen_string_literal: true

require "hatchet-sdk"
require "optparse"

HATCHET = Hatchet::Client.new(debug: true) unless defined?(HATCHET)

AFFINITY_EXAMPLE_TASK = HATCHET.task(name: :affinity_example_task) do |_input, ctx|
  { "worker_id" => ctx.worker_id }
end

if __FILE__ == $PROGRAM_NAME
  options = {}
  OptionParser.new do |opts|
    opts.on("--label LABEL", String, "Worker affinity label") { |v| options[:label] = v }
  end.parse!

  raise "Missing --label argument" unless options[:label]

  worker = HATCHET.worker(
    "runtime-affinity-worker",
    labels: { "affinity" => options[:label] },
    workflows: [AFFINITY_EXAMPLE_TASK]
  )

  worker.start
end
