/**
 * Engine streams multiplexed over the invocation socket. A task cannot hold a stream to the
 * engine itself, so it asks the operator to: a `stream_open` frame names the procedure and
 * the first request, the operator relays the engine's messages back as `stream_message`
 * frames carrying the endpoint-chosen id, and a `stream_close` in either direction ends the
 * stream. The operator makes the engine call with its own credentials, so no token is
 * involved on this path.
 *
 * The operator allows three procedures and at most `MAX_SOCKET_STREAMS` open streams per
 * socket; both limits are mirrored here so a task learns about them from a local error
 * instead of a closed stream. Every stream is closed when the task ends, before the done
 * frame, and every listener hears about the socket going away.
 */
import type { ServerlessDurableFrame } from '../../generated/proto/v1/serverless';
import type { ConsoleLike } from '../context';
import { encodeFrame } from './frames';
import type { DurableSocket } from './socket';

/** The engine streams the operator opens on a task's behalf, by fully qualified procedure. */
export const SUBSCRIBE_TO_WORKFLOW_RUNS = '/Dispatcher/SubscribeToWorkflowRuns';
export const SUBSCRIBE_TO_WORKFLOW_EVENTS = '/Dispatcher/SubscribeToWorkflowEvents';
export const LISTEN_FOR_DURABLE_EVENT = '/v1.V1Dispatcher/ListenForDurableEvent';

export const SOCKET_STREAM_PROCEDURES: readonly string[] = [
  SUBSCRIBE_TO_WORKFLOW_RUNS,
  SUBSCRIBE_TO_WORKFLOW_EVENTS,
  LISTEN_FOR_DURABLE_EVENT,
];

/** The operator's per-socket cap on open streams (`WSMaxStreams`). */
export const MAX_SOCKET_STREAMS = 16;

/** The connect codes a stream_close carries, as the operator numbers them. */
export const StreamCloseCode = {
  OK: 0,
  CANCELED: 1,
  UNKNOWN: 2,
  INVALID_ARGUMENT: 3,
  PERMISSION_DENIED: 7,
  RESOURCE_EXHAUSTED: 8,
  UNIMPLEMENTED: 12,
  UNAVAILABLE: 14,
} as const;

/** Raised when a stream cannot be opened: the cap is reached, the procedure is not served or the socket is gone. */
export class StreamOpenError extends Error {
  constructor(message: string) {
    super(message);
    this.name = 'StreamOpenError';
  }
}

export interface StreamListener {
  /** A message the engine sent on the stream, as the protojson document the operator relayed. */
  onMessage(message: string): void;
  /**
   * The stream ended without the task closing it: code 0 when the engine finished it, a
   * connect code when the operator or the engine refused or dropped it, `UNAVAILABLE` when
   * the socket itself went away. Called at most once, and never after `close()`.
   */
  onClose(code: number, message: string): void;
}

/** One open stream, as the task sees it. */
export interface SocketStream {
  readonly id: string;
  readonly procedure: string;
  /** Sends a further request on a bidi stream (a subscription on `SubscribeToWorkflowRuns`). */
  send(message: string): void;
  /** Ends the stream; the operator cancels the engine call. Idempotent. */
  close(): void;
}

export interface SocketStreamsOptions {
  socket: DurableSocket;
  /** The per-socket cap; defaults to the operator's `MAX_SOCKET_STREAMS`. */
  maxStreams?: number;
  console?: ConsoleLike;
}

interface OpenStream {
  handle: SocketStream;
  listener: StreamListener;
}

export class SocketStreams {
  private readonly open = new Map<string, OpenStream>();
  private readonly maxStreams: number;
  private readonly out: ConsoleLike;
  private nextId = 0;
  /** Set once the socket is gone or the task is over; no stream can be opened past it. */
  private ended: string | undefined;

  constructor(private readonly options: SocketStreamsOptions) {
    this.maxStreams = options.maxStreams ?? MAX_SOCKET_STREAMS;
    this.out = options.console ?? console;
  }

