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
 * Serves `POST <basePath>/healthcheck` and `POST <basePath>/trigger`, reads the signing
 * secret from `env.HATCHET_SIGNING_SECRET` and hands every other path to the `fetch`
 * option or answers 404. Typed against `@cloudflare/workers-types`; nothing from Cloudflare
 * is imported at runtime.
 * @module Cloudflare
 */
/// <reference types="@cloudflare/workers-types" />
import { createHandler, type HandlerOptions } from '../handler';

/** The bindings the adapter reads. Extend it with your own for the `fetch` fallback. */
export interface HatchetEnv {
  /** `wrangler secret put HATCHET_SIGNING_SECRET`; the endpoint's signingSecret in Hatchet. */
  HATCHET_SIGNING_SECRET?: string;
  /** Optional: when set, durable upgrades from any other endpoint id are refused. */
  HATCHET_ENDPOINT_ID?: string;
}

export interface CloudflareOptions<Env = HatchetEnv> extends Omit<
  HandlerOptions,
  'secret' | 'endpointId' | 'runtime'
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

export function cloudflare<Env = HatchetEnv>(
  options: CloudflareOptions<Env>
): ExportedHandler<Env> {
  const { secret, endpointId, fetch: fallback, ...rest } = options;

  const handler = createHandler({
    ...rest,
    runtime: { name: 'cloudflare-workers' },
    secret: (env) => fromEnv(secret, env as Env, (bindings) => bindings.HATCHET_SIGNING_SECRET),
    endpointId: (env) =>
      fromEnv(endpointId, env as Env, (bindings) => bindings.HATCHET_ENDPOINT_ID),
  });

  return {
    async fetch(request, env, ctx) {
      if (handler.matches(request)) {
        return handler.fetch(request, env, { waitUntil: (promise) => ctx.waitUntil(promise) });
      }

      if (fallback) {
        return fallback(request, env, ctx);
      }

      return new Response('not found', { status: 404 });
    },
  };
}
