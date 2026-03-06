import { ConcurrencyLimitStrategy } from '@hatchet/protoc/v1/workflows';
import { hatchet } from '../../hatchet-client';
import { callLlm, runTool } from './mock-agent';

// > Step 01 Define Agent Task
export const agentTask = hatchet.durableTask({
  name: 'reasoning-loop-agent',
  executionTimeout: '30m',
  concurrency: {
    expression: "input.session_id != null ? string(input.session_id) : 'constant'",
    maxRuns: 1,
    limitStrategy: ConcurrencyLimitStrategy.CANCEL_IN_PROGRESS,
  },
  fn: async (input) => {
    const query = (input as { query?: string })?.query ?? 'Hello';
    return agentReasoningLoop(query);
  },
});
// !!

// > Step 02 Reasoning Loop
async function agentReasoningLoop(query: string) {
  const messages: Array<{ role: string; content: string }> = [{ role: 'user', content: query }];
  for (let i = 0; i < 10; i++) {
    const resp = callLlm(messages);
    if (resp.done) return { response: resp.content };
    for (const tc of resp.toolCalls) {
      const result = runTool(tc.name, tc.args);
      messages.push({ role: 'tool', content: result });
    }
  }
  return { response: 'Max iterations reached' };
}
// !!

// > Step 03 Stream Response
export const streamingAgentTask = hatchet.durableTask({
  name: 'streaming-agent-task',
  executionTimeout: '30m',
  concurrency: {
    expression: "'constant'",
    maxRuns: 1,
    limitStrategy: ConcurrencyLimitStrategy.CANCEL_IN_PROGRESS,
  },
  fn: async (_, ctx) => {
    const tokens = ['Hello', ' ', 'world', '!'];
    for (const t of tokens) {
      ctx.putStream(t);
    }
    return { done: true };
  },
});
// !!
