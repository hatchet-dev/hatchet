import HatchetError from '@util/errors/hatchet-error';
import {
  V1CELDebugResponseStatus,
  V1TaskStatus,
} from '@hatchet/clients/rest/generated/data-contracts';
import { HatchetCore } from './client';

const TENANT_ID = '707d0855-80ab-4e1f-a156-f1c4546cbf52';
const WORKFLOW_ID = '2c9f0f2e-5d6c-4a1f-9f0e-3b1c4d5e6f70';
const CRON_ID = '3d9f0f2e-5d6c-4a1f-9f0e-3b1c4d5e6f71';
const TASK_ID = '4e9f0f2e-5d6c-4a1f-9f0e-3b1c4d5e6f72';

function makeToken(claims: Record<string, unknown>): string {
  const encode = (value: unknown) => Buffer.from(JSON.stringify(value)).toString('base64url');
  return `${encode({ alg: 'HS256', typ: 'JWT' })}.${encode(claims)}.signature`;
}

const TOKEN = makeToken({
  sub: TENANT_ID,
  grpc_broadcast_address: 'engine.example.com:7070',
  server_url: 'https://app.example.com',
});
const TOKEN_WITHOUT_SERVER_URL = makeToken({
  sub: TENANT_ID,
  grpc_broadcast_address: 'engine.example.com:7070',
});

type Recorded = { url: URL; method: string; headers: Headers; body: unknown };
type Reply = { status: number; json?: unknown; text?: string };

/** A fake API server: records every request and answers from a queue, `{}` once it is drained. */
function fakeApi() {
  const requests: Recorded[] = [];
  const replies: Reply[] = [];

  const fetch: typeof globalThis.fetch = async (input, init) => {
    const raw = typeof input === 'string' ? input : input instanceof URL ? input.href : input.url;
    const body = typeof init?.body === 'string' ? JSON.parse(init.body) : (init?.body ?? null);
    requests.push({
      url: new URL(raw),
      method: init?.method ?? 'GET',
      headers: new Headers(init?.headers),
      body,
    });

    const reply = replies.shift() ?? { status: 200, json: {} };
    if (reply.status === 204) {
      return new Response(null, { status: 204 });
    }
    if (reply.text !== undefined) {
      return new Response(reply.text, {
        status: reply.status,
        headers: { 'content-type': 'text/plain' },
      });
    }
    return new Response(JSON.stringify(reply.json), {
      status: reply.status,
      headers: { 'content-type': 'application/json' },
    });
  };

  return {
    fetch,
    requests,
    reply(json: unknown, status = 200) {
      replies.push({ status, json });
    },
    replyText(text: string) {
      replies.push({ status: 200, text });
    },
  };
}

const workflowRow = { metadata: { id: WORKFLOW_ID }, name: 'greet' };

