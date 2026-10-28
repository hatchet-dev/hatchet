/**
 * What a durable websocket looks like to the relay. Cloudflare's server-side WebSocket,
 * `ws`, Bun's and Deno's sockets all fit; adapters wrap theirs.
 */
export interface DurableSocket {
  send(text: string): void;
  close(code?: number, reason?: string): void;
  onMessage(listener: (text: string) => void): void;
  onClose(listener: (code: number, reason: string) => void): void;
}

/**
 * How an adapter lets the handler run the durable relay. `upgrade` accepts the websocket,
 * starts `run` with the server side of it and returns the switching-protocols response;
 * `waitUntil` keeps the runtime alive until `run` settles where that needs asking for.
 * Without `upgrade` the handler answers durable upgrades with 426, and the healthcheck
 * reports `durable.supported: false` so the operator never dials.
 */
export interface DurableHooks {
  upgrade?(
    request: Request,
    run: (socket: DurableSocket) => Promise<void>
  ): Response | Promise<Response>;
  waitUntil?(promise: Promise<unknown>): void;
}
