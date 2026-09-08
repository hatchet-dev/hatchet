import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import { NonRetryableError, hatchet, type Duration } from '../../index';
import { createTestOperator, type DurableResult } from '../../testing';

const secret = 'test-secret-at-least-32-characters-long';
const quiet = { debug: vi.fn(), info: vi.fn(), warn: vi.fn(), error: vi.fn() };

const echo = hatchet.task({
  name: 'echo',
  fn: async (input: { message: string }) => ({ echo: input.message }),
});

const sleeper = hatchet.durableTask({
  name: 'sleep-then-echo',
  fn: async (input: { message: string; sleep?: string }, ctx) => {
    const startedAt = await ctx.now();
    await ctx.sleepFor((input.sleep ?? '3s') as Duration);
    return {
      echo: input.message,
      startedAt: startedAt.toISOString(),
      invocation: ctx.invocationCount,
    };
  },
});

const waiter = hatchet.durableTask({
  name: 'waiter',
  fn: async (_input: {}, ctx) => {
    const event = await ctx.waitForEvent('order:paid');
    return { paid: event };
  },
});

const parent = hatchet.durableTask({
  name: 'parent',
  fn: async (input: { message: string }, ctx) => {
    const child = await ctx.spawnChild(echo, { message: `${input.message} from parent` });
    return { child };
  },
});

const failing = hatchet.durableTask({
  name: 'failing',
  fn: async (input: { permanent?: boolean }) => {
    throw input.permanent ? new NonRetryableError('permanent') : new Error('transient');
  },
});

const workflows = [echo, sleeper, waiter, parent, failing];

function operator(extra: Partial<Parameters<typeof createTestOperator>[0]> = {}) {
  return createTestOperator({ workflows, secret, console: quiet, ...extra });
}

async function settle(): Promise<void> {
  // Drain microtasks and zero-delay timers between protocol steps.
  await vi.advanceTimersByTimeAsync(0);
}

