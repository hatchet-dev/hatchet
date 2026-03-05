// Third-party integration - requires: pnpm add groq-sdk
// See: /guides/ai-agents

import Groq from 'groq-sdk';

const client = new Groq();

// > Groq usage
export async function complete(messages: Array<{ role: string; content: string }>) {
  const r = await client.chat.completions.create({
    model: 'llama-3.3-70b-versatile',
    messages: messages as Groq.ChatCompletionMessageParam[],
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