  /** How many streams are open right now. */
  get size(): number {
    return this.open.size;
  }

  /**
   * Opens an engine stream: sends `stream_open` with the procedure and the first request as a
   * protojson document. Throws a `StreamOpenError` when the procedure is not one the operator
   * serves, the cap is reached or the socket is gone.
   */
  openStream(procedure: string, request: string, listener: StreamListener): SocketStream {
    if (this.ended) {
      throw new StreamOpenError(`cannot open ${procedure}: ${this.ended}`);
    }

    if (!SOCKET_STREAM_PROCEDURES.includes(procedure)) {
      throw new StreamOpenError(
        `cannot open ${procedure}: the operator serves only ${SOCKET_STREAM_PROCEDURES.join(', ')} over the invocation socket`
      );
    }

    if (this.open.size >= this.maxStreams) {
      throw new StreamOpenError(
        `cannot open ${procedure}: the invocation socket already holds ${this.open.size} streams, the operator's limit`
      );
    }

    this.nextId += 1;
    const id = String(this.nextId);

    const handle: SocketStream = {
      id,
      procedure,
      send: (message) => {
        if (!this.open.has(id)) {
          throw new Error(`stream ${id} (${procedure}) is closed`);
        }

        this.send({ streamMessage: { id, message } });
      },
      close: () => {
        if (!this.open.delete(id)) {
          return;
        }

        if (!this.ended) {
          this.trySend({ streamClose: { id, code: 0, message: '' } });
        }
      },
    };

    this.open.set(id, { handle, listener });
    // The slot is taken before the frame goes out, so a send failure leaves nothing behind.
    try {
      this.send({ streamOpen: { id, procedure, request } });
    } catch (err) {
      this.open.delete(id);
      throw new StreamOpenError(
        `cannot open ${procedure}: ${err instanceof Error ? err.message : String(err)}`
      );
    }

    return handle;
  }

  /** Routes a stream frame from the operator; reports whether the frame was one. */
  handleFrame(frame: ServerlessDurableFrame): boolean {
    if (frame.streamMessage) {
      const { id, message } = frame.streamMessage;
      const stream = this.open.get(id);

      if (!stream) {
        // A message crossing the task's own close is expected; nothing to route it to.
        this.out.debug(`[hatchet] dropping stream_message for stream ${id}, which is not open`);
        return true;
      }

      stream.listener.onMessage(message);
      return true;
    }

    if (frame.streamClose) {
      const { id, code, message } = frame.streamClose;
      const stream = this.open.get(id);

      if (!stream) {
        return true;
      }

      this.open.delete(id);
      stream.listener.onClose(code, message);
      return true;
    }

    return false;
  }

  /**
   * Closes every open stream, telling the operator: the task is over and its done frame is
   * about to go out. Listeners are not called: nothing awaits them once the task is over.
   */
  closeAll(): void {
    if (this.ended) {
      return;
    }

    this.ended = 'the invocation is over';

    for (const id of [...this.open.keys()]) {
      this.open.delete(id);
      this.trySend({ streamClose: { id, code: 0, message: '' } });
    }
  }

  /**
   * The socket went away: nothing more can be sent, and every listener hears the stream end
   * with `UNAVAILABLE` and the reason.
   */
  fail(reason: string): void {
    if (this.ended) {
      return;
    }

    this.ended = reason;

    for (const [id, stream] of [...this.open]) {
      this.open.delete(id);
      stream.listener.onClose(StreamCloseCode.UNAVAILABLE, reason);
    }
  }

  private send(frame: ServerlessDurableFrame): void {
    this.options.socket.send(encodeFrame(frame));
  }

  private trySend(frame: ServerlessDurableFrame): void {
    try {
      this.send(frame);
    } catch {
      // The socket is closed; the operator drops the stream with it.
    }
  }
}
