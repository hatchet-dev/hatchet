import { Code, ConnectError, createRouterTransport } from '@connectrpc/connect';
import type { MessageInitShape } from '@bufbuild/protobuf';
import {
  BulkTriggerWorkflowRequest,
  WorkflowService,
} from '@hatchet/protoc-es/workflows/workflows_pb';
import {
  TriggerWorkflowRequest,
  WorkerLabelComparator,
} from '@hatchet/protoc-es/v1/shared/trigger_pb';
import {
  AdminService,
  CancelTasksRequest,
  GetRunDetailsResponseSchema,
  IdempotencyCollisionErrorSchema,
  RunStatus,
} from '@hatchet/protoc-es/v1/workflows_pb';
import { EventsService, PushEventRequest } from '@hatchet/protoc-es/events/events_pb';
import { declarations } from '@hatchet/edge/declarations';
import { V1TaskStatus } from '@hatchet/clients/rest/generated/data-contracts';
import HatchetError from '@util/errors/hatchet-error';
import { IdempotencyCollisionError } from '@util/errors/idempotency-collision-error';
import { AbortError } from '@hatchet/util/abort-error';
import { HatchetCore } from './client';
import { INITIAL_POLL_INTERVAL_MS } from './run-ref';

type RunDetails = MessageInitShape<typeof GetRunDetailsResponseSchema>;

const TENANT_ID = '707d0855-80ab-4e1f-a156-f1c4546cbf52';

function makeToken(claims: Record<string, unknown>): string {
  const encode = (value: unknown) => Buffer.from(JSON.stringify(value)).toString('base64url');
  return `${encode({ alg: 'HS256', typ: 'JWT' })}.${encode(claims)}.signature`;
}

const TOKEN = makeToken({
  sub: TENANT_ID,
  grpc_broadcast_address: 'engine.example.com:7070',
  server_url: 'https://app.example.com',
});

const encodeJson = (value: unknown) => new TextEncoder().encode(JSON.stringify(value));

function taskRun(
  readableId: string,
  status: RunStatus,
  output?: unknown,
  error?: string
): NonNullable<RunDetails['taskRuns']>[string] {
  return {
    externalId: `${readableId}-id`,
    readableId,
    status,
    output: output === undefined ? undefined : encodeJson(output),
    error,
    isEvicted: false,
  };
}

/**
 * An in-memory engine: records every request and answers `GetRunDetails` from a queue, the
 * last entry of which is repeated once the queue is drained.
 */
function fakeEngine() {
  const triggers: TriggerWorkflowRequest[] = [];
  const bulkTriggers: BulkTriggerWorkflowRequest[] = [];
  const pushes: PushEventRequest[] = [];
  const cancels: CancelTasksRequest[] = [];
  const detailsQueue: RunDetails[] = [];
  const detailsTimeouts: Array<number | undefined> = [];
  let detailsCalls = 0;
  let detailsStalled = false;
  let triggerError: ConnectError | undefined;

  const transport = createRouterTransport(({ service }) => {
    service(WorkflowService, {
      triggerWorkflow: (req) => {
        if (triggerError) throw triggerError;
        triggers.push(req);
        return { workflowRunId: `run-${triggers.length}` };
      },
      bulkTriggerWorkflow: (req) => {
        bulkTriggers.push(req);
        const offset = bulkTriggers.slice(0, -1).reduce((n, b) => n + b.workflows.length, 0);
        return { workflowRunIds: req.workflows.map((_, i) => `bulk-${offset + i}`) };
      },
    });
    service(AdminService, {
      getRunDetails: (_req, ctx) => {
        detailsCalls += 1;
        detailsTimeouts.push(ctx.timeoutMs());
        if (detailsStalled) {
          // A stalled engine answers only when the call is abandoned, as a fetch aborted by
          // its signal does.
          return new Promise<RunDetails>((_, reject) => {
            ctx.signal.addEventListener('abort', () => reject(ctx.signal.reason), {
              once: true,
            });
          });
        }
        if (detailsQueue.length === 0) throw new Error('no run details queued');
        return detailsQueue.length > 1 ? detailsQueue.shift()! : detailsQueue[0];
      },
      cancelTasks: (req) => {
        cancels.push(req);
        return { cancelledTasks: req.externalIds };
      },
      replayTasks: (req) => ({ replayedTasks: req.externalIds }),
    });
    service(EventsService, {
      push: (req) => {
        pushes.push(req);
        return {
          tenantId: TENANT_ID,
          eventId: `evt-${pushes.length}`,
          key: req.key,
          payload: req.payload,
          eventTimestamp: req.eventTimestamp,
          additionalMetadata: req.additionalMetadata,
        };
      },
      bulkPush: (req) => ({
        events: req.events.map((e, i) => ({
          tenantId: TENANT_ID,
          eventId: `evt-${i}`,
          key: e.key,
          payload: e.payload,
          eventTimestamp: e.eventTimestamp,
        })),
      }),
    });
  });

  return {
    transport,
    triggers,
    bulkTriggers,
    pushes,
    cancels,
    detailsQueue,
    detailsTimeouts,
    get detailsCalls() {
      return detailsCalls;
    },
    failTriggersWith(error: ConnectError) {
      triggerError = error;
    },
    stallDetails() {
      detailsStalled = true;
    },
  };
}

