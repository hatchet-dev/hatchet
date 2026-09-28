import { createServer as createHttpServer, type IncomingMessage, type Server } from 'node:http';
import type { AddressInfo } from 'node:net';
import { WebSocket } from 'ws';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import { ActionType, AssignedAction } from '../generated/proto/dispatcher';
import { ServerlessTriggerRequest } from '../generated/proto/v1/serverless';
import { SIGNATURE_HEADER, TRIGGER_ENVELOPE_VERSION, hatchet, signHex } from '../index';
import { decodeFrame, encodeFrame } from '../index';
import { createTestOperator } from '../testing';
import { createServer, nodeHandler } from './node';

const secret = 'test-secret-at-least-32-characters-long';

const echo = hatchet.task({
  name: 'echo',
  fn: async (input: { message: string }) => ({ echo: input.message }),
});

const durableEcho = hatchet.durableTask({
  name: 'durable-echo',
  fn: async (input: { message: string }) => ({ echo: input.message }),
});

const workflows = [echo, durableEcho];

async function signed(body: string) {
  return { [SIGNATURE_HEADER]: await signHex(secret, body), 'content-type': 'application/json' };
}

function healthcheckBody() {
  return JSON.stringify({
    endpointId: 'e',
    timestampUnixSeconds: String(Math.floor(Date.now() / 1000)),
  });
}

function triggerBody(input: unknown) {
  const action = AssignedAction.fromPartial({
    tenantId: crypto.randomUUID(),
    workflowRunId: crypto.randomUUID(),
    jobId: crypto.randomUUID(),
    jobName: 'echo',
    jobRunId: crypto.randomUUID(),
    taskId: crypto.randomUUID(),
    taskRunExternalId: crypto.randomUUID(),
    actionId: 'echo:echo',
    actionType: ActionType.START_STEP_RUN,
    actionPayload: JSON.stringify({ input, parents: {}, triggered_by: 'manual' }),
    taskName: 'echo',
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
