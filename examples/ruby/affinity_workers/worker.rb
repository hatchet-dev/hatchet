# frozen_string_literal: true

require "hatchet-sdk"

HATCHET = Hatchet::Client.new(debug: true) unless defined?(HATCHET)

# > AffinityWorkflow
AFFINITY_WORKER_WORKFLOW = HATCHET.workflow(name: "AffinityWorkflow")


# > AffinityTask
AFFINITY_WORKER_WORKFLOW.task(
  :step,
  desired_worker_labels: {
    "model" => Hatchet::DesiredWorkerLabel.new(value: "fancy-ai-model-v2", weight: 10),
    "memory" => Hatchet::DesiredWorkerLabel.new(
      value: 256,
      required: true,
      comparator: :less_than
    )
  }
) do |input, ctx|
  if ctx.worker.labels["model"] != "fancy-ai-model-v2"
    ctx.worker.upsert_labels("model" => "unset")
    # DO WORK TO EVICT OLD MODEL / LOAD NEW MODEL
    ctx.worker.upsert_labels("model" => "fancy-ai-model-v2")
  end

  { "worker" => ctx.worker.id }
end


# > AffinityWorker
def main
  worker = HATCHET.worker(
    "affinity-worker",
    slots: 10,
    labels: {
      "model" => "fancy-ai-model-v2",
      "memory" => 512
    },
    workflows: [AFFINITY_WORKER_WORKFLOW]
  )
  worker.start
end


main if __FILE__ == $PROGRAM_NAME
