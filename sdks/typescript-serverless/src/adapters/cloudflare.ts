/**
 * `@hatchet-dev/serverless/cloudflare`: the Cloudflare Workers adapter.
 *
 * ```ts
 * import { cloudflare } from '@hatchet-dev/serverless/cloudflare';
 * import { workflows } from './tasks';
 *
 * export default cloudflare({ workflows });
 * ```
 *
 * Serves `POST <basePath>/healthcheck`, `POST <basePath>/trigger` and the durable websocket
 * upgrade on `<basePath>/trigger`, reads the signing secret from `env.HATCHET_SIGNING_SECRET`
 * and hands every other path to the `fetch` option or answers 404. A durable invocation
 * runs under `ctx.waitUntil` with its socket open, so it outlives the 101 response. Typed
 * against `@cloudflare/workers-types`; nothing from Cloudflare is imported at runtime.
 * @module Cloudflare
 */
/// <reference types="@cloudflare/workers-types" />
import { createHandler, type DurableSocket, type HandlerOptions } from '../handler';

/** The bindings the adapter reads. Extend it with your own for the `fetch` fallback. */
export interface HatchetEnv {
  /** `wrangler secret put HATCHET_SIGNING_SECRET`; the endpoint's signingSecret in Hatchet. */
  HATCHET_SIGNING_SECRET?: string;
  /** Optional: when set, durable upgrades from any other endpoint id are refused. */
  HATCHET_ENDPOINT_ID?: string;
}

export interface CloudflareOptions<Env = HatchetEnv> extends Omit<
  HandlerOptions,
  'secret' | 'endpointId' | 'runtime' | 'durable'
> {
  /** The signing secret; defaults to `env.HATCHET_SIGNING_SECRET`. */
  secret?: string | ((env: Env) => string | undefined);
  /** The endpoint id; defaults to `env.HATCHET_ENDPOINT_ID`. */
  endpointId?: string | ((env: Env) => string | undefined);
  /** Handles every request outside `basePath`. Without it those requests get a 404. */
  fetch?: (request: Request, env: Env, ctx: ExecutionContext) => Response | Promise<Response>;
}

function fromEnv<Env, T>(
  value: T | ((env: Env) => T | undefined) | undefined,
  env: Env,
  fallback: (env: HatchetEnv) => T | undefined
): T | undefined {
  if (typeof value === 'function') {
    return (value as (env: Env) => T | undefined)(env);
  }

  return value ?? fallback((env ?? {}) as HatchetEnv);
}

/** Adapts Cloudflare's server-side WebSocket to what the relay expects. */
export function cloudflareSocket(ws: WebSocket): DurableSocket {
  return {
    send: (text) => ws.send(text),
    close: (code, reason) => ws.close(code, reason),
    onMessage: (listener) =>
      ws.addEventListener('message', (event: MessageEvent) => {
        if (typeof event.data === 'string') {
          listener(event.data);
        } else {
          ws.close(1003, 'text frames only');
        }
      }),
    onClose: (listener) => {
      let closed = false;
      const once = (code: number, reason: string) => {
        if (closed) return;
        closed = true;
        listener(code, reason);
      };
      ws.addEventListener('close', (event: CloseEvent) => once(event.code, event.reason));
      ws.addEventListener('error', () => once(1006, 'socket error'));
    },
  };
}

export function cloudflare<Env = HatchetEnv>(
  options: CloudflareOptions<Env>
): ExportedHandler<Env> {
  const { secret, endpointId, fetch: fallback, ...rest } = options;

  const handler = createHandler({
    ...rest,
    runtime: { name: 'cloudflare-workers' },
    durable: true,
    secret: (env) => fromEnv(secret, env as Env, (bindings) => bindings.HATCHET_SIGNING_SECRET),
    endpointId: (env) =>
      fromEnv(endpointId, env as Env, (bindings) => bindings.HATCHET_ENDPOINT_ID),
  });

  return {
    async fetch(request, env, ctx) {
      if (handler.matches(request)) {
        return handler.fetch(request, env, {
          waitUntil: (promise) => ctx.waitUntil(promise),
          upgrade: (_request, run) => {
            const pair = new WebSocketPair();
            const [client, server] = Object.values(pair) as [WebSocket, WebSocket];

            server.accept();
            ctx.waitUntil(run(cloudflareSocket(server)));

            return new Response(null, { status: 101, webSocket: client });
          },
        });
      }

      if (fallback) {
        return fallback(request, env, ctx);
      }

      return new Response('not found', { status: 404 });
    },
  };
}
