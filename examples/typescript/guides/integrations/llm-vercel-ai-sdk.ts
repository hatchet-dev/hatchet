// Third-party integration - requires: pnpm add ai @ai-sdk/openai
// See: /guides/ai-agents
// Vercel AI SDK: unified interface for OpenAI, Anthropic, Google, etc.

import { generateText, tool } from 'ai';
import { openai } from '@ai-sdk/openai';
import { z } from 'zod';

// > Vercel AI SDK usage
export async function complete(messages: Array<{ role: string; content: string }>) {
  const tools = {
    get_weather: tool({
      description: 'Get weather for a location',
      parameters: z.object({ location: z.string() }),
      execute: async ({ location }) => `Weather in ${location}: 72°F, sunny`,
    }),
  };
  const { text, toolCalls } = await generateText({
    model: openai('gpt-4o-mini'),
    messages: messages.map((m) => ({
      role: m.role as 'user' | 'assistant' | 'system',
      content: m.content,
    })),
    maxSteps: 5, // SDK runs tool loop internally
    tools,
  });
  return {
    content: text,
    tool_calls: toolCalls.map((tc) => ({ name: tc.toolName, args: tc.args })),
    done: true, // maxSteps handles full agent loop
  };
}