function makeClient(engine: ReturnType<typeof fakeEngine>, namespace?: string) {
  return new HatchetCore({
    token: TOKEN,
    transport: engine.transport,
    namespace,
    logLevel: 'OFF',
    retrier: { maxAttempts: 1 },
  });
}

const completed = (taskRuns: RunDetails['taskRuns']): RunDetails => ({
  status: RunStatus.COMPLETED,
  done: true,
  input: encodeJson({}),
  additionalMetadata: new Uint8Array(),
  isEvicted: false,
  taskRuns,
});

const running: RunDetails = {
  status: RunStatus.RUNNING,
  done: false,
  input: encodeJson({}),
  additionalMetadata: new Uint8Array(),
  isEvicted: false,
  taskRuns: { echo: taskRun('echo', RunStatus.RUNNING) },
};

describe('HatchetCore configuration', () => {
  it('reads the engine address from the token when no address is configured', () => {
    const client = new HatchetCore({ token: TOKEN, logLevel: 'OFF' });
    expect(client.config.serverUrl).toBe('https://engine.example.com:7070');
    expect(client.tenantId).toBe(TENANT_ID);
  });

  it('builds the address from hostPort and the TLS strategy', () => {
    const client = new HatchetCore({
      token: TOKEN,
      hostPort: 'localhost:7070',
      tls: { strategy: 'none' },
      logLevel: 'OFF',
    });
    expect(client.config.serverUrl).toBe('http://localhost:7070');
  });

  it('takes serverUrl as given without a trailing slash', () => {
    const client = new HatchetCore({
      token: TOKEN,
      serverUrl: 'https://e.example/',
      logLevel: 'OFF',
    });
    expect(client.config.serverUrl).toBe('https://e.example');
  });

  it('normalizes the namespace the way the Node config loader does', () => {
    const client = new HatchetCore({ token: TOKEN, namespace: 'Prod', logLevel: 'OFF' });
    expect(client.config.namespace).toBe('prod_');
  });

  it('needs only sub and grpc_broadcast_address from the token', () => {
    const token = makeToken({ sub: TENANT_ID, grpc_broadcast_address: 'engine.example.com:7070' });
    const client = new HatchetCore({ token, logLevel: 'OFF' });
    expect(client.config.serverUrl).toBe('https://engine.example.com:7070');
    expect(client.tenantId).toBe(TENANT_ID);
  });

  it('takes hostPort when the token carries no address claims', () => {
    const client = new HatchetCore({
      token: makeToken({ sub: TENANT_ID }),
      hostPort: 'localhost:7070',
      tls: { strategy: 'none' },
      logLevel: 'OFF',
    });
    expect(client.config.serverUrl).toBe('http://localhost:7070');
  });

  it('rejects a missing token and names the claim and fields when no address is found', () => {
    expect(() => new HatchetCore({ token: '' })).toThrow(HatchetError);
    expect(() => new HatchetCore({ token: makeToken({ sub: TENANT_ID }) })).toThrow(
      /grpc_broadcast_address claim; set serverUrl or hostPort/
    );
  });

  it('takes the gRPC target forms the Node client takes for hostPort', () => {
    const forms: Array<[string, string]> = [
      ['dns:///engine.example.com:7070', 'http://engine.example.com:7070'],
      ['dns:engine.example.com:7070', 'http://engine.example.com:7070'],
      ['ipv4:127.0.0.1:7070', 'http://127.0.0.1:7070'],
      ['ipv6:[::1]:7070', 'http://[::1]:7070'],
      ['engine.example.com', 'http://engine.example.com:443'],
    ];
    for (const [hostPort, serverUrl] of forms) {
      const client = new HatchetCore({
        token: TOKEN,
        hostPort,
        tls: { strategy: 'none' },
        logLevel: 'OFF',
      });
      expect(client.config.serverUrl).toBe(serverUrl);
    }
    expect(
      () =>
        new HatchetCore({ token: TOKEN, hostPort: 'unix:/var/run/engine.sock', logLevel: 'OFF' })
    ).toThrow(/unix domain socket/);
  });

  it('refuses a token that cannot travel in a header without quoting it', () => {
    const token = `${TOKEN}\n`;
    expect(() => new HatchetCore({ token, logLevel: 'OFF' })).toThrow(
      /cannot be sent in an HTTP header/
    );
    try {
      new HatchetCore({ token, logLevel: 'OFF' });
    } catch (e) {
      expect((e as Error).message).not.toContain(TOKEN.split('.')[1]);
    }
  });
});

