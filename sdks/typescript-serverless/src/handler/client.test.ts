import { createRouterTransport } from '@connectrpc/connect';
import { EventsService } from '@hatchet-dev/typescript-sdk/protoc-es/events/events_pb.js';
import {
  AdminService,
  RunStatus,
  type GetRunDetailsResponse,
} from '@hatchet-dev/typescript-sdk/protoc-es/v1/workflows_pb.js';
import { WorkflowService } from '@hatchet-dev/typescript-sdk/protoc-es/workflows/workflows_pb.js';
import type { MessageInitShape } from '@bufbuild/protobuf';
import { describe, expect, it, vi } from 'vitest';
import { HatchetCore, ServerlessLimitationError, hatchet } from '../index';
import { createTestOperator } from '../testing';
import { createClientResolver } from './client';
import { RunStreamClosedError } from './runs';

const secret = 'test-secret-at-least-32-characters-long';
const quiet = { debug: vi.fn(), info: vi.fn(), warn: vi.fn(), error: vi.fn() };
const TENANT_ID = '707d0855-80ab-4e1f-a156-f1c4546cbf52';

function makeToken(claims: Record<string, unknown>): string {
  const encode = (value: unknown) => Buffer.from(JSON.stringify(value)).toString('base64url');
  return `${encode({ alg: 'HS256', typ: 'JWT' })}.${encode(claims)}.signature`;
}

const TOKEN = makeToken({ sub: TENANT_ID, grpc_broadcast_address: 'engine.example.com:7070' });

type RunDetails = MessageInitShape<typeof AdminService.method.getRunDetails.output>;

/** The engine's unary side, as the core client reaches it over its transport. */
function fakeEngine() {
  const triggers: Array<Record<string, unknown>> = [];
  const streamed: Array<{ taskRunExternalId: string; message: string; index: number }> = [];
  const logs: Array<{ taskRunExternalId: string; message: string; level: string }> = [];
  const cancels: string[][] = [];
  let details: RunDetails | undefined;

  const transport = createRouterTransport(({ service }) => {
    service(WorkflowService, {
      triggerWorkflow: (req) => {
        triggers.push({
          name: req.name,
          input: JSON.parse(req.input),
          parentId: req.parentId,
          parentTaskRunExternalId: req.parentTaskRunExternalId,
          childIndex: req.childIndex,
          childKey: req.childKey,
        });
        return { workflowRunId: `run-${triggers.length}` };
      },
    });
    service(AdminService, {
      getRunDetails: (): GetRunDetailsResponse => {
        if (!details) throw new Error('no run details queued');
        return details as GetRunDetailsResponse;
      },
      cancelTasks: (req) => {
        cancels.push(req.externalIds);
        return { cancelledTasks: req.externalIds };
      },
    });
    service(EventsService, {
      putStreamEvent: (req) => {
        streamed.push({
          taskRunExternalId: req.taskRunExternalId,
          message: new TextDecoder().decode(req.message),
          index: Number(req.eventIndex ?? -1),
        });
        return {};
      },
      putLog: (req) => {
        logs.push({
          taskRunExternalId: req.taskRunExternalId,
          message: req.message,
          level: String(req.level),
        });
        return {};
      },
    });
  });

  return {
    transport,
    triggers,
    streamed,
    logs,
    cancels,
    completeDetails(taskName: string, output: unknown) {
      details = {
        status: RunStatus.COMPLETED,
        done: true,
        taskRuns: {
          [taskName]: {
            externalId: `${taskName}-id`,
            readableId: taskName,
            status: RunStatus.COMPLETED,
            output: new TextEncoder().encode(JSON.stringify(output)),
          },
        },
      };
    },
  };
}

const echo = hatchet.task({
  name: 'echo',
  fn: async (input: { message: string }) => ({ echo: input.message }),
});

const reporter = hatchet.task({
  name: 'reporter',
  fn: async (_input: {}, ctx) => {
    await ctx.putStream('chunk one');
    await ctx.putStream(new TextEncoder().encode('chunk two'));
    await ctx.log('hello from the task', 'WARN');
    await ctx.cancel();
    return { taskRunExternalId: ctx.taskRunExternalId() };
  },
});

