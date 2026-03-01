# frozen_string_literal: true

require "hatchet-sdk"
require_relative "mock_llm"

HATCHET = Hatchet::Client.new(debug: true) unless defined?(HATCHET)

RESEARCH_WF = HATCHET.workflow(name: "ResearchSpecialist")
WRITING_WF = HATCHET.workflow(name: "WritingSpecialist")
CODE_WF = HATCHET.workflow(name: "CodeSpecialist")

# > Step 01 Specialist Agents
RESEARCH_WF.task(:research) do |input, ctx|
  { "result" => mock_specialist_llm(input["task"], "research") }
end

WRITING_WF.task(:write) do |input, ctx|
  { "result" => mock_specialist_llm(input["task"], "writing") }
end

CODE_WF.task(:code) do |input, ctx|
  { "result" => mock_specialist_llm(input["task"], "code") }
end

SPECIALISTS = {
  "research" => RESEARCH_WF,
  "writing" => WRITING_WF,
  "code" => CODE_WF
}.freeze

# > Step 02 Orchestrator Loop
ORCHESTRATOR = HATCHET.durable_task(name: "MultiAgentOrchestrator", execution_timeout: "15m") do |input, ctx|
  messages = [{ "role" => "user", "content" => input["goal"] }]

  result = nil
  10.times do
    response = mock_orchestrator_llm(messages)

    if response["done"]
      result = { "result" => response["content"] }
      break
    end

    specialist_wf = SPECIALISTS[response["tool_call"]["name"]]
    raise "Unknown specialist: #{response['tool_call']['name']}" unless specialist_wf

    specialist_result = specialist_wf.run(
      "task" => response["tool_call"]["args"]["task"],
      "context" => messages.map { |m| m["content"] }.join("\n")
    )

    messages << { "role" => "assistant", "content" => "Called #{response['tool_call']['name']}" }
    messages << { "role" => "tool", "content" => specialist_result["result"] }
  end

  result || { "result" => "Max iterations reached" }
end

def main
  # > Step 03 Run Worker
  worker = HATCHET.worker("multi-agent-worker", slots: 10, workflows: [RESEARCH_WF, WRITING_WF, CODE_WF, ORCHESTRATOR])
  worker.start
end

main if __FILE__ == $PROGRAM_NAME
