# frozen_string_literal: true

require 'hatchet-sdk'
require_relative 'mock_llm'

HATCHET = Hatchet::Client.new(debug: true) unless defined?(HATCHET)

# > Step 01 Specialist Agents
RESEARCH_TASK = HATCHET.durable_task(name: 'ResearchSpecialist', execution_timeout: '3m') do |input, _ctx|
  { 'result' => mock_specialist_llm(input['task'], 'research') }
end

WRITING_TASK = HATCHET.durable_task(name: 'WritingSpecialist', execution_timeout: '2m') do |input, _ctx|
  { 'result' => mock_specialist_llm(input['task'], 'writing') }
end

CODE_TASK = HATCHET.durable_task(name: 'CodeSpecialist', execution_timeout: '2m') do |input, _ctx|
  { 'result' => mock_specialist_llm(input['task'], 'code') }
end
# !!

SPECIALISTS = {
  'research' => RESEARCH_TASK,
  'writing' => WRITING_TASK,
  'code' => CODE_TASK
}.freeze

# > Step 02 Orchestrator Loop
ORCHESTRATOR = HATCHET.durable_task(name: 'MultiAgentOrchestrator', execution_timeout: '15m') do |input, _ctx|
  messages = [{ 'role' => 'user', 'content' => input['goal'] }]

  result = nil
  10.times do
    response = mock_orchestrator_llm(messages)

    if response['done']
      result = { 'result' => response['content'] }
      break
    end

    specialist = SPECIALISTS[response['tool_call']['name']]
    raise "Unknown specialist: #{response['tool_call']['name']}" unless specialist

    specialist_result = specialist.run(
      'task' => response['tool_call']['args']['task'],
      'context' => messages.map { |m| m['content'] }.join("\n")
    )

    messages << { 'role' => 'assistant', 'content' => "Called #{response['tool_call']['name']}" }
    messages << { 'role' => 'tool', 'content' => specialist_result['result'] }
  end

  result || { 'result' => 'Max iterations reached' }
end
# !!

def main
  # > Step 03 Run Worker
  worker = HATCHET.worker('multi-agent-worker', slots: 10, workflows: [RESEARCH_TASK, WRITING_TASK, CODE_TASK, ORCHESTRATOR])
  worker.start
  # !!
end

main if __FILE__ == $PROGRAM_NAME
