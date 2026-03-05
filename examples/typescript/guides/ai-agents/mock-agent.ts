/** Mock LLM and tools - no external API dependencies */

let callCount = 0;

export interface LLMResponse {
  content: string;
  toolCalls: Array<{ name: string; args: Record<string, unknown> }>;
  done: boolean;
}

export function callLlm(messages: Array<{ role: string; content: string }>): LLMResponse {
  callCount += 1;
  if (callCount === 1) {
    return {
      content: '',
      toolCalls: [{ name: 'get_weather', args: { location: 'SF' } }],
      done: false,
    };
  }
  return { content: "It's 72°F and sunny in SF.", toolCalls: [], done: true };
}

export function runTool(name: string, args: Record<string, unknown>): string {
  if (name === 'get_weather') {
    const loc = String(args?.location ?? 'unknown');
    return `Weather in ${loc}: 72°F, sunny`;
  }
  return `Unknown tool: ${name}`;
}
