import {
  createServer as createHttpServer,
  request,
  type IncomingMessage,
  type Server,
} from 'node:http';
import { connect, type AddressInfo } from 'node:net';
import { WebSocket } from 'ws';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import { ActionType, AssignedAction } from '../generated/proto/dispatcher';
import { ServerlessTriggerRequest } from '../generated/proto/v1/serverless';
import {
  SIGNATURE_HEADER,
  TRIGGER_ENVELOPE_VERSION,
  clientFromContext,
  hatchet,
  signHex,
} from '../index';
import { decodeFrame, encodeFrame } from '../index';
import { createTestOperator } from '../testing';
import { createServer, nodeHandler } from './node';

const secret = 'test-secret-at-least-32-characters-long';

const echo = hatchet.task({
  name: 'echo',
  fn: async (input: { message: string }) => ({ echo: input.message }),
});

// Answers with the tenant of the client the adapter built, so the default is observable.
const whoami = hatchet.task({
  name: 'whoami',
  fn: async (_input: {}, ctx) => ({ tenant: clientFromContext(ctx).tenantId }),
});

const TENANT_ID = '707d0855-80ab-4e1f-a156-f1c4546cbf52';
const TOKEN = `${Buffer.from('{"alg":"HS256"}').toString('base64url')}.${Buffer.from(
  JSON.stringify({ sub: TENANT_ID, grpc_broadcast_address: 'engine.example.com:7070' })
).toString('base64url')}.sig`;

const durableEcho = hatchet.durableTask({
  name: 'durable-echo',
  fn: async (input: { message: string }) => ({ echo: input.message }),
});

// A task that holds its invocation open until the test releases it, exposing its signal.
const blocked = { signals: [] as AbortSignal[], release: () => {} };

const blocker = hatchet.task({
  name: 'blocker',
  fn: async (_input: {}, ctx) => {
    blocked.signals.push(ctx.abortController.signal);
    await new Promise<void>((resolve) => {
      blocked.release = resolve;
    });
    return {};
  },
});

const workflows = [echo, durableEcho];

const quiet = { debug: vi.fn(), info: vi.fn(), warn: vi.fn(), error: vi.fn() };

async function signed(body: string) {
  return { [SIGNATURE_HEADER]: await signHex(secret, body), 'content-type': 'application/json' };
}

function healthcheckBody() {
  return JSON.stringify({
    endpointId: 'e',
    timestampUnixSeconds: String(Math.floor(Date.now() / 1000)),
  });
}

function triggerBody(input: unknown, name = 'echo') {
  const action = AssignedAction.fromPartial({
    tenantId: crypto.randomUUID(),
    workflowRunId: crypto.randomUUID(),
    jobId: crypto.randomUUID(),
    jobName: name,
    jobRunId: crypto.randomUUID(),
    taskId: crypto.randomUUID(),
    taskRunExternalId: crypto.randomUUID(),
    actionId: `${name}:${name}`,
    actionType: ActionType.START_STEP_RUN,
    actionPayload: JSON.stringify({ input, parents: {}, triggered_by: 'manual' }),
    taskName: name,
    retryCount: 0,
    priority: 1,
  });

  return JSON.stringify(
    ServerlessTriggerRequest.toJSON({
      endpointId: 'e',
      action,
      timestampUnixSeconds: Math.floor(Date.now() / 1000),
      version: TRIGGER_ENVELOPE_VERSION,
    })
  );
}

function firstFrame(taskRunExternalId: string) {
  const action = AssignedAction.fromPartial({
    tenantId: crypto.randomUUID(),
    workflowRunId: crypto.randomUUID(),
    jobId: crypto.randomUUID(),
    jobName: 'durable-echo',
    jobRunId: crypto.randomUUID(),
    taskId: crypto.randomUUID(),
    taskRunExternalId,
    actionId: 'durable-echo:durable-echo',
    actionType: ActionType.START_STEP_RUN,
    actionPayload: JSON.stringify({
      input: { message: 'hi' },
      parents: {},
      triggered_by: 'manual',
    }),
    taskName: 'durable-echo',
    retryCount: 0,
    priority: 1,
    durableTaskInvocationCount: 1,
  });

  return encodeFrame({ first: { action, invocationCount: 1, inlineWaitBudgetMs: 5000 } });
}

