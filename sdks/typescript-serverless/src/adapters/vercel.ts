/**
 * `@hatchet-dev/serverless/vercel`: the Vercel adapter.
 *
 * ```ts
 * // app/api/hatchet/[...hatchet]/route.ts
 * import { vercel } from '@hatchet-dev/serverless/vercel';
 * import { workflows } from '@/hatchet/tasks';
 *
 * export const { GET, POST } = vercel({ workflows });
 * export const maxDuration = 300;
 * ```
 *
 * `POST` serves the healthcheck and non-durable triggers; `GET` is the durable websocket
 * upgrade, accepted through `experimental_upgradeWebSocket` from `@vercel/functions`, which
 * needs Fluid compute and the `ws` package (both optional peer dependencies). The route is
 * read from the catch-all segment, so the route file may live anywhere; without a route
 * context the URL is matched against `basePath` (`/api/hatchet` by default). The signing
 * secret comes from `process.env.HATCHET_SIGNING_SECRET`.
 *
 * `vercelServer` is the same endpoint as an `http.Server`, the shape a standalone
 * `api/*.ts` function exports by default.
 * @module Vercel
 */
import type { Server } from 'node:http';
import type { HandlerOptions } from '../handler';
import type { ConsoleLike } from '../handler/context';
import { triggerError } from '../handler/http';
import { OPERATOR_MAX_FRAME_BYTES, createServer, nodeSocket, type NodeOptions } from './node';
import { createOptionalDurableHandler } from './optional-durable';

export interface VercelOptions extends Omit<
  HandlerOptions,
  'secret' | 'endpointId' | 'runtime' | 'durable' | 'basePath'
> {
  /**
   * The path the routes live under when the handlers are called without a Next.js route
   * context. Defaults to `/api/hatchet`. With a catch-all route the segment decides.
   */
  basePath?: string;
  /** The signing secret; defaults to `process.env.HATCHET_SIGNING_SECRET`. */
  secret?: string | (() => string | undefined);
  /** The endpoint id; defaults to `process.env.HATCHET_ENDPOINT_ID`. */
  endpointId?: string | (() => string | undefined);
  /**
   * Whether to serve durable tasks over the websocket relay. Defaults to true on Vercel
   * (`VERCEL` is set, also by `vc dev`) when `@vercel/functions` and `ws` are installed; set
   * it to false on a project without Fluid compute, where the upgrade is unavailable, so the
   * healthcheck reports `durable.supported: false` and the operator never dials.
   */
  durable?: boolean;
  /** The largest websocket frame accepted; defaults to the operator's 4 MiB limit. */
  maxPayload?: number;
}

/** The second argument Next.js passes to a route handler; `params` is a promise on 15+. */
export interface VercelRouteContext {
  params?:
    | Promise<Record<string, string | string[] | undefined>>
    | Record<string, string | string[] | undefined>;
}

export type VercelRouteHandler = (
  request: Request,
  context?: VercelRouteContext
) => Promise<Response>;

export interface VercelRoutes {
  /** The durable websocket upgrade on `<basePath>/trigger`. */
  GET: VercelRouteHandler;
  /** The healthcheck and non-durable triggers. */
  POST: VercelRouteHandler;
}

interface VercelFunctions {
  experimental_upgradeWebSocket: typeof import('@vercel/functions').experimental_upgradeWebSocket;
  waitUntil: typeof import('@vercel/functions').waitUntil;
}

const UPGRADE_GUIDANCE =
  'durable tasks on Vercel need Fluid compute (WebSockets are available on Fluid compute only), the @vercel/functions and ws packages, and `vc dev` rather than `next dev` locally; pass `durable: false` to serve this endpoint without them';

function fromProcessEnv(
  value: string | (() => string | undefined) | undefined,
  name: 'HATCHET_SIGNING_SECRET' | 'HATCHET_ENDPOINT_ID'
): string | undefined {
  if (typeof value === 'function') {
    return value();
  }

  return value ?? process.env[name];
}

