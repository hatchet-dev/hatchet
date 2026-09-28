/**
 * `@hatchet-dev/serverless/node`: the adapter for a Node process you run yourself, such as a
 * Next.js custom server, an Express app or a bare `http.Server`.
 *
 * ```ts
 * import { createServer } from '@hatchet-dev/serverless/node';
 * import { workflows } from './tasks';
 *
 * createServer({ workflows }).listen(3000);
 * ```
 *
 * `nodeHandler` returns an `http` request listener that answers `POST <basePath>/healthcheck`
 * and `POST <basePath>/trigger`, calls `next()` outside `basePath` when it is mounted as
 * Express middleware and answers 404 there otherwise, and carries an `upgrade` listener for
 * the durable websocket relay on `<basePath>/trigger`. The relay needs the `ws` package,
 * an optional peer dependency: without it the healthcheck reports `durable.supported: false`
 * and the operator sends only non-durable tasks. The signing secret comes from
 * `process.env.HATCHET_SIGNING_SECRET` unless `secret` says otherwise.
 * @module Node
 */
import {
  STATUS_CODES,
  createServer as createHttpServer,
  type IncomingMessage,
  type Server,
  type ServerResponse,
} from 'node:http';
import type { Duplex } from 'node:stream';
import type { WebSocket, WebSocketServer } from 'ws';
import type { DurableSocket, HandlerOptions, ServerlessRuntimeName } from '../handler';
import type { ConsoleLike } from '../handler/context';
import { triggerError } from '../handler/http';
import { createOptionalDurableHandler } from './optional-durable';

/** The largest frame the operator's relay accepts (pkg/serverlessoperator/durable/relay.go). */
export const OPERATOR_MAX_FRAME_BYTES = 4 * 1024 * 1024;

export interface NodeOptions extends Omit<
  HandlerOptions,
  'secret' | 'endpointId' | 'runtime' | 'durable'
> {
  /** The signing secret; defaults to `process.env.HATCHET_SIGNING_SECRET`. */
  secret?: string | (() => string | undefined);
  /** The endpoint id; defaults to `process.env.HATCHET_ENDPOINT_ID`. */
  endpointId?: string | (() => string | undefined);
  /**
   * Whether to serve durable tasks over the websocket relay. Defaults to true when the `ws`
   * package is installed; false leaves durable tasks to another endpoint.
   */
  durable?: boolean;
  /** The largest websocket frame accepted; defaults to the operator's 4 MiB limit. */
  maxPayload?: number;
  /** The runtime name reported in the healthcheck. Defaults to `"node"`. */
  runtime?: { name: ServerlessRuntimeName };
}

/**
 * An `http.Server` request listener that is also Express middleware (it calls `next` outside
 * `basePath` when given one) and carries the websocket `upgrade` listener.
 */
export interface NodeListener {
  (req: IncomingMessage, res: ServerResponse, next?: (err?: unknown) => void): void;
  /** The normalized base path the routes live under. */
  readonly basePath: string;
  /** Whether the request targets a route under `basePath`. */
  matches(req: IncomingMessage): boolean;
  /**
   * The `upgrade` listener. Returns false, leaving the socket untouched, when the request is
   * not a durable upgrade under `basePath`, so other websocket servers on the same `http.Server`
   * can take it.
   */
  upgrade(req: IncomingMessage, socket: Duplex, head: Buffer): boolean;
}

type WebSocketServerClass = typeof WebSocketServer;

function fromProcessEnv(
  value: string | (() => string | undefined) | undefined,
  name: 'HATCHET_SIGNING_SECRET' | 'HATCHET_ENDPOINT_ID'
): string | undefined {
  if (typeof value === 'function') {
    return value();
  }

  return value ?? process.env[name];
}

/** Adapts a `ws` socket to what the relay expects. */
export function nodeSocket(ws: WebSocket): DurableSocket {
  return {
    send: (text) => ws.send(text),
    close: (code, reason) => ws.close(code, reason),
    onMessage: (listener) =>
      ws.on('message', (data, isBinary) => {
        if (isBinary) {
          ws.close(1003, 'text frames only');
        } else {
          listener(data.toString());
        }
      }),
    onClose: (listener) => {
      let closed = false;
      const once = (code: number, reason: string) => {
        if (closed) return;
        closed = true;
        listener(code, reason);
      };
      ws.on('close', (code, reason) => once(code, reason.toString()));
      ws.on('error', () => once(1006, 'socket error'));
    },
  };
}

/** The request's URL as the client sent it, before an Express mount path was stripped. */
function requestUrl(req: IncomingMessage): URL {
  const path = (req as { originalUrl?: string }).originalUrl ?? req.url ?? '/';

  return new URL(path, `http://${req.headers.host ?? 'localhost'}`);
}

function readBody(req: IncomingMessage): Promise<Buffer> {
  return new Promise((resolve, reject) => {
    const chunks: Buffer[] = [];
    req.on('data', (chunk: Buffer) => chunks.push(chunk));
    req.on('end', () => resolve(Buffer.concat(chunks)));
    req.on('error', reject);
  });
}

async function toRequest(req: IncomingMessage, url: URL, withBody: boolean): Promise<Request> {
  const headers = new Headers();

  for (const [name, value] of Object.entries(req.headers)) {
    if (Array.isArray(value)) {
      for (const item of value) headers.append(name, item);
    } else if (value !== undefined) {
      headers.set(name, value);
    }
  }

  const method = req.method ?? 'GET';
  const body = withBody && method !== 'GET' && method !== 'HEAD' ? await readBody(req) : undefined;

  return new Request(url, { method, headers, body });
}

