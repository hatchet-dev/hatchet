# frozen_string_literal: true

require "hatchet-sdk"
require_relative "mock_llm"

HATCHET = Hatchet::Client.new(debug: true) unless defined?(HATCHET)

GENERATOR_WF = HATCHET.workflow(name: "GenerateDraft")
EVALUATOR_WF = HATCHET.workflow(name: "EvaluateDraft")

# > Step 01 Define Tasks
GENERATOR_WF.task(:generate_draft) do |input, _ctx|
  prompt = if input["feedback"]
             "Improve this draft.\n\nDraft: #{input["previous_draft"]}\nFeedback: #{input["feedback"]}"
           else
             "Write a social media post about \"#{input["topic"]}\" for #{input["audience"]}. Under 100 words."
           end
  { "draft" => mock_generate(prompt) }
end

EVALUATOR_WF.task(:evaluate_draft) do |input, _ctx|
  mock_evaluate(input["draft"])
end
# !!

# > Step 02 Optimization Loop
OPTIMIZER_TASK = HATCHET.durable_task(name: "EvaluatorOptimizer", execution_timeout: "5m") do |input, _ctx|
  max_iterations = 3
  threshold = 0.8
  draft = ""
  feedback = ""

  max_iterations.times do |i|
    generated = GENERATOR_WF.run(
      "topic" => input["topic"], "audience" => input["audience"],
      "previous_draft" => draft.empty? ? nil : draft,
      "feedback" => feedback.empty? ? nil : feedback,
    )
    draft = generated["draft"]

    evaluation = EVALUATOR_WF.run(
      "draft" => draft, "topic" => input["topic"], "audience" => input["audience"],
    )

    next { "draft" => draft, "iterations" => i + 1, "score" => evaluation["score"] } if evaluation["score"] >= threshold

    feedback = evaluation["feedback"]
  end

  { "draft" => draft, "iterations" => max_iterations, "score" => -1 }
end
# !!

def main
  # > Step 03 Run Worker
  worker = HATCHET.worker("evaluator-optimizer-worker", slots: 5,
                                                        workflows: [GENERATOR_WF, EVALUATOR_WF, OPTIMIZER_TASK],)
  worker.start
  # !!
end

main if __FILE__ == $PROGRAM_NAME