describe('core REST base URL', () => {
  it("strips trailing slashes from the token's server_url claim", async () => {
    const api = fakeApi();
    const client = new HatchetCore({
      token: makeToken({ sub: TENANT_ID, server_url: 'https://app.example.com///' }),
      fetch: api.fetch,
    });

    await client.api.infoGetVersion();

    expect(client.config.apiUrl).toBe('https://app.example.com');
    expect(api.requests[0].url.href).toBe('https://app.example.com/api/v1/version');
  });

  it("applies the client's namespace to workflow lookups and leaves a prefixed name alone", async () => {
    const api = fakeApi();
    api.reply({ rows: [{ metadata: { id: 'wf-1' }, name: 'prod_greet' }] });
    const client = new HatchetCore({ token: TOKEN, namespace: 'prod', fetch: api.fetch });

    const byShortName = await client.workflows.get('greet');
    // the prefixed name resolves to the same cached entry, so nothing else is fetched
    const byFullName = await client.workflows.get('prod_greet');

    expect(byShortName.metadata.id).toBe('wf-1');
    expect(byFullName.metadata.id).toBe('wf-1');
    expect(api.requests).toHaveLength(1);
    expect(api.requests[0].url.searchParams.get('name')).toBe('prod_greet');
  });

  it("defaults to the token's server_url claim and sends the token as a bearer", async () => {
    const api = fakeApi();
    const client = new HatchetCore({ token: TOKEN, fetch: api.fetch });
    api.reply({ rows: [] });

    expect(client.config.apiUrl).toBe('https://app.example.com');
    await client.workers.list();

    expect(api.requests[0].url.href).toBe(
      `https://app.example.com/api/v1/tenants/${TENANT_ID}/worker`
    );
    expect(api.requests[0].headers.get('authorization')).toBe(`Bearer ${TOKEN}`);
  });

  it('takes an explicit apiUrl over the claim, without its trailing slash', async () => {
    const api = fakeApi();
    const client = new HatchetCore({
      token: TOKEN,
      apiUrl: 'http://localhost:8080/',
      fetch: api.fetch,
    });

    expect(client.config.apiUrl).toBe('http://localhost:8080');
    await client.tenant.get();
    expect(api.requests[0].url.href).toBe(`http://localhost:8080/api/v1/tenants/${TENANT_ID}`);
  });

  it('rejects an apiUrl that is not an absolute http or https URL', () => {
    expect(() => new HatchetCore({ token: TOKEN, apiUrl: 'localhost:8080' })).toThrow(HatchetError);
    expect(() => new HatchetCore({ token: TOKEN, apiUrl: 'ftp://files.example.com' })).toThrow(
      'apiUrl must be an absolute http:// or https:// URL'
    );
  });

  it('leaves apiUrl unset for a token without the claim and fails REST calls naming it', async () => {
    const api = fakeApi();
    const client = new HatchetCore({ token: TOKEN_WITHOUT_SERVER_URL, fetch: api.fetch });

    expect(client.config.apiUrl).toBeUndefined();
    await expect(client.workers.list()).rejects.toThrow(HatchetError);
    await expect(client.workers.list()).rejects.toThrow(/set apiUrl on the client config/);
    expect(api.requests).toHaveLength(0);
  });

  it('rejects with the response when the API answers with an error status', async () => {
    const api = fakeApi();
    const client = new HatchetCore({ token: TOKEN, fetch: api.fetch });
    api.reply({ errors: [{ description: 'no such tenant' }] }, 404);

    await expect(client.tenant.get()).rejects.toMatchObject({
      status: 404,
      error: { errors: [{ description: 'no such tenant' }] },
    });
  });
});

