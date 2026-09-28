import type { AddressInfo } from 'node:net';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import { ActionType, AssignedAction } from '../generated/proto/dispatcher';
import { SIGNATURE_HEADER, hatchet, signHex } from '../index';
import { decodeFrame, encodeFrame } from '../index';
import { createTestOperator } from '../testing';
import { vercel, vercelServer } from './vercel';

// A stand-in for @vercel/functions: the upgrade hands the callback a fake `ws` socket the
// test drives, and, like the real one, resolves with a placeholder response once the
// callback settles or throws when the runtime has no upgrade.
const fake = await vi.hoisted(async () => {
  const { EventEmitter } = await import('node:events');
  const state = {
    unavailable: undefined as Error | undefined,
    sockets: [] as FakeSocket[],
    waited: [] as Promise<unknown>[],
  };

  class FakeSocket extends EventEmitter {
    sent: string[] = [];
    closed: { code?: number; reason?: string } | undefined;
    send(text: string) {
      this.sent.push(text);
    }
    close(code?: number, reason?: string) {
      this.closed = { code, reason };
      queueMicrotask(() => this.emit('close', code ?? 1005, Buffer.from(reason ?? '')));
    }
  }

  return { state, FakeSocket };
});

vi.mock('@vercel/functions', () => ({
  experimental_upgradeWebSocket: async (
    handler: (ws: unknown) => void | Promise<void>,
    _options?: { maxPayload?: number }
  ) => {
    if (fake.state.unavailable) {
      throw fake.state.unavailable;
    }

    const socket = new fake.FakeSocket();
    fake.state.sockets.push(socket);
    await handler(socket);

    return new Response(null, { status: 204 });
  },
  waitUntil: (promise: Promise<unknown>) => {
    fake.state.waited.push(promise);
  },
}));

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

