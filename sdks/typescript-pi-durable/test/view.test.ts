import type { ConversationView } from '@earendil-works/pi-durable';
import { describe, expect, it } from 'vitest';
import { summarize } from '../src/view.js';

// Only the fields summarize reads.
function view(partial: {
  entries?: unknown[];
  docs?: Record<string, unknown>;
  parent?: unknown;
  before?: number;
}): ConversationView {
  return {
    conversation: { id: 1, parent: partial.parent },
    entries: partial.entries ?? [],
    before: partial.before,
    docs: partial.docs ?? {},
  } as unknown as ConversationView;
}

describe('summarize', () => {
  it('turns entries into items', () => {
    const summary = summarize(
      view({
        entries: [
          { id: 1, kind: 'pi.message', model: [{ role: 'user', content: 'fix the test' }] },
          {
            id: 2,
            kind: 'pi.message',
            model: [
              {
                role: 'assistant',
                model: 'm',
                content: [
                  { type: 'thinking', thinking: 'look first' },
                  { type: 'text', text: 'reading' },
                  { type: 'toolCall', id: 'c1', name: 'read', arguments: { path: 'a.ts' } },
                ],
              },
            ],
          },
          {
            id: 3,
            kind: 'pi.message',
            model: [
              {
                role: 'toolResult',
                toolCallId: 'c1',
                toolName: 'read',
                isError: false,
                content: [{ type: 'text', text: 'x' }],
              },
            ],
          },
          { id: 4, kind: 'pi.compaction', model: [] },
          { id: 5, kind: 'pi.system', model: [{ role: 'system', content: 'prompt' }] },
        ],
      }),
      { owner: 1, sandboxId: 'sb', snapshotId: 'snap' }
    );

    expect(summary.items).toEqual([
      { id: 1, kind: 'user', text: 'fix the test' },
      {
        id: 2,
        kind: 'assistant',
        text: 'reading',
        thinking: 'look first',
        toolCalls: [{ id: 'c1', name: 'read', arguments: { path: 'a.ts' } }],
        model: 'm',
      },
      { id: 3, kind: 'tool', callId: 'c1', name: 'read', isError: false, output: 'x' },
      { id: 4, kind: 'note', text: 'Older messages were compacted.' },
    ]);
    expect(summary).toMatchObject({
      id: 1,
      sandboxId: 'sb',
      snapshotId: 'snap',
      busy: false,
      inbox: 0,
    });
    expect(summary.live).toBeUndefined();
  });

  it('adds up usage across models', () => {
    const usage = (n: number) => ({
      input: n,
      output: 2 * n,
      cacheRead: 3 * n,
      cost: { total: n / 10 },
    });
    const summary = summarize(
      view({ docs: { 'pi.usage': { models: { a: usage(1), b: usage(2) } } } }),
      {}
    );
    expect(summary.usage).toEqual({ input: 3, output: 6, cacheRead: 9, cost: expect.closeTo(0.3) });
  });

  it('shows the live run, the inbox and the parent', () => {
    const summary = summarize(
      view({
        parent: { conversationId: 0, at: 7 },
        docs: {
          'pi.inbox': { items: [{}, {}] },
          'pi.live': {
            run: {},
            generation: {
              message: { content: [{ type: 'text', text: 'half' }] },
              retry: { error: 'overloaded' },
            },
            tools: [{ callId: 'c', name: 'bash', status: 'running', output: 'ok' }],
          },
        },
      }),
      {}
    );
    expect(summary.parent).toEqual({ conversationId: 0, entryId: 7 });
    expect(summary.busy).toBe(true);
    expect(summary.inbox).toBe(2);
    expect(summary.live).toEqual({
      text: 'half',
      thinking: '',
      retry: 'overloaded',
      tools: [{ callId: 'c', name: 'bash', status: 'running', output: 'ok' }],
    });
  });

  it('clips long text', () => {
    const text = 'x'.repeat(5_000);
    const [item] = summarize(
      view({ entries: [{ id: 1, kind: 'pi.message', model: [{ role: 'user', content: text }] }] }),
      {}
    ).items;
    expect(item).toMatchObject({
      kind: 'user',
      text: `${'x'.repeat(4_000)}\n… 1000 more characters`,
    });
  });

  it('says where the window starts and why the last run failed', () => {
    const summary = summarize(
      view({
        before: 41,
        docs: { 'pi.live': { runFailure: { taskId: 3, inputs: [], reason: 'aborted' } } },
      }),
      {}
    );
    expect(summary.before).toBe(41);
    expect(summary.failure).toEqual({ reason: 'aborted', detail: undefined });
    expect(summary.busy).toBe(false);
  });
});
