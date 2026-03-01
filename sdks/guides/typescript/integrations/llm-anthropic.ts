// Third-party integration - requires: pnpm add @anthropic-ai/sdk
// See: /guides/ai-agents

import Anthropic from '@anthropic-ai/sdk';

const client = new Anthropic();

// > Anthropic usage
export async function complete(messages: Array<{ role: string; content: string }>) {
  const resp = await client.messages.create({
    model: 'claude-3-5-haiku-20241022',
    max_tokens: 1024,
    messages: messages.map((m) => ({ role: m.role as 'user' | 'assistant', content: m.content })),
  });
  const toolUse = resp.content.find((b) => b.type === 'tool_use');
  if (toolUse && toolUse.type === 'tool_use') {
    return {
      content: '',
      toolCalls: [{ name: toolUse.name, args: toolUse.input }],
      done: false,
    };
  }
  const text = resp.content
    .filter((b): b is { type: 'text'; text: string } => b.type === 'text')
    .map((b) => b.text)
    .join('');
  return { content: text, toolCalls: [], done: true };
}
// !!
