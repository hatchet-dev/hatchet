import { createRouterTransport } from '@connectrpc/connect';
import { Dispatcher } from '@hatchet-dev/typescript-sdk/protoc-es/dispatcher/dispatcher_pb.js';
import { EventsService } from '@hatchet-dev/typescript-sdk/protoc-es/events/events_pb.js';
import {
  AdminService,
  RunStatus,
  type GetRunDetailsResponse,
} from '@hatchet-dev/typescript-sdk/protoc-es/v1/workflows_pb.js';
import { WorkflowService } from '@hatchet-dev/typescript-sdk/protoc-es/workflows/workflows_pb.js';
import type { MessageInitShape } from '@bufbuild/protobuf';
import { describe, expect, it, vi } from 'vitest';
import { HatchetCore, ServerlessLimitationError, clientFromContext, hatchet } from '../index';
import { ServerlessTriggerRequest } from '../generated/proto/v1/serverless';
import { ActionType } from '../generated/proto/dispatcher';
import { createTestOperator } from '../testing';
import { clientConfigFromEnv, createClientResolver } from './client';
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
  const refreshes: Array<{ taskRunExternalId: string; incrementTimeoutBy: string }> = [];
  const releases: string[] = [];
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
    service(Dispatcher, {
      refreshTimeout: (req) => {
        refreshes.push({
          taskRunExternalId: req.taskRunExternalId,
          incrementTimeoutBy: req.incrementTimeoutBy,
        });
        return {};
      },
      releaseSlot: (req) => {
        releases.push(req.taskRunExternalId);
        return {};
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
    refreshes,
    releases,
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

// The child: a function of its input, so it is POSTed.
const echo = hatchet.task({
  name: 'echo',
  hatchetClient: false,
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

// Extends its own timeout in each duration form the context accepts, then gives up its slot.
const extender = hatchet.task({
  name: 'extender',
  fn: async (_input: {}, ctx) => {
    await ctx.refreshTimeout('30s');
    await ctx.refreshTimeout({ seconds: 30 });
    await ctx.refreshTimeout(30_000);
    await ctx.releaseSlot();
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

// Awaits its child on the socket every task has by default.
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

/** A pure task that reaches for one client-backed member, named by its input. */
const pureReporter = hatchet.task({
  name: 'pure-reporter',
  hatchetClient: false,
  fn: async (input: { member: string }, ctx): Promise<{ logged?: boolean }> => {
    switch (input.member) {
      case 'ctx.runChild':
        await ctx.runChild(echo, { message: 'x' });
        break;
      case 'ctx.runNoWaitChild':
        await ctx.runNoWaitChild(echo, { message: 'x' });
        break;
      case 'ctx.bulkRunChildren':
        await ctx.bulkRunChildren([{ workflow: echo, input: { message: 'x' } }]);
        break;
      case 'ctx.bulkRunNoWaitChildren':
        await ctx.bulkRunNoWaitChildren([{ workflow: echo, input: { message: 'x' } }]);
        break;
      case 'ctx.spawnWorkflow':
        await ctx.spawnWorkflow('echo', { message: 'x' });
        break;
      case 'ctx.spawnWorkflows':
        await ctx.spawnWorkflows([{ workflow: 'echo', input: { message: 'x' } }]);
        break;
      case 'ctx.putStream':
        await ctx.putStream('chunk');
        break;
      case 'ctx.cancel':
        await ctx.cancel();
        break;
      case 'ctx.refreshTimeout':
        await ctx.refreshTimeout('30s');
        break;
      case 'ctx.releaseSlot':
        await ctx.releaseSlot();
        break;
      case 'clientFromContext(ctx)':
        clientFromContext(ctx);
        break;
      case 'ctx.log':
        await ctx.log('console only', 'WARN');
        return { logged: true };
      default:
        throw new Error(`unknown member ${input.member}`);
    }

    return {};
  },
});

const workflows = [echo, reporter, extender, spawner, parent, fanOut, durableSpawner, pureReporter];

describe('ctx over the client', () => {
  it('putStream, log and cancel go to the engine as unary calls through the client', async () => {
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

  it('refreshTimeout and releaseSlot reach the dispatcher through the client, in either delivery', async () => {
    const engine = fakeEngine();
    const client = { token: TOKEN, transport: engine.transport, logLevel: 'OFF' as const };

    // Over the socket, the default delivery.
    const dialed = createTestOperator({ workflows, secret, console: quiet, client });
    const { taskRunExternalId: socketId } = await dialed.invoke<{ taskRunExternalId: string }>(
      extender,
      {}
    );

    // Over a POST: the calls are unary, so the socket is not needed.
    const posted = createTestOperator({
      workflows,
      secret,
      console: quiet,
      client,
      durable: false,
    });
    const { taskRunExternalId: postId } = await posted.invoke<{ taskRunExternalId: string }>(
      extender,
      {}
    );

    expect(socketId).not.toBe(postId);
    expect(engine.refreshes).toEqual([
      { taskRunExternalId: socketId, incrementTimeoutBy: '30s' },
      { taskRunExternalId: socketId, incrementTimeoutBy: '30s' },
      { taskRunExternalId: socketId, incrementTimeoutBy: '30s' },
      { taskRunExternalId: postId, incrementTimeoutBy: '30s' },
      { taskRunExternalId: postId, incrementTimeoutBy: '30s' },
      { taskRunExternalId: postId, incrementTimeoutBy: '30s' },
    ]);
    expect(engine.releases).toEqual([socketId, postId]);
  });

  it('without a client the same members throw, naming the option, over the socket and over a POST', async () => {
    const op = createTestOperator({ workflows, secret, console: quiet });
    const outcome = await op.deliver(reporter, {});

    expect(outcome).toMatchObject({ status: 'failed', retry: false, httpStatus: 101 });
    expect((outcome as { error: string }).error).toMatch(
      /ctx\.putStream is not available in the serverless runtime: configure `client` to enable this/
    );

    const posted = await createTestOperator({ workflows, secret, console: quiet, durable: false });
    expect(await posted.deliver(reporter, {})).toMatchObject({
      status: 'failed',
      retry: false,
      httpStatus: 422,
      error: expect.stringMatching(/ctx\.putStream .* configure `client` to enable this/),
    });

    expect(await op.deliver(extender, {})).toMatchObject({
      status: 'failed',
      retry: false,
      httpStatus: 101,
      error: expect.stringMatching(
        /ctx\.refreshTimeout is not available in the serverless runtime: configure `client` to enable this/
      ),
    });
  });

  it('a pure task is POSTed and a default task is dialed', async () => {
    const engine = fakeEngine();
    const op = createTestOperator({
      workflows,
      secret,
      console: quiet,
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
              output: JSON.stringify({ echo: 'dialed' }),
            },
          ],
        }),
      },
    });

    expect(await op.deliver(echo, { message: 'posted' })).toEqual({
      status: 'completed',
      output: { echo: 'posted' },
      httpStatus: 200,
    });
    expect(await op.deliver(parent, { message: 'dialed' })).toEqual({
      status: 'completed',
      output: { child: { echo: 'dialed' } },
      httpStatus: 101,
    });
  });

  it('a pure task has no client even when the handler does, and ctx.log stays on the console', async () => {
    const engine = fakeEngine();
    const out = { debug: vi.fn(), info: vi.fn(), warn: vi.fn(), error: vi.fn() };
    const op = createTestOperator({
      workflows,
      secret,
      console: out,
      client: { token: TOKEN, transport: engine.transport, logLevel: 'OFF' },
    });

    for (const member of [
      'ctx.runChild',
      'ctx.runNoWaitChild',
      'ctx.bulkRunChildren',
      'ctx.bulkRunNoWaitChildren',
      'ctx.spawnWorkflow',
      'ctx.spawnWorkflows',
      'ctx.putStream',
      'ctx.cancel',
      'ctx.refreshTimeout',
      'ctx.releaseSlot',
      'clientFromContext(ctx)',
    ]) {
      const outcome = await op.deliver(pureReporter, { member });

      // The SDK's deprecated `ctx.spawnWorkflow` and `ctx.spawnWorkflows` rewrap what they
      // catch as a `HatchetError`, so only the message carries through them; every other
      // member fails without retry.
      if (member === 'ctx.spawnWorkflow' || member === 'ctx.spawnWorkflows') {
        expect(outcome, member).toMatchObject({ status: 'failed' });
      } else {
        expect(outcome, member).toMatchObject({ status: 'failed', retry: false, httpStatus: 422 });
      }

      expect((outcome as { error: string }).error, member).toMatch(
        /is not available in the serverless runtime: this task is declared with `hatchetClient: false`, a function of its input with no Hatchet client\. Remove `hatchetClient: false`/
      );
      expect((outcome as { error: string }).error, member).toContain(
        member.startsWith('ctx.run') ||
          member.startsWith('ctx.bulk') ||
          member.startsWith('ctx.spawn')
          ? 'ctx.runChild'
          : member
      );
    }

    expect(await op.invoke(pureReporter, { member: 'ctx.log' })).toEqual({ logged: true });
    expect(out.warn).toHaveBeenCalledWith(
      expect.stringContaining('console only'),
      expect.anything()
    );
    expect(engine.logs).toEqual([]);
    expect(engine.triggers).toEqual([]);
    expect(engine.streamed).toEqual([]);
    expect(engine.cancels).toEqual([]);
    expect(engine.refreshes).toEqual([]);
    expect(engine.releases).toEqual([]);
  });

  it('runNoWaitChild over a POST (an adapter without the socket) triggers the child with the parent set, and its result() says why there is no socket', async () => {
    const engine = fakeEngine();
    const op = createTestOperator({
      workflows,
      secret,
      console: quiet,
      durable: false,
      client: { token: TOKEN, transport: engine.transport, logLevel: 'OFF' },
    });

    expect((await op.healthcheck()).tasks).toEqual([]);

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
      /invoked over a POST and has no invocation socket, which happens with an operator that predates stream delivery or an endpoint reporting no durable support/
    );
  });

  it('runChild awaits the child on one runs stream over the socket, the default delivery', async () => {
    const engine = fakeEngine();
    const op = createTestOperator({
      workflows,
      secret,
      console: quiet,
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

    // Every non-durable task but the pure ones; the durable one has a socket regardless.
    expect((await op.healthcheck()).tasks).toEqual([
      { action: 'extender:extender', streams: true },
      { action: 'fan-out:fan-out', streams: true },
      { action: 'parent:parent', streams: true },
      { action: 'reporter:reporter', streams: true },
      { action: 'spawner:spawner', streams: true },
    ]);

    // The catalog lists the task, so `invoke` takes the socket the way the operator does.
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

  it('a listed task delivered over a POST by an older operator still runs', async () => {
    const engine = fakeEngine();
    const op = createTestOperator({
      workflows,
      secret,
      console: quiet,
      client: { token: TOKEN, transport: engine.transport, logLevel: 'OFF' },
    });

    // The test operator dials listed tasks; a raw POST is what an older operator sends.
    const response = await op.request(`${op.handler.basePath}/healthcheck`);
    expect(response.status).toBe(200);
    expect((await op.healthcheck()).tasks).toContainEqual({
      action: 'spawner:spawner',
      streams: true,
    });
    const envelope = ServerlessTriggerRequest.toJSON(
      ServerlessTriggerRequest.fromPartial({
        endpointId: op.endpointId,
        timestampUnixSeconds: Math.floor(Date.now() / 1000),
        version: 1,
        action: {
          actionId: 'spawner:spawner',
          actionType: ActionType.START_STEP_RUN,
          actionPayload: JSON.stringify({ input: {}, parents: {}, triggered_by: 'manual' }),
          workflowRunId: 'parent-run',
          taskRunExternalId: 'task-run',
        },
      })
    );
    const posted = await op.request(`${op.handler.basePath}/trigger`, {
      body: JSON.stringify(envelope),
    });
    expect(posted.status).toBe(200);
    expect(await posted.json()).toEqual({ childRunId: 'run-1' });
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
    // A non-object environment has no identity to cache by: a client per request.
    expect(resolve(undefined)()).not.toBe(resolve(undefined)());
    // Once per source that was used: envA, envA again, envB, and undefined twice.
    expect(option).toHaveBeenCalledTimes(5);
  });

  it('never shares a credential between two primitive environments', () => {
    const tenantA = '11111111-1111-4111-8111-111111111111';
    const tenantB = '22222222-2222-4222-8222-222222222222';
    const resolve = createClientResolver((env) => ({
      token: makeToken({
        sub: env === 'tenant-a' ? tenantA : tenantB,
        grpc_broadcast_address: 'engine.example.com:7070',
      }),
      transport: fakeEngine().transport,
      logLevel: 'OFF' as const,
    }));

    const a = resolve('tenant-a')();
    const b = resolve('tenant-b')();

    expect(a?.tenantId).toBe(tenantA);
    expect(b?.tenantId).toBe(tenantB);
    expect(resolve('tenant-a')()).not.toBe(a);
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
      new ServerlessLimitationError('ctx.worker.labels', 'Nothing to do', 'there is no worker')
        .message
    ).toBe(
      'ctx.worker.labels is not available in the serverless runtime: there is no worker. Nothing to do. See LIMITATIONS.md in @hatchet-dev/serverless.'
    );
  });
});

describe('clientConfigFromEnv', () => {
  it('is no client without a token', () => {
    expect(clientConfigFromEnv(undefined)).toBeUndefined();
    expect(clientConfigFromEnv({})).toBeUndefined();
    expect(clientConfigFromEnv({ HATCHET_CLIENT_HOST_PORT: 'localhost:7070' })).toBeUndefined();
  });

  it('is the token alone when nothing else is set, so the token names the engine', () => {
    expect(clientConfigFromEnv({ HATCHET_CLIENT_TOKEN: 't' })).toEqual({ token: 't' });
    expect(
      clientConfigFromEnv({ HATCHET_CLIENT_TOKEN: 't', HATCHET_CLIENT_TLS_STRATEGY: 'tls' })
    ).toEqual({ token: 't' });
  });

  it('accepts the resolver forms the client supports in front of the host and port', () => {
    for (const hostPort of [
      'dns:///engine.example.com:7070',
      'dns:engine.example.com:7070',
      'ipv4:10.0.0.1:7070',
      'ipv6:[::1]:7070',
    ]) {
      expect(
        clientConfigFromEnv({ HATCHET_CLIENT_TOKEN: 't', HATCHET_CLIENT_HOST_PORT: hostPort })
      ).toEqual({ token: 't', hostPort });
    }
  });

  it('refuses a host and port carrying a path, query, fragment or user info', () => {
    for (const hostPort of ['engine.example/api', 'engine.example:7070/api', 'a@b:1', 'x:1?y']) {
      expect(() =>
        clientConfigFromEnv({ HATCHET_CLIENT_TOKEN: 't', HATCHET_CLIENT_HOST_PORT: hostPort })
      ).toThrow(/must be a host and port/);
    }
  });

  it('carries the host and the plain-HTTP choice for a local engine', () => {
    expect(
      clientConfigFromEnv({
        HATCHET_CLIENT_TOKEN: 't',
        HATCHET_CLIENT_HOST_PORT: 'localhost:7070',
        HATCHET_CLIENT_TLS_STRATEGY: 'none',
      })
    ).toEqual({ token: 't', hostPort: 'localhost:7070', tls: { strategy: 'none' } });
  });
});