async function signedHealthcheck(url: string, signWith = secret) {
  const body = JSON.stringify({
    endpointId: 'e',
    timestampUnixSeconds: String(Math.floor(Date.now() / 1000)),
  });

  return new Request(url, {
    method: 'POST',
    body,
    headers: { [SIGNATURE_HEADER]: await signHex(signWith, body) },
  });
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

const plain = { params: Promise.resolve({}) };

const quiet = { debug: vi.fn(), info: vi.fn(), warn: vi.fn(), error: vi.fn() };

describe('vercel adapter', () => {
  beforeEach(() => {
    process.env.HATCHET_SIGNING_SECRET = secret;
    fake.state.unavailable = undefined;
    fake.state.sockets = [];
    fake.state.waited = [];
    quiet.warn.mockClear();
  });

  afterEach(() => {
    delete process.env.HATCHET_SIGNING_SECRET;
    delete process.env.VERCEL;
  });

  it('serves the healthcheck under /api/hatchet with the secret from process.env', async () => {
    const { POST } = vercel({ workflows, durable: true });
    const response = await POST(
      await signedHealthcheck('https://app.test/api/hatchet/healthcheck'),
      plain
    );

    expect(response.status).toBe(200);
    expect(await response.json()).toMatchObject({
      workflows: [{ name: 'echo' }, { name: 'durable-echo' }],
      durable: { supported: true },
      runtime: { name: 'vercel' },
    });
  });

  it('routes by the catch-all segment wherever the route file lives', async () => {
    const { POST } = vercel({ workflows, durable: true });
    const context = { params: Promise.resolve({ slug: ['healthcheck'] }) };

    expect(
      (await POST(await signedHealthcheck('https://app.test/internal/x/healthcheck'), context))
        .status
    ).toBe(200);
    expect(
      (await POST(await signedHealthcheck('https://app.test/internal/x/healthcheck'), plain)).status
    ).toBe(404);
    expect(
      (
        await POST(await signedHealthcheck('https://app.test/internal/x/nope'), {
          params: Promise.resolve({ slug: ['nope'] }),
        })
      ).status
    ).toBe(404);
  });

  it('honours basePath and the secret option', async () => {
    delete process.env.HATCHET_SIGNING_SECRET;
    const { POST } = vercel({
      workflows,
      basePath: '/hooks/hatchet',
      secret: () => secret,
      durable: true,
    });

    expect(
      (await POST(await signedHealthcheck('https://app.test/hooks/hatchet/healthcheck'), plain))
        .status
    ).toBe(200);
    expect(
      (
        await POST(
          await signedHealthcheck('https://app.test/hooks/hatchet/healthcheck', 'other'),
          plain
        )
      ).status
    ).toBe(401);
  });

  it('advertises durable only on Vercel or when asked', async () => {
    const off = vercel({ workflows });
    // protojson leaves a false `supported` out.
    expect(
      await (
        await off.POST(await signedHealthcheck('https://app.test/api/hatchet/healthcheck'), plain)
      ).json()
    ).toMatchObject({ durable: {} });

    process.env.VERCEL = '1';
    const on = vercel({ workflows });
    expect(
      await (
        await on.POST(await signedHealthcheck('https://app.test/api/hatchet/healthcheck'), plain)
      ).json()
    ).toMatchObject({ durable: { supported: true } });
  });

  it('runs a durable invocation through experimental_upgradeWebSocket and waitUntil', async () => {
    const { GET } = vercel({ workflows, durable: true });
    const operator = createTestOperator({ workflows, secret });
    const taskId = crypto.randomUUID();
    const headers = await operator.upgradeHeaders(taskId, 1);

    const pending = GET(new Request('https://app.test/api/hatchet/trigger', { headers }), plain);

    await vi.waitFor(() => expect(fake.state.sockets).toHaveLength(1));
    const [socket] = fake.state.sockets;
    socket.emit('message', Buffer.from(firstFrame(taskId)), false);

    const response = await pending;

    expect(response.status).toBe(204);
    expect(socket.sent).toHaveLength(1);
    expect(decodeFrame(socket.sent[0]).done).toMatchObject({
      output: JSON.stringify({ echo: 'hi' }),
    });
    expect(socket.closed).toMatchObject({ code: 1000 });
    expect(fake.state.waited).toHaveLength(1);
  });

  it('verifies the upgrade before touching the runtime', async () => {
    const { GET } = vercel({ workflows, durable: true });
    const operator = createTestOperator({ workflows, secret });
    const headers = await operator.upgradeHeaders(crypto.randomUUID(), 1, { secret: 'other' });

    const response = await GET(
      new Request('https://app.test/api/hatchet/trigger', { headers }),
      plain
    );

    expect(response.status).toBe(401);
    expect(fake.state.sockets).toHaveLength(0);
  });

  it('answers 426 with guidance when the runtime has no websocket upgrade', async () => {
    fake.state.unavailable = new Error(
      'experimental_upgradeWebSocket is not available in the current runtime environment.'
    );
    const { GET } = vercel({ workflows, durable: true, console: quiet });
    const operator = createTestOperator({ workflows, secret });
    const headers = await operator.upgradeHeaders(crypto.randomUUID(), 1);

    const response = await GET(
      new Request('https://app.test/api/hatchet/trigger', { headers }),
      plain
    );

    expect(response.status).toBe(426);
    expect(await response.json()).toMatchObject({
      error: expect.stringMatching(/not available in the current runtime/),
      retry: false,
    });
    expect(quiet.warn).toHaveBeenCalledWith(expect.stringMatching(/Fluid compute.*vc dev/));
  });

  it('refuses upgrades and reports durable.supported false with durable off', async () => {
    const { GET, POST } = vercel({ workflows, durable: false });
    const operator = createTestOperator({ workflows, secret });
    const headers = await operator.upgradeHeaders(crypto.randomUUID(), 1);

    expect(
      await (
        await POST(await signedHealthcheck('https://app.test/api/hatchet/healthcheck'), plain)
      ).json()
    ).toMatchObject({ durable: {} });
    expect(
      (await GET(new Request('https://app.test/api/hatchet/trigger', { headers }), plain)).status
    ).toBe(426);
    expect(fake.state.sockets).toHaveLength(0);
  });

  it('vercelServer is the Node server reporting the vercel runtime', async () => {
    const server = vercelServer({ workflows });
    const host = await new Promise<string>((resolve) =>
      server.listen(0, '127.0.0.1', () =>
        resolve(`127.0.0.1:${(server.address() as AddressInfo).port}`)
      )
    );

    try {
      const request = await signedHealthcheck(`http://${host}/hatchet/healthcheck`);
      const response = await fetch(request);

      expect(response.status).toBe(200);
      expect(await response.json()).toMatchObject({
        runtime: { name: 'vercel' },
        durable: { supported: true },
      });
    } finally {
      server.closeAllConnections();
      await new Promise((resolve) => server.close(resolve));
    }
  });
});
