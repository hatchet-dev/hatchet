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
 * `process.env.HATCHET_SIGNING_SECRET` unless `secret` says otherwise, and the tasks' client,
 * unless `client` is given, from `HATCHET_CLIENT_TOKEN` (with `HATCHET_CLIENT_HOST_PORT` and
 * `HATCHET_CLIENT_TLS_STRATEGY` for a local engine), the variables the Node SDK reads too.
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
import { OPERATOR_MAX_FRAME_BYTES } from '../handler/contract';
import { bodyTooLarge, errorMessage, triggerError } from '../handler/http';
import { clientConfigFromEnv } from '../handler/client';
import { createOptionalDurableHandler } from './optional-durable';

export { OPERATOR_MAX_FRAME_BYTES };

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
  /**
   * The largest POST body accepted, in bytes. The body is read up to this many bytes and
   * no further: past it the request is answered 413 and its socket destroyed. Defaults to
   * the operator's 4 MiB frame limit.
   */
  maxBodyBytes?: number;
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

/**
 * The request's URL as the client sent it, before an Express mount path was stripped. Only
 * its path routes, so a `Host` the URL parser rejects (Node's parser accepts more) is
 * replaced by a fixed authority; `undefined` means the request target itself does not parse
 * and the request is answered 400.
 */
function requestUrl(req: IncomingMessage): URL | undefined {
  const path = (req as { originalUrl?: string }).originalUrl ?? req.url ?? '/';

  for (const authority of [req.headers.host, 'localhost']) {
    try {
      return new URL(path, `http://${authority ?? 'localhost'}`);
    } catch {
      // Try the next authority.
    }
  }

  return undefined;
}

class BodyTooLargeError extends Error {
  constructor(readonly maxBytes: number) {
    super(`the request body exceeds the ${maxBytes} byte limit`);
    this.name = 'BodyTooLargeError';
  }
}

/** Reads the body up to `maxBytes`; past that nothing more is read and the read rejects. */
function readBody(req: IncomingMessage, maxBytes: number): Promise<Buffer> {
  return new Promise((resolve, reject) => {
    const declared = Number(req.headers['content-length']);

    if (Number.isFinite(declared) && declared > maxBytes) {
      reject(new BodyTooLargeError(maxBytes));
      return;
    }

    const chunks: Buffer[] = [];
    let received = 0;

    req.on('data', (chunk: Buffer) => {
      received += chunk.length;

      if (received > maxBytes) {
        req.pause();
        reject(new BodyTooLargeError(maxBytes));
        return;
      }

      chunks.push(chunk);
    });
    req.on('end', () => resolve(Buffer.concat(chunks)));
    req.on('error', reject);
  });
}

