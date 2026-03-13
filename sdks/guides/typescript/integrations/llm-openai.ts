// Third-party integration - requires: pnpm add openai
// See: /guides/ai-agents

import OpenAI from 'openai';

const client = new OpenAI();

// > OpenAI usage
export async function complete(messages: Array<{ role: string; content: string }>) {
  const r = await client.chat.completions.create({
    model: 'gpt-4o-mini',
    messages: messages as OpenAI.ChatCompletionMessageParam[],
    tool_choice: 'auto',
    tools: [
      {
        type: 'function',
        function: {
          name: 'get_weather',
          description: 'Get weather for a location',
          parameters: {
            type: 'object',
            properties: { location: { type: 'string' } },
            required: ['location'],
          },
        },
      },
    ],
  });
  const msg = r.choices[0]?.message;
  const toolCalls = (msg?.tool_calls ?? []).map((tc) => ({
    name: tc.function?.name ?? '',
    args: JSON.parse(tc.function?.arguments ?? '{}'),
  }));
  return {
    content: msg?.content ?? '',
    toolCalls,
    done: toolCalls.length === 0,
  };
}
// !!