describe('durable invocations', () => {
  beforeEach(() => {
    vi.useFakeTimers();
  });

  afterEach(() => {
    expect(vi.getTimerCount()).toBe(0);
    vi.useRealTimers();
  });

  it('completes a sleep shorter than the inline budget inline', async () => {
    const op = operator();
    const run = op.startDurable(
      sleeper,
      { message: 'hi', sleep: '1s' },
      { inlineWaitBudgetMs: 5000 }
    );

    await run.frame('waitFor');
    op.clock.advance('1s');

    const result = await run.result;

    expect(result.status).toBe('completed');
    expect(result.output).toMatchObject({ echo: 'hi', invocation: 1 });
    expect(result.closeCode).toBe(1000);
    expect(result.endpointFrames).toEqual(['memo', 'completeMemo', 'waitFor', 'done:output']);
    expect(result.operatorFrames).toEqual([
      'first',
      'memoAck',
      'entryCompleted',
      'waitForAck',
      'entryCompleted',
    ]);
  });

  it('evicts when the sleep outlives the budget and resumes with the memoized value', async () => {
    const op = operator();
    const run = op.startDurable(sleeper, { message: 'hi' }, { inlineWaitBudgetMs: 100 });

    await run.frame('waitFor');
    await vi.advanceTimersByTimeAsync(100);

    const first = await run.result;

    expect(first.status).toBe('evicted');
    expect(first.endpointFrames).toEqual([
      'memo',
      'completeMemo',
      'waitFor',
      'evictInvocation',
      'done:evicted',
    ]);
    expect(first.operatorFrames).toEqual([
      'first',
      'memoAck',
      'entryCompleted',
      'waitForAck',
      'evictionAck',
    ]);

    const evict = first.frames.find((f) => f.frame.request?.evictInvocation);
    expect(evict?.frame.request?.evictInvocation?.reason).toBe(
      'inline wait budget of 100ms elapsed'
    );

    const memoized = first.frames.find((f) => f.frame.request?.completeMemo);
    const startedAt = JSON.parse(
      new TextDecoder().decode(memoized!.frame.request!.completeMemo!.payload)
    ).ts;

    op.clock.advance('3s');

    const second = await op.resume(first);

    expect(second.status).toBe('completed');
    expect(second.invocationCount).toBe(2);
    expect(second.output).toEqual({ echo: 'hi', startedAt, invocation: 2 });
    // The memo replays from the log: no completeMemo, and the wait completes at once.
    expect(second.endpointFrames).toEqual(['memo', 'waitFor', 'done:output']);
    expect(second.operatorFrames).toEqual([
      'first',
      'memoAck',
      'entryCompleted',
      'waitForAck',
      'entryCompleted',
    ]);
  });

  it('satisfies waitForEvent from an emitted event', async () => {
    const op = operator();
    const run = op.startDurable(waiter, {}, { inlineWaitBudgetMs: 5000 });

    await run.frame('waitFor');
    op.emit('order:paid', { amount: 42 });

    const result = await run.result;

    expect(result.status).toBe('completed');
    expect(result.output).toEqual({ paid: { amount: 42 } });
  });

  it('spawns a child through trigger_runs and gets its output back', async () => {
    const op = operator();
    const result = await op.invokeDurable(
      parent,
      { message: 'hello' },
      { inlineWaitBudgetMs: 5000 }
    );

    expect(result.status).toBe('completed');
    expect(result.output).toEqual({ child: { echo: 'hello from parent' } });
    expect(result.endpointFrames).toEqual(['triggerRuns', 'done:output']);
    expect(result.operatorFrames).toEqual(['first', 'triggerRunsAck', 'entryCompleted']);

    const trigger = result.frames.find((f) => f.frame.request?.triggerRuns);
    expect(trigger?.frame.request?.triggerRuns?.triggerOpts[0].name).toBe('echo');
  });

  it('aborts cleanly on a server eviction without sending a done frame', async () => {
    const op = operator();
    const run = op.startDurable(sleeper, { message: 'hi' }, { inlineWaitBudgetMs: 5000 });

    await run.frame('waitFor');
    run.serverEvict('superseded');

    const result = await run.result;

    expect(result.status).toBe('evicted');
    expect(result.closeCode).toBe(4001);
    expect(result.endpointFrames).toEqual(['memo', 'completeMemo', 'waitFor']);
  });

  it('finishes with done error retry false after a nondeterminism error frame', async () => {
    const op = operator();
    const run = op.startDurable(sleeper, { message: 'hi' }, { inlineWaitBudgetMs: 5000 });

    await run.frame('waitFor');
    run.sendError('nondeterminism', 'replay diverged');

    const result = await run.result;

    expect(result.status).toBe('failed');
    expect(result.retry).toBe(false);
    expect(result.error).toMatch(/replay diverged/);
    expect(result.endpointFrames).toEqual(['memo', 'completeMemo', 'waitFor', 'done:error']);
  });

  it('reports nondeterminism when a replay diverges from the log', async () => {
    const op = operator();
    const run = op.startDurable(sleeper, { message: 'hi' }, { inlineWaitBudgetMs: 100 });

    await run.frame('waitFor');
    await vi.advanceTimersByTimeAsync(100);

    const first = await run.result;
    expect(first.status).toBe('evicted');

    // Resume with a different task at the same run: its first event is trigger_runs, not memo.
    const diverged = await op.invokeDurable(
      parent,
      { message: 'x' },
      {
        taskRunExternalId: first.taskRunExternalId,
        invocationCount: 2,
        inlineWaitBudgetMs: 5000,
      }
    );

    expect(diverged.status).toBe('failed');
    expect(diverged.retry).toBe(false);
    expect(diverged.operatorFrames).toEqual(['first', 'error']);
    expect(diverged.endpointFrames).toEqual(['triggerRuns', 'done:error']);
  });

  it('aborts without a done frame when the operator closes mid-task', async () => {
    const op = operator();
    const run = op.startDurable(sleeper, { message: 'hi' }, { inlineWaitBudgetMs: 5000 });

    await run.frame('waitFor');
    run.close(4002, 'operator shutting down');

    const result = await run.result;

    expect(result.status).toBe('failed');
    expect(result.retry).toBe(true);
    expect(result.closeCode).toBe(4002);
    expect(result.endpointFrames).toEqual(['memo', 'completeMemo', 'waitFor']);
    await settle();
  });

  it('maps task failures onto done error with the retry decision', async () => {
    const op = operator();

    const transient = await op.invokeDurable(failing, {}, { inlineWaitBudgetMs: 5000 });
    expect(transient).toMatchObject({ status: 'failed', error: 'transient', retry: true });

    const permanent = await op.invokeDurable(failing, { permanent: true });
    expect(permanent).toMatchObject({ status: 'failed', error: 'permanent', retry: false });
  });

  it('refuses a non-durable action over the socket', async () => {
    const op = operator();
    const result = await op.invokeDurable(echo, { message: 'hi' });

    expect(result.status).toBe('failed');
    expect(result.retry).toBe(false);
    expect(result.error).toMatch(/not a durable task/);
    expect(result.endpointFrames).toEqual(['done:error']);
  });

  it('refuses a bad signature, a stale timestamp and a foreign endpoint id', async () => {
    const op = operator();
    const taskId = crypto.randomUUID();
    const url = `https://endpoint.test${op.handler.basePath}/trigger`;
    const upgrade = vi.fn(() => new Response(null, { status: 200 }));

    const attempt = async (overrides: Parameters<typeof op.upgradeHeaders>[2]) =>
      op.handler.fetch(
        new Request(url, { headers: await op.upgradeHeaders(taskId, 1, overrides) }),
        undefined,
        { upgrade }
      );

    expect((await attempt({ signature: 'nope' })).status).toBe(401);
    expect((await attempt({ secret: 'another-secret-of-thirty-two-chars!!' })).status).toBe(401);
    expect((await attempt({ timestamp: String(Math.floor(Date.now() / 1000) - 301) })).status).toBe(
      401
    );
    expect((await attempt({ endpointId: 'someone-else' })).status).toBe(403);
    expect((await attempt({})).status).toBe(200);
    expect(upgrade).toHaveBeenCalledTimes(1);
  });

  it('answers 426 when the adapter has no upgrade hook', async () => {
    const op = operator();
    const response = await op.handler.fetch(
      new Request(`https://endpoint.test${op.handler.basePath}/trigger`, {
        headers: await op.upgradeHeaders(crypto.randomUUID(), 1),
      })
    );

    expect(response.status).toBe(426);
  });

  it('advertises durable support only with an upgrade hook and a durable task', async () => {
    expect((await operator().healthcheck()).durable?.supported).toBe(true);
    expect((await operator({ durable: false }).healthcheck()).durable?.supported).toBe(false);
    expect((await operator({ workflows: [echo] }).healthcheck()).durable?.supported).toBe(false);
  });

  it('closes the socket with 1000 after the done frame', async () => {
    const op = operator();
    const result: DurableResult = await op.invokeDurable(parent, { message: 'x' });

    expect(result.closeCode).toBe(1000);
  });
});