describe('HatchetCore.runNoWait', () => {
  it('sends the namespaced, lowercased name and JSON input', async () => {
    const engine = fakeEngine();
    const client = makeClient(engine, 'Prod');

    const ref = await client.runNoWait('MyWorkflow', { hello: 'world' });

    expect(ref.workflowRunId).toBe('run-1');
    expect(engine.triggers).toHaveLength(1);
    const [req] = engine.triggers;
    expect(req.name).toBe('prod_myworkflow');
    expect(req.input).toBe(JSON.stringify({ hello: 'world' }));
    expect(req.additionalMetadata).toBeUndefined();
    expect(req.desiredWorkerLabels).toEqual({});
  });

  it('maps the options onto the trigger request the way the Node admin client does', async () => {
    const engine = fakeEngine();
    const client = makeClient(engine);

    await client.runNoWait(
      'wf',
      {},
      {
        additionalMetadata: { source: 'test' },
        priority: 3,
        parentStepRunId: 'parent-task',
        parentId: 'parent-run',
        childIndex: 2,
        childKey: 'child',
        desiredWorkerId: 'worker-1',
        desiredWorkerLabels: {
          region: { value: 'us-east', required: true },
          cores: { value: 8, weight: 2, comparator: WorkerLabelComparator.GREATER_THAN },
        },
      }
    );

    const [req] = engine.triggers;
    expect(req.additionalMetadata).toBe(JSON.stringify({ source: 'test' }));
    expect(req.priority).toBe(3);
    expect(req.parentTaskRunExternalId).toBe('parent-task');
    expect(req.parentId).toBe('parent-run');
    expect(req.childIndex).toBe(2);
    expect(req.childKey).toBe('child');
    expect(req.desiredWorkerId).toBe('worker-1');
    expect(req.desiredWorkerLabels.region).toMatchObject({ strValue: 'us-east', required: true });
    expect(req.desiredWorkerLabels.cores).toMatchObject({
      intValue: 8,
      weight: 2,
      comparator: WorkerLabelComparator.GREATER_THAN,
    });
  });

  it('accepts a declaration from the edge entry and unwraps a task output', async () => {
    const engine = fakeEngine();
    const client = makeClient(engine, 'ns');
    const { task } = declarations();
    const echo = task({ name: 'Echo', fn: (input: { message: string }) => input });
    engine.detailsQueue.push(
      completed({ Echo: taskRun('Echo', RunStatus.COMPLETED, { message: 'hi' }) })
    );

    const output = await client.run(echo, { message: 'hi' });

    expect(engine.triggers[0].name).toBe('ns_echo');
    expect(output).toEqual({ message: 'hi' });
  });

  it('raises an IdempotencyCollisionError from the ALREADY_EXISTS details', async () => {
    const engine = fakeEngine();
    engine.failTriggersWith(
      new ConnectError('collision', Code.AlreadyExists, undefined, [
        {
          desc: IdempotencyCollisionErrorSchema,
          value: { existingRunExternalId: 'existing-run', collidingRunExternalId: '' },
        },
      ])
    );
    const client = makeClient(engine);

    await expect(client.runNoWait('wf', {})).rejects.toThrow(IdempotencyCollisionError);
    await expect(client.runNoWait('wf', {})).rejects.toMatchObject({
      existingRunExternalId: 'existing-run',
    });
  });
});