describe('core REST features', () => {
  it('runs.list resolves workflow names and sends the Node list query', async () => {
    const api = fakeApi();
    const client = new HatchetCore({ token: TOKEN, fetch: api.fetch });
    const rows = [{ metadata: { id: 'run-1' }, status: V1TaskStatus.COMPLETED }];
    api.reply({ rows: [workflowRow] });
    api.reply({ rows, pagination: {} });

    const result = await client.runs.list({
      workflowNames: ['greet'],
      onlyTasks: true,
      limit: 5,
      since: new Date('2026-10-01T00:00:00Z'),
      additionalMetadata: { env: 'prod' },
    });

    expect(result.rows).toEqual(rows);
    const [lookup, list] = api.requests;
    expect(lookup.url.pathname).toBe(`/api/v1/tenants/${TENANT_ID}/workflows`);
    expect(lookup.url.searchParams.get('name')).toBe('greet');
    expect(list.url.pathname).toBe(`/api/v1/stable/tenants/${TENANT_ID}/workflow-runs`);
    expect(Object.fromEntries(list.url.searchParams)).toEqual({
      since: '2026-10-01T00:00:00.000Z',
      limit: '5',
      workflow_ids: WORKFLOW_ID,
      additional_metadata: 'env:prod',
      only_tasks: 'true',
    });
  });

  it('runs.list defaults since to one hour ago', async () => {
    const api = fakeApi();
    const client = new HatchetCore({ token: TOKEN, fetch: api.fetch });
    api.reply({ rows: [], pagination: {} });

    const before = Date.now();
    await client.runs.list();
    const since = new Date(api.requests[0].url.searchParams.get('since')!).getTime();

    expect(since).toBeGreaterThanOrEqual(before - 60 * 60 * 1000 - 1000);
    expect(since).toBeLessThanOrEqual(Date.now() - 60 * 60 * 1000);
    expect(api.requests[0].url.searchParams.get('only_tasks')).toBe('false');
  });

  it('runs.restoreTask and runs.branchDurableTask post to the durable task endpoints', async () => {
    const api = fakeApi();
    const client = new HatchetCore({ token: TOKEN, fetch: api.fetch });
    api.reply({ taskExternalId: TASK_ID });
    api.reply({ workflowRunId: 'run-2' });

    await client.runs.restoreTask(TASK_ID);
    await client.runs.branchDurableTask(TASK_ID, 3);

    expect(api.requests[0].method).toBe('POST');
    expect(api.requests[0].url.pathname).toBe(`/api/v1/stable/tasks/${TASK_ID}/restore`);
    expect(api.requests[1].url.pathname).toBe(
      `/api/v1/stable/tenants/${TENANT_ID}/durable-tasks/branch`
    );
    expect(api.requests[1].body).toEqual({ taskExternalId: TASK_ID, nodeId: 3, branchId: 0 });
  });

  it('crons.create namespaces the workflow and posts the trigger', async () => {
    const api = fakeApi();
    const client = new HatchetCore({ token: TOKEN, namespace: 'Prod', fetch: api.fetch });
    const created = { metadata: { id: CRON_ID }, cron: '0 0 * * *', name: 'nightly' };
    api.reply(created);

    const cron = await client.crons.create('greet', {
      name: 'nightly',
      expression: '0 0 * * *',
      input: { name: 'world' },
      priority: 2,
    });

    expect(cron).toEqual(created);
    expect(api.requests[0].method).toBe('POST');
    expect(api.requests[0].url.pathname).toBe(
      `/api/v1/tenants/${TENANT_ID}/workflows/prod_greet/crons`
    );
    expect(api.requests[0].body).toEqual({
      cronName: 'nightly',
      cronExpression: '0 0 * * *',
      input: { name: 'world' },
      additionalMetadata: {},
      priority: 2,
    });
  });

  it("crons.create surfaces the API's errors the way the Node client does", async () => {
    const api = fakeApi();
    const client = new HatchetCore({ token: TOKEN, fetch: api.fetch });
    api.reply({ errors: [{ description: 'invalid cron expression' }] }, 400);

    await expect(client.crons.create('greet', { name: 'bad', expression: 'nope' })).rejects.toThrow(
      JSON.stringify([{ description: 'invalid cron expression' }])
    );
  });

  it('crons.get and crons.delete address the cron by id', async () => {
    const api = fakeApi();
    const client = new HatchetCore({ token: TOKEN, fetch: api.fetch });
    api.reply({ metadata: { id: CRON_ID } });
    api.reply({}, 204);

    await client.crons.get(CRON_ID);
    await client.crons.delete({ metadata: { id: CRON_ID } } as never);

    expect(api.requests[0].url.pathname).toBe(
      `/api/v1/tenants/${TENANT_ID}/workflows/crons/${CRON_ID}`
    );
    expect(api.requests[1].method).toBe('DELETE');
    await expect(client.crons.get('not-a-uuid')).rejects.toThrow('Invalid cron ID');
  });

  it('schedules.create posts the trigger time as ISO 8601', async () => {
    const api = fakeApi();
    const client = new HatchetCore({ token: TOKEN, fetch: api.fetch });
    const created = { metadata: { id: CRON_ID }, triggerAt: '2026-11-01T00:00:00.000Z' };
    api.reply(created);

    const scheduled = await client.schedules.create('greet', {
      triggerAt: new Date('2026-11-01T00:00:00Z'),
      input: { name: 'world' },
    });

    expect(scheduled).toEqual(created);
    expect(api.requests[0].url.pathname).toBe(
      `/api/v1/tenants/${TENANT_ID}/workflows/greet/scheduled`
    );
    expect(api.requests[0].body).toEqual({
      input: { name: 'world' },
      additionalMetadata: {},
      triggerAt: '2026-11-01T00:00:00.000Z',
    });
  });

  it('schedules.list resolves the workflow filter by name', async () => {
    const api = fakeApi();
    const client = new HatchetCore({ token: TOKEN, fetch: api.fetch });
    api.reply({ rows: [workflowRow] });
    api.reply({ rows: [], pagination: {} });

    await client.schedules.list({ workflow: 'greet', limit: 10 });

    expect(api.requests[1].url.pathname).toBe(`/api/v1/tenants/${TENANT_ID}/workflows/scheduled`);
    expect(Object.fromEntries(api.requests[1].url.searchParams)).toEqual({
      workflowId: WORKFLOW_ID,
      limit: '10',
    });
  });

  it('workers.list, get and pause use the worker endpoints', async () => {
    const api = fakeApi();
    const client = new HatchetCore({ token: TOKEN, fetch: api.fetch });
    const worker = { metadata: { id: 'worker-1' }, name: 'w', status: 'PAUSED' };
    api.reply({ rows: [worker] });
    api.reply(worker);
    api.reply(worker);

    const list = await client.workers.list();
    const paused = await client.workers.isPaused('worker-1');
    await client.workers.pause('worker-1');

    expect(list.rows).toEqual([worker]);
    expect(paused).toBe(true);
    expect(api.requests[1].url.pathname).toBe('/api/v1/workers/worker-1');
    expect(api.requests[2].method).toBe('PATCH');
    expect(api.requests[2].body).toEqual({ isPaused: true });
  });

  it('metrics reduce task status counts and read Prometheus metrics as text', async () => {
    const api = fakeApi();
    const client = new HatchetCore({ token: TOKEN, fetch: api.fetch });
    api.reply([
      { status: 'QUEUED', count: 1 },
      { status: 'RUNNING', count: 2 },
      { status: 'COMPLETED', count: 3 },
      { status: 'FAILED', count: 0 },
      { status: 'CANCELLED', count: 0 },
    ]);
    api.replyText('# HELP hatchet_queue_depth\nhatchet_queue_depth 4\n');

    const metrics = await client.metrics.getTaskStatusMetrics({
      since: '2026-10-01T00:00:00Z',
      workflow_ids: [WORKFLOW_ID],
    });
    const prometheus = await client.metrics.scrapePrometheusMetrics();

    expect(metrics).toEqual({ queued: 1, running: 2, completed: 3, failed: 0, cancelled: 0 });
    expect(api.requests[0].url.pathname).toBe(`/api/v1/stable/tenants/${TENANT_ID}/task-metrics`);
    expect(api.requests[0].url.searchParams.get('workflow_ids')).toBe(WORKFLOW_ID);
    expect(prometheus).toBe('# HELP hatchet_queue_depth\nhatchet_queue_depth 4\n');
  });

  it('filters, webhooks and the tenant go to their stable endpoints', async () => {
    const api = fakeApi();
    const client = new HatchetCore({ token: TOKEN, fetch: api.fetch });
    api.reply({ rows: [], pagination: {} });
    api.reply({ name: 'github' });
    api.reply({ metadata: { id: TENANT_ID }, name: 'acme', version: 'V1' });

    await client.filters.list({ workflowIds: [WORKFLOW_ID, CRON_ID], scopes: ['s'] });
    const webhook = await client.webhooks.get('github');
    const tenant = await client.tenant.get();

    expect(api.requests[0].url.pathname).toBe(`/api/v1/stable/tenants/${TENANT_ID}/filters`);
    expect(api.requests[0].url.searchParams.getAll('workflowIds')).toEqual([WORKFLOW_ID, CRON_ID]);
    expect(webhook).toEqual({ name: 'github' });
    expect(api.requests[1].url.pathname).toBe(
      `/api/v1/stable/tenants/${TENANT_ID}/webhooks/github`
    );
    expect(tenant.name).toBe('acme');
  });

  it('cel.debug reports a successful evaluation', async () => {
    const api = fakeApi();
    const client = new HatchetCore({ token: TOKEN, fetch: api.fetch });
    api.reply({ status: V1CELDebugResponseStatus.SUCCESS, output: true });

    const result = await client.cel.debug({ expression: 'input.x > 1', input: { x: 2 } });

    expect(result).toEqual({ status: V1CELDebugResponseStatus.SUCCESS, output: true });
    expect(api.requests[0].url.pathname).toBe(`/api/v1/stable/tenants/${TENANT_ID}/cel/debug`);
    expect(api.requests[0].body).toEqual({ expression: 'input.x > 1', input: { x: 2 } });
  });

  it('workflows.get caches by name and workflows.pause patches the workflow', async () => {
    const api = fakeApi();
    const client = new HatchetCore({ token: TOKEN, fetch: api.fetch });
    api.reply({ rows: [workflowRow] });
    api.reply({ ...workflowRow, isPaused: true });

    const first = await client.workflows.get('greet');
    const second = await client.workflows.get('greet');
    await client.workflows.pause('greet', { queueTTL: '1h' });

    expect(first).toEqual(workflowRow);
    expect(second).toBe(first);
    expect(api.requests).toHaveLength(2);
    expect(api.requests[1].method).toBe('PATCH');
    expect(api.requests[1].url.pathname).toBe(`/api/v1/workflows/${WORKFLOW_ID}`);
    expect(api.requests[1].body).toMatchObject({
      pause: { action: 'pause', pausedWorkflowQueueTTL: '1h' },
    });
  });

  it('the generated client is exposed as api for everything the features do not wrap', async () => {
    const api = fakeApi();
    const client = new HatchetCore({ token: TOKEN, fetch: api.fetch });
    api.reply({ version: 'v0.80.0' });

    const response = await client.api.infoGetVersion();

    expect(response.data).toEqual({ version: 'v0.80.0' });
    expect(api.requests[0].url.pathname).toBe('/api/v1/version');
  });
});