function listen(server: Server): Promise<string> {
  return new Promise((resolve) => {
    server.listen(0, '127.0.0.1', () => {
      const { port } = server.address() as AddressInfo;
      resolve(`127.0.0.1:${port}`);
    });
  });
}

/** Writes one HTTP/1.1 request as given and resolves with everything the server sent back. */
function rawRequest(host: string, text: string): Promise<string> {
  const [address, port] = host.split(':');

  return new Promise((resolve, reject) => {
    const socket = connect(Number(port), address, () => socket.write(text));
    let data = '';

    socket.on('data', (chunk) => {
      data += chunk.toString();
    });
    socket.on('close', () => resolve(data));
    socket.on('error', reject);
  });
}

function close(server: Server): Promise<void> {
  return new Promise((resolve) => {
    server.closeAllConnections();
    server.close(() => resolve());
  });
}

/** Dials the durable upgrade; resolves with the frames received and the close code. */
function dial(
  url: string,
  headers: Headers,
  onOpen: (ws: WebSocket) => void
): Promise<{ frames: string[]; code: number } | { status: number; body: string }> {
  return new Promise((resolve, reject) => {
    const ws = new WebSocket(url, { headers: Object.fromEntries(headers.entries()) });
    const frames: string[] = [];

    ws.on('open', () => onOpen(ws));
    ws.on('message', (data) => frames.push(data.toString()));
    ws.on('close', (code) => resolve({ frames, code }));
    ws.on('unexpected-response', (_req, res) => {
      let body = '';
      res.on('data', (chunk) => (body += chunk));
      res.on('end', () => resolve({ status: res.statusCode ?? 0, body }));
    });
    ws.on('error', (err) => {
      if (!err.message.includes('Unexpected server response')) reject(err);
    });
  });
}