const spawner = hatchet.task({
  name: 'spawner',
  fn: async (input: { wait?: boolean }, ctx) => {
    const ref = await ctx.runNoWaitChild(echo, { message: 'child' }, { key: 'first' });

    if (!input.wait) {
      return { childRunId: await ref.getWorkflowRunId() };
    }

    return { child: await ref.result() };
  },
});

const parent = hatchet.task({
  name: 'parent',
  fn: async (input: { message: string }, ctx) => ({
    child: await ctx.runChild(echo, { message: input.message }),
  }),
});

const fanOut = hatchet.task({
  name: 'fan-out',
  fn: async (input: { count: number }, ctx) => ({
    children: await ctx.bulkRunChildren(
      Array.from({ length: input.count }, (_, i) => ({
        workflow: echo,
        input: { message: `child ${i}` },
      }))
    ),
  }),
});

const durableSpawner = hatchet.durableTask({
  name: 'durable-spawner',
  fn: async (_input: {}, ctx) => ({ child: await ctx.runChild(echo, { message: 'x' }) }),
});

const workflows = [echo, reporter, spawner, parent, fanOut, durableSpawner];

describe('ctx over the client', () => {
  it('putStream, log and cancel go to the engine as unary calls over a POST', async () => {
    const engine = fakeEngine();
    const op = createTestOperator({
      workflows,
      secret,
      console: quiet,
      client: { token: TOKEN, transport: engine.transport, logLevel: 'OFF' },
    });

    const output = await op.invoke<{ taskRunExternalId: string }>(reporter, {});
    const id = output.taskRunExternalId;

    expect(engine.streamed).toEqual([
      { taskRunExternalId: id, message: 'chunk one', index: 0 },
      { taskRunExternalId: id, message: 'chunk two', index: 1 },
    ]);
    expect(engine.logs).toEqual([
      { taskRunExternalId: id, message: 'hello from the task', level: 'WARN' },
    ]);
    expect(engine.cancels).toEqual([[id]]);
  });

  it('without a client the same members throw, naming the option', async () => {
    const op = createTestOperator({ workflows, secret, console: quiet });
    const outcome = await op.deliver(reporter, {});

    expect(outcome).toMatchObject({ status: 'failed', retry: false, httpStatus: 422 });
    expect((outcome as { error: string }).error).toMatch(
      /ctx\.putStream is not available in the serverless runtime: configure `client` to enable this/
    );
  });

  it('runNoWaitChild over a POST triggers the child with the parent set, and its result() names the streams option', async () => {
    const engine = fakeEngine();
    const op = createTestOperator({
      workflows,
      secret,
      console: quiet,
      client: { token: TOKEN, transport: engine.transport, logLevel: 'OFF' },
    });

    const output = await op.invoke(spawner, {}, { workflowRunId: 'parent-run' });
    expect(output).toEqual({ childRunId: 'run-1' });
    expect(engine.triggers).toEqual([
      {
        name: 'echo',
        input: { message: 'child' },
        parentId: 'parent-run',
        parentTaskRunExternalId: expect.any(String),
        childIndex: 0,
        childKey: 'first',
      },
    ]);

    const waited = await op.deliver(spawner, { wait: true });
    expect(waited).toMatchObject({ status: 'failed', retry: false, httpStatus: 422 });
    expect((waited as { error: string }).error).toMatch(
      /invoked over a POST and has no invocation socket.*`streams` option/
    );
  });

  it('runChild on a task listed in streams awaits the child on one runs stream over the socket', async () => {
    const engine = fakeEngine();
    const op = createTestOperator({
      workflows,
      secret,
      console: quiet,
      streams: [parent],
      client: { token: TOKEN, transport: engine.transport, logLevel: 'OFF' },
      runStream: {
        workflowRun: (workflowRunId) => ({
          workflowRunId,
          eventType: 0,
          eventTimestamp: undefined,
          results: [
            {
              taskRunExternalId: 'echo-id',
              taskName: 'echo',
              jobRunId: 'job',
              output: JSON.stringify({ echo: `answer for ${workflowRunId}` }),
            },
          ],
        }),
      },
    });

    expect((await op.healthcheck()).tasks).toEqual([{ action: 'parent:parent', streams: true }]);

    // The catalog flags the task, so `invoke` takes the socket the way the operator does.
    expect(await op.invoke(parent, { message: 'hi' })).toEqual({
      child: { echo: 'answer for run-1' },
    });
    expect(engine.triggers[0]).toMatchObject({ name: 'echo', input: { message: 'hi' } });

    const run = op.startDurable(parent, { message: 'again' });
    const result = await run.result;

    expect(result.status).toBe('completed');
    expect(result.output).toEqual({ child: { echo: 'answer for run-2' } });
    expect(result.endpointFrames).toEqual(['streamOpen', 'streamClose', 'done:output']);
    expect(result.frames[1].frame.streamOpen).toEqual({
      id: '1',
      procedure: '/Dispatcher/SubscribeToWorkflowRuns',
      request: '{"workflowRunId":"run-2"}',
    });
    expect(result.operatorFrames).toEqual(['first', 'streamMessage']);
  });

  it('bulkRunChildren fans out on the one stream and keeps input order', async () => {
    const engine = fakeEngine();
    const op = createTestOperator({
      workflows,
      secret,
      console: quiet,
      streams: ['fan-out:fan-out'],
      client: { token: TOKEN, transport: engine.transport, logLevel: 'OFF' },
    });

    const run = op.startDurable(fanOut, { count: 3 });
    await run.frame('streamMessage', 2);

    expect(run.openStreams()).toEqual(['1']);
    op.completeRun('run-3', [{ taskName: 'echo', output: { n: 3 } }]);
    op.completeRun('run-1', [{ taskName: 'echo', output: { n: 1 } }]);
    op.completeRun('run-2', [{ taskName: 'echo', output: { n: 2 } }]);

    const result = await run.result;
    expect(result.status).toBe('completed');
    expect(result.output).toEqual({ children: [{ n: 1 }, { n: 2 }, { n: 3 }] });
    expect(result.endpointFrames).toEqual([
      'streamOpen',
      'streamMessage',
      'streamMessage',
      'streamClose',
      'done:output',
    ]);
    expect(engine.triggers.map((t) => t.childIndex)).toEqual([0, 1, 2]);
  });

  it('a failed child rejects the wait with its error, and an empty terminal event falls back to run details', async () => {
    const engine = fakeEngine();
    const op = createTestOperator({
      workflows,
      secret,
      console: quiet,
      streams: [parent],
      client: { token: TOKEN, transport: engine.transport, logLevel: 'OFF' },
    });

    const failing = op.startDurable(parent, { message: 'fail' });
    await failing.frame('streamOpen');
    op.completeRun('run-1', [{ taskName: 'echo', error: 'child exploded' }]);
    expect(await failing.result).toMatchObject({
      status: 'failed',
      error: 'child exploded',
      retry: true,
    });

    const empty = op.startDurable(parent, { message: 'empty' });
    await empty.frame('streamOpen');
    engine.completeDetails('echo', { echo: 'from details' });
    op.completeRun('run-2', []);
    expect(await empty.result).toMatchObject({
      status: 'completed',
      output: { child: { echo: 'from details' } },
    });
  });

  it('the operator ending the runs stream or the socket fails the wait', async () => {
    const engine = fakeEngine();
    const op = createTestOperator({
      workflows,
      secret,
      console: quiet,
      streams: [parent],
      client: { token: TOKEN, transport: engine.transport, logLevel: 'OFF' },
    });

    const dropped = op.startDurable(parent, { message: 'x' });
    await dropped.frame('streamOpen');
    dropped.closeStream('1', 14, 'the engine stream was closed');
    const outcome = await dropped.result;

    expect(outcome.status).toBe('failed');
    expect(outcome.error).toMatch(
      /run stream closed before run run-1 finished \(code 14: the engine stream was closed\)/
    );
    expect(outcome.endpointFrames).toEqual(['streamOpen', 'done:error']);

    const errors = quiet.error.mock.calls.map((call) => call[1]);
    expect(errors.at(-1)).toBeInstanceOf(RunStreamClosedError);

    const closed = op.startDurable(parent, { message: 'y' });
    await closed.frame('streamOpen');
    closed.close(4003, 'cancelled');
    expect((await closed.result).status).toBe('failed');
  });

  it('a durable task must spawn children through the durable log', async () => {
    const engine = fakeEngine();
    const op = createTestOperator({
      workflows,
      secret,
      console: quiet,
      client: { token: TOKEN, transport: engine.transport, logLevel: 'OFF' },
    });

    const result = await op.invokeDurable(durableSpawner, {});

    expect(result).toMatchObject({ status: 'failed', retry: false });
    expect(result.error).toMatch(/ctx\.runChild .* durable log\. Use ctx\.spawnChild/);
    expect(engine.triggers).toEqual([]);
  });

  it('a flagged task delivered over a POST by an older operator still runs', async () => {
    const engine = fakeEngine();
    const op = createTestOperator({
      workflows,
      secret,
      console: quiet,
      streams: [spawner],
      client: { token: TOKEN, transport: engine.transport, logLevel: 'OFF' },
    });

    // The test operator routes flagged tasks over the socket; a raw POST is the older path.
    const response = await op.request(`${op.handler.basePath}/healthcheck`);
    expect(response.status).toBe(200);
    expect((await op.healthcheck()).tasks).toEqual([{ action: 'spawner:spawner', streams: true }]);
    expect(await op.invoke(spawner, {})).toEqual({ childRunId: 'run-1' });
  });
});

