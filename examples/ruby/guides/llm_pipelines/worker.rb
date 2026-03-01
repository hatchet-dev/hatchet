# frozen_string_literal: true

require "hatchet-sdk"
require_relative "mock_llm"

HATCHET = Hatchet::Client.new(debug: true) unless defined?(HATCHET)

# > Step 01 Define Pipeline
LLM_WF = HATCHET.workflow(name: "LLMPipeline")

PROMPT_TASK = LLM_WF.task(:prompt_task) do |input, ctx|
  { "prompt" => input["prompt"] }
end


# > Step 02 Prompt Task
def build_prompt(user_input, context = "")
  base = "Process the following: #{user_input}"
  context.empty? ? base : "#{base}\nContext: #{context}"
end

# > Step 03 Validate Task
LLM_WF.task(:generate_task, parents: [PROMPT_TASK]) do |input, ctx|
  prev = ctx.task_output(PROMPT_TASK)
  output = generate(prev["prompt"])
  raise "Validation failed" unless output["valid"]
  output
end


def main
  # > Step 04 Run Worker
  worker = HATCHET.worker("llm-pipeline-worker", workflows: [LLM_WF])
  worker.start
end

main if __FILE__ == $PROGRAM_NAME
