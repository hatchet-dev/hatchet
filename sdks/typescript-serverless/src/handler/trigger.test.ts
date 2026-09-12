import { describe, expect, it, vi } from 'vitest';
import { NonRetryableError, ServerlessLimitationError, hatchet } from '../index';
import { InvocationFailedError, createTestOperator } from '../testing';
import { ServerlessTriggerRequest } from '../generated/proto/v1/serverless';
import { ActionType } from '../generated/proto/dispatcher';

const secret = 'test-secret-at-least-32-characters-long';
const quiet = { debug: vi.fn(), info: vi.fn(), warn: vi.fn(), error: vi.fn() };

const echo = hatchet.task({
  name: 'echo',
  fn: async (input: { message: string }, ctx) => ({
    echo: input.message,
    runId: ctx.workflowRunId(),
    retry: ctx.retryCount(),
    metadata: ctx.additionalMetadata(),
    workflow: ctx.workflowNameV1(),
    actionId: ctx.action.actionId,
    taskName: ctx.taskName(),
  }),
});

const nothing = hatchet.task({
  name: 'nothing',
  fn: async () => {},
});

const permanent = hatchet.task({
  name: 'permanent',
  fn: async () => {
    throw new NonRetryableError('do not retry me');
  },
});

const spawner = hatchet.task({
  name: 'spawner',
  fn: async (_input: {}, ctx) => {
    await ctx.runChild(echo, { message: 'child' });
    return {};
  },
});

const crasher = hatchet.task({
  name: 'crasher',
  fn: async () => {
    throw new Error('boom');
  },
});

const pipeline = hatchet.workflow<{ url: string }>({ name: 'pipeline' });
const fetchStep = pipeline.task({ name: 'fetch', fn: async () => ({ body: 'a b c' }) });
const summarize = pipeline.task({
  name: 'summarize',
  parents: [fetchStep],
  fn: async (_input, ctx) => ({
    words: (await ctx.parentOutput(fetchStep)).body.split(' ').length,
  }),
});

const sleeper = hatchet.durableTask({
  name: 'sleep-then-echo',
  fn: async (input: { message: string }) => ({ echo: input.message }),
});

const logger = hatchet.task({
  name: 'logger',
  fn: async (_input: {}, ctx) => {
    await ctx.logger.info('hello from the task', { key: 'value' });
    ctx.log('legacy log line');
    return { logged: true };
  },
});

const workflows = [echo, nothing, permanent, spawner, crasher, pipeline, sleeper, logger];

function operator(extra: Partial<Parameters<typeof createTestOperator>[0]> = {}) {
  return createTestOperator({ workflows, secret, console: quiet, ...extra });
}