async function toRequest(
  req: IncomingMessage,
  res: ServerResponse | undefined,
  url: URL,
  maxBodyBytes: number
): Promise<Request> {
  const headers = new Headers();

  for (const [name, value] of Object.entries(req.headers)) {
    if (Array.isArray(value)) {
      for (const item of value) headers.append(name, item);
    } else if (value !== undefined) {
      headers.set(name, value);
    }
  }

  const method = req.method ?? 'GET';
  const body =
    res && method !== 'GET' && method !== 'HEAD' ? await readBody(req, maxBodyBytes) : undefined;

  // The operator cancels a delivery by dropping the connection: a response closed before it
  // finished aborts the request the handler sees, and the task with it.
  const controller = new AbortController();

  res?.on('close', () => {
    if (!res.writableFinished) {
      controller.abort();
    }
  });

  return new Request(url, { method, headers, body, signal: controller.signal });
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
    maxBodyBytes = OPERATOR_MAX_FRAME_BYTES,
    runtime = { name: 'node' },
    client = () => clientConfigFromEnv(process.env),
    ...rest
  } = options;
  const out: ConsoleLike = options.console ?? console;
  const webSocketServer: Promise<WebSocketServerClass | undefined> = durable
    ? loadWebSocketServer(out)
    : Promise.resolve(undefined);

  const handler = createOptionalDurableHandler(
    {
      ...rest,
      client,
      maxBodyBytes,
      runtime,
      secret: () => fromProcessEnv(secret, 'HATCHET_SIGNING_SECRET'),
      endpointId: () => fromProcessEnv(endpointId, 'HATCHET_ENDPOINT_ID'),
    },
    webSocketServer.then((loaded) => loaded !== undefined)
  );

  const answer = (res: ServerResponse, status: number, body: unknown) => {
    if (!res.headersSent) {
      res.statusCode = status;
      res.setHeader('content-type', 'application/json');
    }
    res.end(JSON.stringify(body));
  };

  const fail = (res: ServerResponse, err: unknown) => {
    out.error(`[hatchet] request failed: ${errorMessage(err)}`);
    answer(res, 500, { error: 'internal error' });
  };

  // The body past the cap is never read: the 413 goes out with `connection: close` and the
  // socket is destroyed once it has been written, so the rest of the upload is dropped.
  const tooLarge = async (req: IncomingMessage, res: ServerResponse, maxBytes: number) => {
    const body = Buffer.from(await bodyTooLarge(maxBytes).arrayBuffer());

    res.statusCode = 413;
    res.setHeader('content-type', 'application/json');
    res.setHeader('content-length', body.length);
    res.setHeader('connection', 'close');
    res.end(body, () => req.socket?.destroy());
  };

  const serve = async (req: IncomingMessage, res: ServerResponse, url: URL) => {
    let request: Request;

    try {
      request = await toRequest(req, res, url, maxBodyBytes);
    } catch (err) {
      if (err instanceof BodyTooLargeError) {
        await tooLarge(req, res, err.maxBytes);
        return;
      }

      throw err;
    }

    const selected = await handler.select();

    await sendResponse(res, await selected.fetch(request, process.env));
  };

  const listener = ((req, res, next) => {
    try {
      const url = requestUrl(req);

      if (!url) {
        answer(res, 400, { error: 'invalid request target' });
        return;
      }

      if (!handler.matches(url)) {
        if (next) {
          next();
          return;
        }

        answer(res, 404, { error: 'not found' });
        return;
      }

      serve(req, res, url).catch((err) => fail(res, err));
    } catch (err) {
      fail(res, err);
    }
  }) as NodeListener;

  const acceptUpgrade = async (req: IncomingMessage, socket: Duplex, head: Buffer, url: URL) => {
    const request = await toRequest(req, undefined, url, maxBodyBytes);
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

  // A failure before the 101 is answered on the raw socket; nothing request-derived goes
  // into the response line.
  const refuseWith = (socket: Duplex, status: number, error: string) => {
    refuseUpgrade(socket, triggerError(status, error, false)).catch(() => socket.destroy());
  };

  const matches = (req: IncomingMessage): boolean => {
    const url = requestUrl(req);

    return url !== undefined && handler.matches(url);
  };

  Object.defineProperties(listener, {
    basePath: { value: handler.basePath, enumerable: true },
    matches: { value: matches },
    upgrade: {
      value: (req: IncomingMessage, socket: Duplex, head: Buffer): boolean => {
        if (req.headers.upgrade?.toLowerCase() !== 'websocket') {
          return false;
        }

        try {
          const url = requestUrl(req);

          if (!url) {
            refuseWith(socket, 400, 'invalid request target');
            return true;
          }

          if (!handler.matches(url)) {
            return false;
          }

          acceptUpgrade(req, socket, head, url).catch((err) => {
            out.error(`[hatchet] upgrade failed: ${errorMessage(err)}`);
            refuseWith(socket, 500, 'internal error');
          });
        } catch (err) {
          out.error(`[hatchet] upgrade failed: ${errorMessage(err)}`);
          refuseWith(socket, 500, 'internal error');
        }

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