async function sendResponse(res: ServerResponse, response: Response): Promise<void> {
  const body = Buffer.from(await response.arrayBuffer());

  res.statusCode = response.status;
  response.headers.forEach((value, name) => res.setHeader(name, value));
  res.setHeader('content-length', body.length);
  res.end(body);
}

/** Writes a non-101 response for an upgrade request straight onto the raw socket. */
async function refuseUpgrade(socket: Duplex, response: Response): Promise<void> {
  const body = Buffer.from(await response.arrayBuffer());
  let head = `HTTP/1.1 ${response.status} ${STATUS_CODES[response.status] ?? ''}\r\n`;

  response.headers.forEach((value, name) => {
    head += `${name}: ${value}\r\n`;
  });
  head += `content-length: ${body.length}\r\nconnection: close\r\n\r\n`;

  socket.end(Buffer.concat([Buffer.from(head), body]));
}

function loadWebSocketServer(out: ConsoleLike): Promise<WebSocketServerClass | undefined> {
  return import('ws').then(
    (ws) => ws.WebSocketServer,
    () => {
      out.warn(
        '[hatchet] the ws package is not installed, so durable tasks are not served by this endpoint; run `npm install ws` to serve them'
      );
      return undefined;
    }
  );
}

export function nodeHandler(options: NodeOptions): NodeListener {
  const {
    secret,
    endpointId,
    durable = true,
    maxPayload = OPERATOR_MAX_FRAME_BYTES,
    runtime = { name: 'node' },
    ...rest
  } = options;
  const out: ConsoleLike = options.console ?? console;
  const webSocketServer: Promise<WebSocketServerClass | undefined> = durable
    ? loadWebSocketServer(out)
    : Promise.resolve(undefined);

  const handler = createOptionalDurableHandler(
    {
      ...rest,
      runtime,
      secret: () => fromProcessEnv(secret, 'HATCHET_SIGNING_SECRET'),
      endpointId: () => fromProcessEnv(endpointId, 'HATCHET_ENDPOINT_ID'),
    },
    webSocketServer.then((loaded) => loaded !== undefined)
  );

  const fail = (res: ServerResponse, err: unknown) => {
    out.error(`[hatchet] request failed: ${err instanceof Error ? err.message : String(err)}`);
    if (!res.headersSent) {
      res.statusCode = 500;
      res.setHeader('content-type', 'application/json');
    }
    res.end(JSON.stringify({ error: 'internal error' }));
  };

  const serve = async (req: IncomingMessage, res: ServerResponse, url: URL) => {
    const request = await toRequest(req, url, true);
    const selected = await handler.select();

    await sendResponse(res, await selected.fetch(request, process.env));
  };

  const listener = ((req, res, next) => {
    const url = requestUrl(req);

    if (!handler.matches(url)) {
      if (next) {
        next();
        return;
      }

      res.statusCode = 404;
      res.setHeader('content-type', 'application/json');
      res.end(JSON.stringify({ error: 'not found' }));
      return;
    }

    serve(req, res, url).catch((err) => fail(res, err));
  }) as NodeListener;

  const acceptUpgrade = async (req: IncomingMessage, socket: Duplex, head: Buffer, url: URL) => {
    const request = await toRequest(req, url, false);
    const Server = await webSocketServer;
    const selected = await handler.select();
    let accepted = false;

    const response = await selected.fetch(request, process.env, {
      upgrade: (_request, run) => {
        if (!Server) {
          return triggerError(
            426,
            'this endpoint does not serve durable tasks: the ws package is not installed',
            false
          );
        }

        const wss = new Server({ noServer: true, maxPayload });

        accepted = true;
        wss.handleUpgrade(req, socket, head, (ws) => {
          void run(nodeSocket(ws));
        });

        // The 101 already went out on the raw socket; the value only marks the hook's outcome
        // (a `Response` cannot carry status 101 in Node).
        return new Response(null, { status: 200 });
      },
    });

    if (!accepted) {
      await refuseUpgrade(socket, response);
    }
  };

  Object.defineProperties(listener, {
    basePath: { value: handler.basePath, enumerable: true },
    matches: { value: (req: IncomingMessage) => handler.matches(requestUrl(req)) },
    upgrade: {
      value: (req: IncomingMessage, socket: Duplex, head: Buffer): boolean => {
        const url = requestUrl(req);

        if (!handler.matches(url) || req.headers.upgrade?.toLowerCase() !== 'websocket') {
          return false;
        }

        acceptUpgrade(req, socket, head, url).catch((err) => {
          out.error(
            `[hatchet] upgrade failed: ${err instanceof Error ? err.message : String(err)}`
          );
          socket.destroy();
        });

        return true;
      },
    },
  });

  return listener;
}

/** An `http.Server` serving the routes and the durable upgrade, ready to `listen`. */
export function createServer(options: NodeOptions): Server {
  const listener = nodeHandler(options);
  const server = createHttpServer(listener);

  server.on('upgrade', (req, socket, head) => {
    if (!listener.upgrade(req, socket, head)) {
      socket.end('HTTP/1.1 404 Not Found\r\ncontent-length: 0\r\nconnection: close\r\n\r\n');
    }
  });

  return server;
}
