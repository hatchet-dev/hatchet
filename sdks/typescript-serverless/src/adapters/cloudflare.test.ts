/// <reference types="@cloudflare/workers-types" />
import { createRouterTransport, type Transport } from '@connectrpc/connect';
import { EventsService } from '@hatchet-dev/typescript-sdk/protoc-es/events/events_pb.js';
import { describe, expect, it, vi } from 'vitest';
import { ActionType, AssignedAction } from '../generated/proto/dispatcher';
import { ServerlessTriggerRequest } from '../generated/proto/v1/serverless';
import { HatchetCore, hatchet } from '../index';
import { SIGNATURE_HEADER, TRIGGER_ENVELOPE_VERSION, signHex } from '../index';
import { cloudflare, type HatchetEnv } from './cloudflare';

const secret = 'test-secret-at-least-32-characters-long';
const TOKEN = `${Buffer.from('{"alg":"HS256"}').toString('base64url')}.${Buffer.from(
  JSON.stringify({ sub: '707d0855-80ab-4e1f-a156-f1c4546cbf52' })
).toString('base64url')}.sig`;

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
    timestampUnixSeconds: String(Math.floor(Date.now() / 1000)),
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

  describe('client', () => {
    const streamer = hatchet.task({
      name: 'streamer',
      fn: async (_input: {}, ctx) => {
        await ctx.putStream('hello');
        return { streamed: true };
      },
    });

    function fakeEngine() {
      const streamed: string[] = [];
      const transport = createRouterTransport(({ service }) => {
        service(EventsService, {
          putStreamEvent: (req) => {
            streamed.push(new TextDecoder().decode(req.message));
            return {};
          },
        });
      });

      return { transport, streamed };
    }

    async function signedTrigger() {
      const body = JSON.stringify(
        ServerlessTriggerRequest.toJSON({
          endpointId: 'e',
          timestampUnixSeconds: Math.floor(Date.now() / 1000),
          version: TRIGGER_ENVELOPE_VERSION,
          action: AssignedAction.fromPartial({
            tenantId: crypto.randomUUID(),
            workflowRunId: crypto.randomUUID(),
            jobId: crypto.randomUUID(),
            jobName: 'streamer',
            jobRunId: crypto.randomUUID(),
            taskId: crypto.randomUUID(),
            taskRunExternalId: crypto.randomUUID(),
            actionId: 'streamer:streamer',
            actionType: ActionType.START_STEP_RUN,
            actionPayload: JSON.stringify({ input: {}, parents: {}, triggered_by: 'manual' }),
            taskName: 'streamer',
          }),
        })
      );

      return new Request('https://worker.test/hatchet/trigger', {
        method: 'POST',
        body,
        headers: { [SIGNATURE_HEADER]: await signHex(secret, body) },
      });
    }

    it('builds the client from env on the request and hands it to the task', async () => {
      type Env = HatchetEnv & { ENGINE: Transport };
      const engine = fakeEngine();
      const worker = cloudflare<Env>({
        workflows: [streamer],
        client: (env) => ({ token: TOKEN, transport: env.ENGINE, logLevel: 'OFF' }),
      });
      const response = await worker.fetch!(
        (await signedTrigger()) as never,
        { HATCHET_SIGNING_SECRET: secret, ENGINE: engine.transport },
        executionContext()
      );

      expect(response.status).toBe(200);
      expect(await response.json()).toEqual({ streamed: true });
      expect(engine.streamed).toEqual(['hello']);
    });

    it('takes a ready client as is', async () => {
      const engine = fakeEngine();
      const worker = cloudflare({
        workflows: [streamer],
        client: new HatchetCore({ token: TOKEN, transport: engine.transport, logLevel: 'OFF' }),
      });

      expect((await call(worker, await signedTrigger())).status).toBe(200);
      expect(engine.streamed).toEqual(['hello']);
    });

    it('has no client without HATCHET_CLIENT_TOKEN, so the task fails without retry', async () => {
      const worker = cloudflare({ workflows: [streamer] });
      const response = await call(worker, await signedTrigger());

      expect(response.status).toBe(422);
      expect(await response.json()).toMatchObject({
        error: expect.stringMatching(/ctx\.putStream .* configure `client`/),
        retry: false,
      });
    });
  });
});