describe('HatchetCore.runManyNoWait', () => {
  it('bulk-triggers in input order with per-run options', async () => {
    const engine = fakeEngine();
    const client = makeClient(engine, 'ns');

    const refs = await client.runManyNoWait('Wf', [
      { input: { i: 1 } },
      { input: { i: 2 }, opts: { additionalMetadata: { k: 'v' }, priority: 1 } },
    ]);

    expect(refs.map((r) => r.workflowRunId)).toEqual(['bulk-0', 'bulk-1']);
    expect(engine.bulkTriggers).toHaveLength(1);
    const [{ workflows }] = engine.bulkTriggers;
    expect(workflows.map((w) => w.name)).toEqual(['ns_wf', 'ns_wf']);
    expect(workflows[1].input).toBe(JSON.stringify({ i: 2 }));
    expect(workflows[1].additionalMetadata).toBe(JSON.stringify({ k: 'v' }));
    expect(workflows[1].priority).toBe(1);
    expect(workflows[0].priority).toBeUndefined();
  });

  it('splits more than 500 runs into batches and keeps the order', async () => {
    const engine = fakeEngine();
    const client = makeClient(engine);

    const refs = await client.runManyNoWait(
      'wf',
      Array.from({ length: 1001 }, (_, i) => ({ input: { i } }))
    );

    expect(engine.bulkTriggers.map((b) => b.workflows.length)).toEqual([500, 500, 1]);
    expect(refs).toHaveLength(1001);
    expect(refs[1000].workflowRunId).toBe('bulk-1000');
  });
});

describe('WorkflowRunRef.result', () => {
  beforeEach(() => {
    jest.useFakeTimers();
  });

  afterEach(() => {
    jest.useRealTimers();
  });

  it('polls until the run is done and returns the outputs keyed by task', async () => {
    const engine = fakeEngine();
    const client = makeClient(engine);
    engine.detailsQueue.push(
      running,
      running,
      completed({
        step1: taskRun('step1', RunStatus.COMPLETED, { a: 1 }),
        step2: taskRun('step2', RunStatus.COMPLETED),
      })
    );

    const ref = await client.runNoWait('wf', {});
    const result = ref.result();
    // Two not-done answers, so two waits: the first interval and the doubled one.
    await jest.advanceTimersByTimeAsync(INITIAL_POLL_INTERVAL_MS * 1.2);
    await jest.advanceTimersByTimeAsync(INITIAL_POLL_INTERVAL_MS * 2.4);

    await expect(result).resolves.toEqual({ step1: { a: 1 }, step2: {} });
    expect(engine.detailsCalls).toBe(3);
  });

  it('rejects a failed run with the task error messages, as the Node client does', async () => {
    const engine = fakeEngine();
    const client = makeClient(engine);
    engine.detailsQueue.push({
      ...completed({
        ok: taskRun('ok', RunStatus.COMPLETED, {}),
        bad: taskRun('bad', RunStatus.FAILED, undefined, 'boom'),
      }),
      status: RunStatus.FAILED,
    });

    const ref = await client.runNoWait('wf', {});
    await expect(ref.result()).rejects.toEqual(['boom']);
  });

  it('rejects a cancelled run with an Error when no task reported one', async () => {
    const engine = fakeEngine();
    const client = makeClient(engine);
    engine.detailsQueue.push({
      ...completed({ t: taskRun('t', RunStatus.CANCELLED) }),
      status: RunStatus.CANCELLED,
    });

    const ref = await client.runNoWait('wf', {});
    await expect(ref.result()).rejects.toThrow(/run run-1 was cancelled/);
  });

  it('times out after timeoutMs with a HatchetError', async () => {
    const engine = fakeEngine();
    const client = makeClient(engine);
    engine.detailsQueue.push(running);

    const ref = await client.runNoWait('wf', {});
    const result = ref.result({ timeoutMs: 1_000 });
    const assertion = expect(result).rejects.toThrow(HatchetError);
    await jest.advanceTimersByTimeAsync(1_100);

    await assertion;
    await expect(result).rejects.toThrow(/timed out after 1000 ms/);
    expect(engine.detailsCalls).toBeGreaterThanOrEqual(2);
  });

  it('stops waiting with an AbortError when the signal fires', async () => {
    const engine = fakeEngine();
    const client = makeClient(engine);
    engine.detailsQueue.push(running);
    const controller = new AbortController();

    const ref = await client.runNoWait('wf', {});
    const result = ref.result({ signal: controller.signal });
    const assertion = expect(result).rejects.toThrow(AbortError);
    await jest.advanceTimersByTimeAsync(10);
    const callsBeforeAbort = engine.detailsCalls;
    controller.abort();
    await jest.advanceTimersByTimeAsync(10_000);

    await assertion;
    expect(engine.detailsCalls).toBe(callsBeforeAbort);
  });

  it('times out a poll in flight at the deadline and passes it to the call', async () => {
    const engine = fakeEngine();
    const client = makeClient(engine);
    engine.stallDetails();

    const ref = await client.runNoWait('wf', {});
    const result = ref.result({ timeoutMs: 1_000 });
    const assertion = expect(result).rejects.toThrow(HatchetError);
    await jest.advanceTimersByTimeAsync(1_100);

    await assertion;
    await expect(result).rejects.toThrow(/timed out after 1000 ms/);
    expect(engine.detailsCalls).toBe(1);
    expect(engine.detailsTimeouts[0]).toBeGreaterThan(0);
    expect(engine.detailsTimeouts[0]).toBeLessThanOrEqual(1_000);
  });

  it('aborts a poll in flight with an AbortError when the signal fires', async () => {
    const engine = fakeEngine();
    const client = makeClient(engine);
    engine.stallDetails();
    const controller = new AbortController();

    const ref = await client.runNoWait('wf', {});
    const result = ref.result({ signal: controller.signal, timeoutMs: 60_000 });
    const assertion = expect(result).rejects.toThrow(AbortError);
    await jest.advanceTimersByTimeAsync(10);
    controller.abort();
    await jest.advanceTimersByTimeAsync(10);

    await assertion;
    expect(engine.detailsCalls).toBe(1);
  });
});

