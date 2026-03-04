# frozen_string_literal: true

# Third-party integration - requires: bundle add openai
# See: /guides/ai-agents

require 'openai'
require 'json'

OpenAI::Client.new

# > OpenAI usage
def complete(messages)
  response = client.chat(
    parameters: {
      model: 'gpt-4o-mini',
      messages: messages,
      tool_choice: 'auto',
      tools: [{
        type: 'function',
        function: {
          name: 'get_weather',
          description: 'Get weather for a location',
          parameters: { type: 'object', properties: { location: { type: 'string' } }, required: ['location'] }
        }
      }]
    }
  )
  msg = response.dig('choices', 0, 'message')
  tool_calls = msg['tool_calls']&.map do |tc|
    { 'name' => tc.dig('function', 'name'), 'args' => JSON.parse(tc.dig('function', 'arguments') || '{}') }
  end || []
  { 'content' => msg['content'] || '', 'tool_calls' => tool_calls, 'done' => tool_calls.empty? }
end
