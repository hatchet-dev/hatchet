# frozen_string_literal: true

require 'hatchet-sdk'
require_relative 'mock_agent'

HATCHET = Hatchet::Client.new(debug: true) unless defined?(HATCHET)

# > Step 02 Reasoning Loop
def agent_reasoning_loop(query)
  messages = [{ 'role' => 'user', 'content' => query }]
  10.times do
    resp = call_llm(messages)
    return { 'response' => resp['content'] } if resp['done']

    (resp['tool_calls'] || []).each do |tc|
      result = run_tool(tc['name'], tc['args'] || {})
      messages << { 'role' => 'tool', 'content' => result }
    end
  end
  { 'response' => 'Max iterations reached' }
end
# !!

# > Step 01 Define Agent Task
AGENT_TASK = HATCHET.durable_task(name: 'ReasoningLoopAgent') do |input, _ctx|
  query = input.is_a?(Hash) && input['query'] ? input['query'].to_s : 'Hello'
  agent_reasoning_loop(query)
end
# !!

# > Step 03 Stream Response
STREAMING_AGENT = HATCHET.durable_task(name: 'StreamingAgentTask') do |_input, ctx|
  %w[Hello \s world !].each { |t| ctx.put_stream(t) }
  { 'done' => true }
end

# !!

def main
  # > Step 04 Run Worker
  worker = HATCHET.worker('agent-worker', slots: 5, workflows: [AGENT_TASK, STREAMING_AGENT])
  worker.start
  # !!
end

main if __FILE__ == $PROGRAM_NAME
