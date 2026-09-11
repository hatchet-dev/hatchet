/// <reference types="@cloudflare/workers-types" />
import { describe, expect, it, vi } from 'vitest';
import { hatchet } from '../index';
import { SIGNATURE_HEADER, signHex } from '../index';
import { cloudflare, type HatchetEnv } from './cloudflare';

const secret = 'test-secret-at-least-32-characters-long';

const echo = hatchet.task({
  name: 'echo',
  fn: async (input: { message: string }) => ({ echo: input.message }),
});

function executionContext(): ExecutionContext {
  return {
    waitUntil: vi.fn(),
    passThroughOnException: vi.fn(),
  } as unknown as ExecutionContext;
}

async function signedHealthcheck(path: string, signWith = secret) {
  const body = JSON.stringify({
    endpointId: 'e',
    namespace: 'n',
    timestamp: String(Math.floor(Date.now() / 1000)),
  });

  return new Request(`https://worker.test${path}`, {
    method: 'POST',
    body,
    headers: { [SIGNATURE_HEADER]: await signHex(signWith, body) },
  });
}

async function call(
  worker: ExportedHandler<HatchetEnv>,
  request: Request,
  env: HatchetEnv = { HATCHET_SIGNING_SECRET: secret }
) {
  return worker.fetch!(request as never, env, executionContext());
}

describe('cloudflare adapter', () => {
  it('serves the healthcheck under /hatchet with the secret from env', async () => {
    const worker = cloudflare({ workflows: [echo] });
    const response = await call(worker, await signedHealthcheck('/hatchet/healthcheck'));

    expect(response.status).toBe(200);
    expect(await response.json()).toMatchObject({
      workflows: [{ name: 'echo' }],
      runtime: { name: 'cloudflare-workers' },
    });
  });

  it('honours basePath', async () => {
    const worker = cloudflare({ workflows: [echo], basePath: '/internal/hatchet/' });

    expect(
      (await call(worker, await signedHealthcheck('/internal/hatchet/healthcheck'))).status
    ).toBe(200);
    expect((await call(worker, await signedHealthcheck('/hatchet/healthcheck'))).status).toBe(404);
  });

  it('answers 500 when no secret is configured and 401 on a bad signature', async () => {
    const worker = cloudflare({ workflows: [echo] });

    expect((await call(worker, await signedHealthcheck('/hatchet/healthcheck'), {})).status).toBe(
      500
    );
    expect(
      (await call(worker, await signedHealthcheck('/hatchet/healthcheck', 'other'))).status
    ).toBe(401);
  });

  it('resolves the secret through the secret option', async () => {
    type Env = { ORDERS_SIGNING_SECRET: string };
    const worker = cloudflare<Env>({
      workflows: [echo],
      secret: (env) => env.ORDERS_SIGNING_SECRET,
    });
    const response = await worker.fetch!(
      (await signedHealthcheck('/hatchet/healthcheck')) as never,
      { ORDERS_SIGNING_SECRET: secret },
      executionContext()
    );

    expect(response.status).toBe(200);
  });

  it('falls through to the user fetch outside basePath, else 404', async () => {
    const fallback = vi.fn(async () => new Response('app', { status: 200 }));
    const worker = cloudflare({ workflows: [echo], fetch: fallback });
    const request = new Request('https://worker.test/anything');

    expect(await (await call(worker, request)).text()).toBe('app');
    expect(fallback).toHaveBeenCalledTimes(1);

    const bare = cloudflare({ workflows: [echo] });

    expect((await call(bare, request)).status).toBe(404);
    expect((await call(bare, new Request('https://worker.test/hatchet/other'))).status).toBe(404);
  });

  it('verifies the upgrade signature before accepting a durable socket', async () => {
    const worker = cloudflare({ workflows: [echo] });
    const request = new Request('https://worker.test/hatchet/trigger', {
      headers: { upgrade: 'websocket', connection: 'upgrade' },
    });
    const response = await call(worker, request);

    expect(response.status).toBe(401);
    expect(await response.json()).toMatchObject({ error: expect.stringMatching(/endpoint id/) });
  });
});