describe('trigger outcomes', () => {
  it('returns a value: 200 with the JSON output', async () => {
    const outcome = await operator().deliver(echo, { message: 'hi' });

    expect(outcome).toMatchObject({ status: 'completed', httpStatus: 200 });
    expect((outcome as { output: { echo: string } }).output.echo).toBe('hi');
  });

  it('returns undefined: 204 with no body', async () => {
    const op = operator();

    expect(await op.deliver(nothing, {})).toEqual({
      status: 'completed',
      output: undefined,
      httpStatus: 204,
    });
    expect(await op.invoke(nothing, {})).toBeUndefined();
  });

  it('throws NonRetryableError: 422, retry false', async () => {
    expect(await operator().deliver(permanent, {})).toEqual({
      status: 'failed',
      error: 'do not retry me',
      retry: false,
      httpStatus: 422,
    });
  });

  it('throws ServerlessLimitationError: 422, retry false, naming the feature', async () => {
    const outcome = await operator().deliver(spawner, {});

    expect(outcome).toMatchObject({ status: 'failed', retry: false, httpStatus: 422 });
    expect((outcome as { error: string }).error).toMatch(/ctx\.runChild/);
    expect((outcome as { error: string }).error).toMatch(/no Hatchet client/);
  });

  it('throws anything else: 500, retry true', async () => {
    expect(await operator().deliver(crasher, {})).toEqual({
      status: 'failed',
      error: 'boom',
      retry: true,
      httpStatus: 500,
    });
    expect(quiet.error).toHaveBeenCalled();
  });

  it('action not served here: 404, retry false', async () => {
    expect(await operator().deliver('missing:task', {})).toEqual({
      status: 'failed',
      error: 'no task served for action missing:task',
      retry: false,
      httpStatus: 404,
    });

    // Declared but outside the serve subset.
    expect(await operator({ serve: [echo] }).deliver(nothing, {})).toMatchObject({
      status: 'failed',
      httpStatus: 404,
    });
  });

  it('bad signature: 401', async () => {
    const op = operator();
    const path = `${op.handler.basePath}/trigger`;
    const response = await op.request(path, { body: '{}', sign: false });

    expect(response.status).toBe(401);
    expect(await response.json()).toEqual({ error: 'bad signature', retry: false });

    const forged = await op.handler.fetch(
      new Request(`https://endpoint.test${path}`, {
        method: 'POST',
        body: '{}',
        headers: { 'X-Hatchet-Signature': 'nope' },
      })
    );

    expect(forged.status).toBe(401);
  });

  it('durable action over POST: 422, retry false', async () => {
    const outcome = await operator().deliver(sleeper, { message: 'x' });

    expect(outcome).toMatchObject({ status: 'failed', retry: false, httpStatus: 422 });
    expect((outcome as { error: string }).error).toMatch(/durable/);
  });

  it('invoke throws InvocationFailedError with the mapping', async () => {
    await expect(operator().invoke(crasher, {})).rejects.toBeInstanceOf(InvocationFailedError);
    await expect(operator().invoke(permanent, {})).rejects.toMatchObject({
      retry: false,
      httpStatus: 422,
    });
  });

  it('refuses a GET, a malformed body, a future envelope version and a cancel action', async () => {
    const op = operator();
    const path = `${op.handler.basePath}/trigger`;

    expect((await op.request(path, { method: 'GET' })).status).toBe(405);
    expect((await op.request(path, { body: 'not json' })).status).toBe(400);

    const v2 = ServerlessTriggerRequest.toJSON({
      endpointId: op.endpointId,
      namespace: op.namespace,
      timestamp: Math.floor(Date.now() / 1000),
      version: 2,
      action: undefined,
    });
    expect((await op.request(path, { body: JSON.stringify(v2) })).status).toBe(400);

    const cancel = ServerlessTriggerRequest.toJSON(
      ServerlessTriggerRequest.fromPartial({
        endpointId: op.endpointId,
        namespace: op.namespace,
        timestamp: Math.floor(Date.now() / 1000),
        version: 1,
        action: { actionId: `${op.namespace}_echo:echo`, actionType: ActionType.CANCEL_STEP_RUN },
      })
    );
    const response = await op.request(path, { body: JSON.stringify(cancel) });

    expect(response.status).toBe(400);
    expect(await response.json()).toMatchObject({
      error: expect.stringMatching(/CANCEL_STEP_RUN/),
      retry: false,
    });
  });
});

describe('namespaces and context', () => {
  it('strips a UUID namespace from the workflow name and the action service part', async () => {
    const namespace = '0f7b2f3a-1c2d-4e5f-8a9b-0c1d2e3f4a5b';
    const output = await operator({ namespace }).invoke<{
      workflow: string;
      actionId: string;
      taskName: string;
    }>(echo, { message: 'ns' });

    expect(output.workflow).toBe('echo');
    expect(output.actionId).toBe('echo:echo');
    expect(output.taskName).toBe('echo');
  });

  it('populates the context from the assigned action', async () => {
    const output = await operator().invoke<{
      runId: string;
      retry: number;
      metadata: Record<string, string>;
    }>(
      echo,
      { message: 'ctx' },
      { workflowRunId: 'run-123', retryCount: 2, additionalMetadata: { source: 'test' } }
    );

    expect(output).toMatchObject({ runId: 'run-123', retry: 2, metadata: { source: 'test' } });
  });

  it('exposes parent outputs from the payload', async () => {
    const output = await operator().invoke<{ words: number }>(
      { workflow: pipeline, task: summarize },
      { url: 'https://example.com' },
      { parents: { fetch: { body: 'one two three four' } } }
    );

    expect(output).toEqual({ words: 4 });
  });

  it('logs to the console and never to the engine', async () => {
    const out = { debug: vi.fn(), info: vi.fn(), warn: vi.fn(), error: vi.fn() };
    const output = await operator({ console: out }).invoke(logger, {});

    expect(output).toEqual({ logged: true });
    expect(out.info).toHaveBeenCalledWith(
      expect.stringContaining('hello from the task'),
      expect.objectContaining({ key: 'value' })
    );
    expect(out.info).toHaveBeenCalledWith(
      expect.stringContaining('legacy log line'),
      expect.anything()
    );
  });

  it('exports the limitation error for users to catch', () => {
    const err = new ServerlessLimitationError('ctx.putStream');

    expect(err.feature).toBe('ctx.putStream');
    expect(err.message).toMatch(/LIMITATIONS\.md/);
  });
});