describe('createClientResolver', () => {
  const config = () => ({
    token: TOKEN,
    transport: fakeEngine().transport,
    logLevel: 'OFF' as const,
  });

  it('builds one client per config object, on first use, and reuses it across environments', () => {
    const resolve = createClientResolver(config());
    const first = resolve({ a: 1 })();
    const second = resolve({ b: 2 })();

    expect(first).toBeInstanceOf(HatchetCore);
    expect(second).toBe(first);
  });

  it('builds one client per environment object for the function form', () => {
    const option = vi.fn(() => config());
    const resolve = createClientResolver(option);
    const envA = { HATCHET_CLIENT_TOKEN: 'a' };
    const envB = { HATCHET_CLIENT_TOKEN: 'b' };

    const sourceA = resolve(envA);
    expect(option).not.toHaveBeenCalled();

    const a = sourceA();
    expect(a).toBeInstanceOf(HatchetCore);
    expect(sourceA()).toBe(a);
    expect(resolve(envA)()).toBe(a);
    expect(resolve(envB)()).not.toBe(a);
    expect(resolve(undefined)()).toBe(resolve(undefined)());
    // Once per source that was used: envA, envA again, envB, and undefined twice.
    expect(option).toHaveBeenCalledTimes(5);
  });

  it('hands a ready client through as is, and nothing without an option', () => {
    const client = new HatchetCore(config());

    expect(createClientResolver(client)({})()).toBe(client);
    expect(createClientResolver(() => client)({})()).toBe(client);
    expect(createClientResolver(undefined)({})()).toBeUndefined();
    expect(createClientResolver(() => undefined)({})()).toBeUndefined();
  });

  it('fails at first use rather than when the handler is built', () => {
    const source = createClientResolver({ token: 'not a jwt' })({});

    expect(() => source()).toThrow();
  });
});

describe('ServerlessLimitationError', () => {
  it('states the reason and the fix', () => {
    expect(new ServerlessLimitationError('ctx.putStream').message).toBe(
      'ctx.putStream is not available in the serverless runtime: configure `client` to enable this. See LIMITATIONS.md in @hatchet-dev/serverless.'
    );
    expect(
      new ServerlessLimitationError('ctx.releaseSlot', 'Nothing to do', 'there is no worker')
        .message
    ).toBe(
      'ctx.releaseSlot is not available in the serverless runtime: there is no worker. Nothing to do. See LIMITATIONS.md in @hatchet-dev/serverless.'
    );
  });
});
