/**
 * The generic handler: WinterCG `Request` in, `Response` out. Adapters route requests under
 * `basePath` to it and resolve the signing secret from their runtime's environment.
 */
import type { BaseWorkflowDeclaration } from '@hatchet-dev/typescript-sdk/edge/index.js';
import type { ServerlessHealthcheckResponse } from '../generated/proto/v1/serverless';
import { createClientResolver, type ClientOption } from './client';
import { OPERATOR_MAX_FRAME_BYTES, SIGNATURE_HEADER } from './contract';
import type { ConsoleLike } from './context';
import { runDurableInvocation } from './durable/invocation';
import type { DurableHooks } from './durable/socket';
import { buildHealthcheck, serializeHealthcheck } from './healthcheck';
import { json, jsonText, readBodyText, triggerError } from './http';
import { NonceSet, type NonceOutcome } from './nonce-set';
import { buildRegistry, type ServeEntry } from './registry';
import { isSignatureShaped, verifySignedBody, verifyUpgradeSignature } from './signature';
import { handleTrigger } from './trigger';

export type { DurableHooks, DurableSocket } from './durable/socket';

/** Resolves a value from the runtime's environment object, whatever shape it has. */
export type EnvResolver<T> = T | ((env: unknown) => T | undefined);

export type ServerlessRuntimeName = 'cloudflare-workers' | 'vercel' | (string & {});

export interface HandlerOptions {
  /** Every workflow this endpoint advertises. */
  // eslint-disable-next-line @typescript-eslint/no-explicit-any
  workflows: BaseWorkflowDeclaration<any, any>[];
  /**
   * The workflows (or action ids, `workflow:task`) this endpoint serves. Defaults to every
   * action of every workflow. Triggers for actions outside the subset are answered 404.
   */
  serve?: ServeEntry[];
  /**
   * The Hatchet client a task's `ctx` methods delegate to: a `HatchetCore` instance, the
   * config `HatchetCore` takes (`{ token, serverUrl?, tls?, namespace?, ... }`), or a function
   * of the runtime's environment returning either, for runtimes that hand secrets to the
   * request. The token comes from the platform's secret store; the operator never sends it.
   * Without a client, `ctx.runChild`, `ctx.putStream`, `ctx.cancel`, `clientFromContext(ctx)`
   * and their relatives throw `ServerlessLimitationError`, and `ctx.log` stays on the console.
   * A task declared with `hatchetClient: false` behaves that way with a client too.
   */
  client?: ClientOption;
  /** The path the healthcheck and trigger routes live under. Defaults to `/hatchet`. */
  basePath?: string;
  /** The endpoint's signing secret, or a function reading it from the runtime's env. */
  secret: EnvResolver<string>;
  /** When set, requests and upgrades carrying another endpoint id are refused with 403. */
  endpointId?: EnvResolver<string>;
  runtime: { name: ServerlessRuntimeName };
  /**
   * Whether the adapter supplies `DurableHooks.upgrade` on every fetch. Only then does the
   * healthcheck advertise durable support (for handlers with a durable task) and list the
   * non-durable tasks the operator invokes over the socket; without it every non-durable
   * task is POSTed, and only the waits that need the socket fail.
   */
  durable?: boolean;
  /**
   * Replay protection for durable upgrades: consumes the nonce and returns true when it was
   * seen before, or `'full'` when there is no room to record it (the upgrade is then refused
   * with 503 and the operator retries later). `expiresAtUnixSeconds` is the last second at
   * which the upgrade's signature is still fresh; keep the nonce through it. The hook may be
   * asynchronous, and a store failure (a thrown error or a rejection) refuses the upgrade
   * with 503 so the operator retries. Defaults to a bounded in-memory set (4096 entries, each
   * kept until its signature expires) that lives in one isolate; back it with a Durable
   * Object or an expiring KV key in production so a replay landing in another isolate is
   * caught too. Check-and-insert must be atomic in the store: a KV get followed by a put is
   * not a replay guarantee.
   */
  seenNonce?: (
    nonce: string,
    expiresAtUnixSeconds: number
  ) => boolean | 'full' | Promise<boolean | 'full'>;
  /**
   * The largest POST body accepted, in bytes, enforced while the body is read and before
   * its signature is checked; a larger one is answered 413. Defaults to the operator's
   * frame limit, `OPERATOR_MAX_FRAME_BYTES` (4 MiB).
   */
  maxBodyBytes?: number;
  /**
   * The largest websocket frame the invocation accepts from the operator, in bytes, checked
   * before the frame is decoded; a larger one fails the invocation. Defaults to
   * `OPERATOR_MAX_FRAME_BYTES`. The Node and Vercel adapters also pass their `maxPayload` to
   * `ws`, which drops larger frames at the socket.
   */
  maxFrameBytes?: number;
  /**
   * How many completions the invocation keeps for durable entries whose waiter has not been
   * installed yet; more than that fails the invocation. Defaults to 4096.
   */
  maxPendingCompletions?: number;
  /** Where warnings and task logs go; defaults to the global console. */
  console?: ConsoleLike;
}