describe('HatchetCore.runs', () => {
  it('rejects an aborted unary call before sending it', async () => {
    const engine = fakeEngine();
    const client = makeClient(engine);
    engine.detailsQueue.push(running);
    const controller = new AbortController();
    controller.abort();

    await expect(client.runs.get('run-x', { signal: controller.signal })).rejects.toMatchObject({
      name: 'AbortError',
    });
    expect(engine.detailsCalls).toBe(0);
  });

  it('returns run details with statuses mapped and payloads decoded', async () => {
    const engine = fakeEngine();
    const client = makeClient(engine);
    engine.detailsQueue.push({
      ...completed({ t: taskRun('t', RunStatus.COMPLETED, { out: true }) }),
      input: encodeJson({ in: 1 }),
      additionalMetadata: encodeJson({ meta: 'x' }),
    });

    const detail = await client.runs.get('run-x');

    expect(detail.status).toBe(V1TaskStatus.COMPLETED);
    expect(detail.done).toBe(true);
    expect(detail.input).toEqual({ in: 1 });
    expect(detail.additionalMetadata).toEqual({ meta: 'x' });
    expect(detail.taskRuns.t).toMatchObject({
      externalId: 't-id',
      status: V1TaskStatus.COMPLETED,
      output: { out: true },
    });
  });

  it('cancels by id, and by filter with since defaulting to an hour ago', async () => {
    const engine = fakeEngine();
    const client = makeClient(engine);

    await client.runs.cancel({ ids: ['a', 'b'] });
    expect(engine.cancels[0].externalIds).toEqual(['a', 'b']);
    expect(engine.cancels[0].filter).toBeUndefined();

    const before = Date.now();
    await client.runs.cancel({
      filters: { statuses: [V1TaskStatus.RUNNING], additionalMetadata: { k: 'v' } },
    });
    const [, { filter, externalIds }] = engine.cancels;
    expect(externalIds).toEqual([]);
    expect(filter?.statuses).toEqual(['RUNNING']);
    expect(filter?.additionalMetadata).toEqual(['k:v']);
    const since = Number(filter?.since?.seconds) * 1000;
    expect(before - since).toBeGreaterThanOrEqual(60 * 60 * 1000 - 1000);
    expect(before - since).toBeLessThan(60 * 60 * 1000 + 5000);
  });
});

describe('HatchetCore.events', () => {
  it('pushes a namespaced event with a JSON payload and metadata', async () => {
    const engine = fakeEngine();
    const client = makeClient(engine, 'ns');

    const event = await client.events.push(
      'user:created',
      { id: 1 },
      { additionalMetadata: { source: 'test' }, priority: 2, scope: 'tenant-a' }
    );

    const [req] = engine.pushes;
    expect(req.key).toBe('ns_user:created');
    expect(req.payload).toBe(JSON.stringify({ id: 1 }));
    expect(req.additionalMetadata).toBe(JSON.stringify({ source: 'test' }));
    expect(req.priority).toBe(2);
    expect(req.scope).toBe('tenant-a');
    expect(event.eventId).toBe('evt-1');
    expect(event.key).toBe('ns_user:created');
    expect(event.eventTimestamp).toBeInstanceOf(Date);
  });

  it('bulk-pushes events with per-event overrides', async () => {
    const engine = fakeEngine();
    const client = makeClient(engine);

    const events = await client.events.bulkPush(
      'batch',
      [{ payload: { n: 1 } }, { payload: { n: 2 }, priority: 3, scope: 'b' }],
      { priority: 1 }
    );

    expect(events.events.map((e) => e.key)).toEqual(['batch', 'batch']);
    expect(events.events[1].payload).toBe(JSON.stringify({ n: 2 }));
  });
});