describe('node adapter', () => {
  const servers: Server[] = [];

  beforeEach(() => {
    process.env.HATCHET_SIGNING_SECRET = secret;
  });

  afterEach(async () => {
    delete process.env.HATCHET_SIGNING_SECRET;
    delete process.env.HATCHET_CLIENT_TOKEN;
    await Promise.all(servers.splice(0).map(close));
  });

  async function start(server: Server) {
    servers.push(server);
    return listen(server);
  }

  it('serves the healthcheck under /hatchet with the secret from process.env', async () => {
    const host = await start(createServer({ workflows }));
    const body = healthcheckBody();
    const response = await fetch(`http://${host}/hatchet/healthcheck`, {
      method: 'POST',
      body,
      headers: await signed(body),
    });

    expect(response.status).toBe(200);
    expect(await response.json()).toMatchObject({
      workflows: [{ name: 'echo' }, { name: 'durable-echo' }],
      durable: { supported: true },
      runtime: { name: 'node' },
    });
  });

  it('runs a non-durable trigger and answers 404 elsewhere', async () => {
    const host = await start(createServer({ workflows }));
    const body = triggerBody({ message: 'hello' });
    const response = await fetch(`http://${host}/hatchet/trigger`, {
      method: 'POST',
      body,
      headers: await signed(body),
    });

    expect(response.status).toBe(200);
    expect(await response.json()).toEqual({ echo: 'hello' });

    expect((await fetch(`http://${host}/other`)).status).toBe(404);
    expect((await fetch(`http://${host}/hatchet/healthcheck`)).status).toBe(405);
  });

  it("builds the tasks' client from HATCHET_CLIENT_TOKEN, and has none without it", async () => {
    process.env.HATCHET_CLIENT_TOKEN = TOKEN;
    const host = await start(createServer({ workflows: [whoami] }));
    const body = triggerBody({}, 'whoami');
    const response = await fetch(`http://${host}/hatchet/trigger`, {
      method: 'POST',
      body,
      headers: await signed(body),
    });

    expect(response.status).toBe(200);
    expect(await response.json()).toEqual({ tenant: TENANT_ID });

    delete process.env.HATCHET_CLIENT_TOKEN;
    const bare = await start(createServer({ workflows: [whoami] }));
    const refused = await fetch(`http://${bare}/hatchet/trigger`, {
      method: 'POST',
      body,
      headers: await signed(body),
    });

    expect(refused.status).toBe(422);
    expect(await refused.json()).toMatchObject({ retry: false });
  });

  it('answers 500 without a secret and 401 on a bad signature', async () => {
    delete process.env.HATCHET_SIGNING_SECRET;
    const host = await start(createServer({ workflows }));
    const body = healthcheckBody();

    expect(
      (
        await fetch(`http://${host}/hatchet/healthcheck`, {
          method: 'POST',
          body,
          headers: await signed(body),
        })
      ).status
    ).toBe(500);

    const withSecret = await start(createServer({ workflows, secret }));

    expect(
      (
        await fetch(`http://${withSecret}/hatchet/healthcheck`, {
          method: 'POST',
          body,
          headers: { [SIGNATURE_HEADER]: await signHex('other', body) },
        })
      ).status
    ).toBe(401);
  });

  it('calls next outside basePath when mounted as middleware, honouring the mount path', async () => {
    const listener = nodeHandler({ workflows, basePath: '/internal/hatchet' });
    const server = createHttpServer((req, res) => {
      // What Express does under app.use('/internal', listener): the mount path is stripped
      // from req.url and kept on originalUrl.
      const mounted = req as IncomingMessage & { originalUrl?: string };
      mounted.originalUrl = req.url;
      req.url = req.url!.replace(/^\/internal/, '') || '/';

      listener(req, res, () => {
        res.statusCode = 200;
        res.end('app');
      });
    });
    const host = await start(server);
    const body = healthcheckBody();

    expect(await (await fetch(`http://${host}/anything`)).text()).toBe('app');
    expect(
      (
        await fetch(`http://${host}/internal/hatchet/healthcheck`, {
          method: 'POST',
          body,
          headers: await signed(body),
        })
      ).status
    ).toBe(200);
  });

  it('runs a durable invocation over a real websocket', async () => {
    const host = await start(createServer({ workflows }));
    const operator = createTestOperator({ workflows, secret });
    const taskId = crypto.randomUUID();
    const headers = await operator.upgradeHeaders(taskId, 1);

    const result = await dial(`ws://${host}/hatchet/trigger`, headers, (ws) =>
      ws.send(firstFrame(taskId))
    );

    expect(result).toMatchObject({ code: 1000 });
    const { frames } = result as { frames: string[] };
    expect(frames).toHaveLength(1);
    expect(decodeFrame(frames[0]).done).toMatchObject({ output: JSON.stringify({ echo: 'hi' }) });
  });

  it('refuses an unsigned upgrade with an HTTP error on the raw socket', async () => {
    const host = await start(createServer({ workflows }));
    const operator = createTestOperator({ workflows, secret });
    const headers = await operator.upgradeHeaders(crypto.randomUUID(), 1, { secret: 'other' });

    const result = await dial(`ws://${host}/hatchet/trigger`, headers, () => {});

    expect(result).toMatchObject({ status: 401 });
    expect(JSON.parse((result as { body: string }).body)).toMatchObject({
      error: 'bad signature',
      retry: false,
    });
  });

  it('leaves upgrades outside basePath to the server and answers 404 from createServer', async () => {
    const listener = nodeHandler({ workflows });
    const server = createHttpServer(listener);
    const other = vi.fn();
    server.on('upgrade', (req, socket, head) => {
      if (!listener.upgrade(req, socket, head)) {
        other();
        socket.end('HTTP/1.1 418 Teapot\r\ncontent-length: 0\r\n\r\n');
      }
    });
    const host = await start(server);

    expect(await dial(`ws://${host}/elsewhere`, new Headers(), () => {})).toMatchObject({
      status: 418,
    });
    expect(other).toHaveBeenCalledTimes(1);

    const bare = await start(createServer({ workflows }));

    expect(await dial(`ws://${bare}/elsewhere`, new Headers(), () => {})).toMatchObject({
      status: 404,
    });
  });

  it('answers requests whose Host or target does not parse as a URL instead of throwing', async () => {
    const host = await start(createServer({ workflows, console: quiet }));

    // Node's parser passes these through; the URL parser rejects them.
    expect(
      await rawRequest(
        host,
        'POST /hatchet/trigger HTTP/1.1\r\nHost: a b\r\nContent-Length: 0\r\nConnection: close\r\n\r\n'
      )
    ).toMatch(/^HTTP\/1\.1 401 /);
    expect(
      await rawRequest(
        host,
        'GET /hatchet/trigger HTTP/1.1\r\nHost: a:99999\r\nUpgrade: websocket\r\nConnection: Upgrade\r\nSec-WebSocket-Key: dGhlIHNhbXBsZSBub25jZQ==\r\nSec-WebSocket-Version: 13\r\n\r\n'
      )
    ).toMatch(/^HTTP\/1\.1 401 /);
    expect(
      await rawRequest(
        host,
        'POST http://a:99999/hatchet/trigger HTTP/1.1\r\nHost: a:99999\r\nContent-Length: 0\r\nConnection: close\r\n\r\n'
      )
    ).toMatch(/^HTTP\/1\.1 400 /);
    expect(
      await rawRequest(
        host,
        'GET http://a:99999/hatchet/trigger HTTP/1.1\r\nHost: a:99999\r\nUpgrade: websocket\r\nConnection: Upgrade\r\nSec-WebSocket-Key: dGhlIHNhbXBsZSBub25jZQ==\r\nSec-WebSocket-Version: 13\r\n\r\n'
      )
    ).toMatch(/^HTTP\/1\.1 400 /);

    // The process and the server are still up.
    const body = healthcheckBody();
    expect(
      (
        await fetch(`http://${host}/hatchet/healthcheck`, {
          method: 'POST',
          body,
          headers: await signed(body),
        })
      ).status
    ).toBe(200);
  });

  it('answers 413 and drops the connection on a body over maxBodyBytes', async () => {
    const host = await start(createServer({ workflows, maxBodyBytes: 1024, console: quiet }));
    const response = await fetch(`http://${host}/hatchet/trigger`, {
      method: 'POST',
      body: 'x'.repeat(2048),
      headers: { [SIGNATURE_HEADER]: '0'.repeat(64) },
    });

    expect(response.status).toBe(413);
    expect(await response.json()).toEqual({
      error: 'the request body exceeds the 1024 byte limit',
      retry: false,
    });

    const body = healthcheckBody();
    expect(
      (
        await fetch(`http://${host}/hatchet/healthcheck`, {
          method: 'POST',
          body,
          headers: await signed(body),
        })
      ).status
    ).toBe(200);
  });

  it('aborts the task when the client drops the connection mid-task', async () => {
    blocked.signals = [];
    const host = await start(createServer({ workflows: [blocker], console: quiet }));
    const body = triggerBody({}, 'blocker');
    const req = request(`http://${host}/hatchet/trigger`, {
      method: 'POST',
      headers: { ...(await signed(body)), 'content-length': Buffer.byteLength(body) },
    });
    req.on('error', () => {});
    req.end(body);

    await vi.waitFor(() => expect(blocked.signals).toHaveLength(1));
    req.destroy();
    await vi.waitFor(() => expect(blocked.signals[0].aborted).toBe(true));

    blocked.release();
  });

  it('reports durable.supported false and refuses upgrades with durable off', async () => {
    const host = await start(createServer({ workflows, durable: false }));
    const body = healthcheckBody();
    const response = await fetch(`http://${host}/hatchet/healthcheck`, {
      method: 'POST',
      body,
      headers: await signed(body),
    });

    // protojson leaves a false `supported` out.
    expect(await response.json()).toMatchObject({ durable: {} });

    const operator = createTestOperator({ workflows, secret });
    const headers = await operator.upgradeHeaders(crypto.randomUUID(), 1);

    expect(await dial(`ws://${host}/hatchet/trigger`, headers, () => {})).toMatchObject({
      status: 426,
    });
  });
});