export interface ServerlessHandler {
  /** The normalized base path, with a leading slash and no trailing one. */
  readonly basePath: string;
  /** Whether the request targets a route under `basePath`. */
  matches(request: Request | URL | string): boolean;
  fetch(request: Request, env?: unknown, hooks?: DurableHooks): Promise<Response>;
  /** The healthcheck body, built once per handler. */
  healthcheck(): ServerlessHealthcheckResponse;
}

type Route = 'healthcheck' | 'trigger';

export function normalizeBasePath(basePath: string | undefined): string {
  const trimmed = (basePath ?? '/hatchet').replace(/^\/+|\/+$/g, '');

  return trimmed ? `/${trimmed}` : '';
}

function resolve<T>(resolver: EnvResolver<T> | undefined, env: unknown): T | undefined {
  if (typeof resolver === 'function') {
    return (resolver as (env: unknown) => T | undefined)(env);
  }

  return resolver;
}

export function createHandler(options: HandlerOptions): ServerlessHandler {
  const out = options.console ?? console;
  const registry = buildRegistry(options.workflows, options.serve, (message) => out.warn(message));
  const clientFor = createClientResolver(options.client);
  const basePath = normalizeBasePath(options.basePath);
  const socket = options.durable === true;
  const healthcheck = buildHealthcheck(registry, options.runtime, {
    socket,
    durable: socket && registry.durableActions.size > 0,
  });
  const healthcheckBody = serializeHealthcheck(healthcheck);
  const nonces = new NonceSet();
  const maxBodyBytes = options.maxBodyBytes ?? OPERATOR_MAX_FRAME_BYTES;
  const consumeNonce = async (nonce: string, expiresAt: number): Promise<NonceOutcome> => {
    if (!options.seenNonce) {
      return nonces.consume(nonce, expiresAt);
    }

    const seen = await options.seenNonce(nonce, expiresAt);

    return seen === 'full' ? 'full' : seen ? 'replayed' : 'accepted';
  };

  const routeOf = (pathname: string): Route | undefined => {
    if (!pathname.startsWith(basePath)) {
      return undefined;
    }

    switch (pathname.slice(basePath.length)) {
      case '/healthcheck':
        return 'healthcheck';
      case '/trigger':
        return 'trigger';
      default:
        return undefined;
    }
  };

  const pathnameOf = (target: Request | URL | string): string => {
    if (typeof target === 'string') {
      return new URL(target).pathname;
    }

    return target instanceof URL ? target.pathname : new URL(target.url).pathname;
  };

  return {
    basePath,

    matches: (target) => routeOf(pathnameOf(target)) !== undefined,

    healthcheck: () => healthcheck,

    async fetch(request, env, hooks) {
      const route = routeOf(pathnameOf(request));

      if (!route) {
        return json({ error: 'not found' }, 404);
      }

      const secret = resolve(options.secret, env);

      if (!secret) {
        return json(
          { error: 'the Hatchet signing secret is not configured on this endpoint' },
          500
        );
      }

      if (route === 'healthcheck') {
        if (request.method !== 'POST') {
          return json({ error: 'method not allowed' }, 405);
        }

        // A request without a signature costs no body bytes.
        const signature = request.headers.get(SIGNATURE_HEADER);

        if (!isSignatureShaped(signature)) {
          return json({ error: 'bad signature' }, 401);
        }

        const read = await readBodyText(request, maxBodyBytes);

        if (!read.ok) {
          return read.response;
        }

        const verified = await verifySignedBody(read.text, signature, secret, {
          endpointId: resolve(options.endpointId, env),
        });

        if (!verified.ok) {
          return json({ error: verified.reason }, verified.status);
        }

        return jsonText(healthcheckBody);
      }

      if (request.headers.get('upgrade')?.toLowerCase() === 'websocket') {
        if (!hooks?.upgrade) {
          return triggerError(
            426,
            'this endpoint does not accept the durable websocket relay; its adapter provides no upgrade hook',
            false
          );
        }

        const verified = await verifyUpgradeSignature(request.headers, secret, {
          endpointId: resolve(options.endpointId, env),
          consumeNonce,
        });

        if (!verified.ok) {
          // 503 is retryable for the operator: the nonce store is full of unexpired entries.
          return triggerError(verified.status, verified.reason, verified.status === 503);
        }

        const client = clientFor(env);

        return hooks.upgrade(request, (socket) =>
          runDurableInvocation({
            socket,
            registry,
            client,
            console: options.console,
            expected: { taskRunExternalId: verified.taskId, invocationCount: verified.invocation },
            maxFrameBytes: options.maxFrameBytes,
            maxPendingCompletions: options.maxPendingCompletions,
          })
        );
      }

      return handleTrigger(request, {
        registry,
        secret,
        endpointId: resolve(options.endpointId, env),
        client: clientFor(env),
        console: options.console,
        maxBodyBytes,
      });
    },
  };
}