async function loadVercelFunctions(out: ConsoleLike): Promise<VercelFunctions | undefined> {
  try {
    const [functions] = await Promise.all([import('@vercel/functions'), import('ws')]);

    return {
      experimental_upgradeWebSocket: functions.experimental_upgradeWebSocket,
      waitUntil: functions.waitUntil,
    };
  } catch (err) {
    out.warn(
      `[hatchet] durable tasks are not served by this endpoint: ${err instanceof Error ? err.message : String(err)}; run \`npm install @vercel/functions ws\` to serve them`
    );

    return undefined;
  }
}

/**
 * The request re-addressed to the handler's route from the catch-all segment, so the route
 * file's location never has to match `basePath`. Without a route context the request is
 * used as is.
 */
async function routed(
  request: Request,
  basePath: string,
  context: VercelRouteContext | undefined
): Promise<Request> {
  const params = context?.params ? await context.params : undefined;
  const segments = params ? Object.values(params).find(Array.isArray) : undefined;

  if (!segments) {
    return request;
  }

  const url = new URL(request.url);
  url.pathname = `${basePath}/${segments.join('/')}`;

  const withBody = request.method !== 'GET' && request.method !== 'HEAD';

  return new Request(url, {
    method: request.method,
    headers: request.headers,
    body: withBody ? await request.arrayBuffer() : undefined,
  });
}

export function vercel(options: VercelOptions): VercelRoutes {
  const {
    secret,
    endpointId,
    durable = process.env.VERCEL !== undefined,
    maxPayload = OPERATOR_MAX_FRAME_BYTES,
    basePath = '/api/hatchet',
    ...rest
  } = options;
  const out: ConsoleLike = options.console ?? console;
  // Loaded once per module instance; a request never waits on the import twice.
  const functions: Promise<VercelFunctions | undefined> = durable
    ? loadVercelFunctions(out)
    : Promise.resolve(undefined);

  const handler = createOptionalDurableHandler(
    {
      ...rest,
      basePath,
      runtime: { name: 'vercel' },
      secret: () => fromProcessEnv(secret, 'HATCHET_SIGNING_SECRET'),
      endpointId: () => fromProcessEnv(endpointId, 'HATCHET_ENDPOINT_ID'),
    },
    functions.then((loaded) => loaded !== undefined)
  );

  let guided = false;
  const guide = (problem: string) => {
    if (guided) return;
    guided = true;
    out.warn(`[hatchet] durable upgrade refused: ${problem}. ${UPGRADE_GUIDANCE}`);
  };

  const POST: VercelRouteHandler = async (request, context) => {
    const selected = await handler.select();

    return selected.fetch(await routed(request, handler.basePath, context), process.env);
  };

  const GET: VercelRouteHandler = async (request, context) => {
    const [selected, loaded] = await Promise.all([handler.select(), functions]);

    return selected.fetch(await routed(request, handler.basePath, context), process.env, {
      waitUntil: loaded?.waitUntil,
      upgrade: async (_request, run) => {
        if (!loaded) {
          guide('@vercel/functions or ws is not installed, or durable is off');
          return triggerError(426, 'this endpoint does not serve durable tasks', false);
        }

        try {
          // The upgrade writes the 101 on the raw socket and resolves with a placeholder
          // response once the callback settles. The callback awaits the relay, so the
          // route stays open for the invocation's lifetime; waitUntil registers the same
          // promise with the runtime so it is accounted for like any background work.
          return await loaded.experimental_upgradeWebSocket(
            async (ws) => {
              const relay = run(nodeSocket(ws));

              loaded.waitUntil(relay);
              await relay;
            },
            { maxPayload }
          );
        } catch (err) {
          const message = err instanceof Error ? err.message : String(err);

          guide(message);

          return triggerError(426, `websocket upgrade unavailable: ${message}`, false);
        }
      },
    });
  };

  return { GET, POST };
}

/**
 * The endpoint as an `http.Server` for a standalone Vercel function (`api/hatchet.ts` with
 * `export default vercelServer({ workflows })`), where Vercel serves the exported server and
 * the durable upgrade goes through `ws` directly. The function only sees its own path unless
 * `vercel.json` rewrites `<basePath>/:path*` to it.
 */
export function vercelServer(options: Omit<NodeOptions, 'runtime'>): Server {
  return createServer({ ...options, runtime: { name: 'vercel' } });
}
