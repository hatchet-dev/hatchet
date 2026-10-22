/**
 * The client a task gets from `clientFromContext(ctx)`: the handler's `HatchetCore` seen
 * through the invocation socket. Every unary call is the client's own. Reading a durable
 * stream topic (`streams.events`) and a run's events (`runs.events`) are server streams, which
 * the unary client does not carry; here each opens one engine stream through the operator
 * over the socket (`SocketStreams`), the way child results are awaited, and reads it as an
 * async iterable with the Node client's shapes (`StreamEvent`, `StreamCallOptions`).
 *
 * The view inherits from the client with `streams` and `runs` replaced by views of their own,
 * so every other member, `instanceof HatchetCore` and the producer state behind
 * `streams.publish` are the client's. Without a socket (a task delivered over a POST) the view
 * serves every unary call and the two iterators throw `ServerlessLimitationError` saying where
 * the socket went.
 *
 * Both iterators are async generators, as the Node client's `events()` is: `break` and
 * `return()` close the stream between messages, and `options.signal` ends a read that is
 * waiting for the next one.
 */
import {
  HatchetError,
  type HatchetCore,
  type RunsClient,
  type StreamCallOptions,
  type StreamsClient,
} from '@hatchet-dev/typescript-sdk/core/index.js';
import { SubscribeToWorkflowEventsRequest, WorkflowEvent } from '../generated/proto/dispatcher';
import { StreamMessage, SubscribeStreamRequest } from '../generated/proto/v1/streams';
import {
  SUBSCRIBE_STREAM,
  SUBSCRIBE_TO_WORKFLOW_EVENTS,
  StreamCloseCode,
  streamCloseCodeName,
  type SocketStreams,
} from './durable/streams';
import { ServerlessLimitationError } from './errors';

/** One message of a durable stream topic, as the Node client's `streams.events` yields it. */
export interface StreamEvent {
  /** The message payload. */
  payload: Uint8Array;
  /** Resumes `events()` right after this message. */
  cursor: string;
  /** When the message was stored. */
  createdAt?: Date;
}

export interface RunEventsOptions {
  /** Ends the iteration cleanly, without throwing. */
  signal?: AbortSignal;
}

/**
 * The operator or the engine ended a stream the task was reading, with a connect code other
 * than 0. `code` is the stream_close code (7, permission denied, on a topic is the engine
 * refusing a tenant without the durable streams entitlement; 12, unimplemented, an operator
 * that does not relay the procedure) and `cursor` the last message yielded, to resume from.
 */
export class StreamClosedError extends Error {
  constructor(
    /** What was being read, as the message names it. */
    readonly subject: string,
    readonly code: number,
    detail: string,
    hint?: string,
    readonly cursor?: string
  ) {
    const name = streamCloseCodeName(code);
    // The operator's own messages start with the code's name; it is not repeated.
    const text = detail.startsWith(`${name}: `) ? detail.slice(name.length + 2) : detail;

    super(
      `${subject} closed with code ${code} (${name}${text ? `: ${text}` : ''})` +
        (hint ? `: ${hint}` : '')
    );
    this.name = 'StreamClosedError';
  }
}

/**
 * The streams client of a task: the handler's public surface, plus reading a topic over the
 * socket. The intersection is over the public members only (the class has private ones of
 * its own, one of them named `events`); the value is the client's own `StreamsClient`.
 */
export type TaskStreamsClient = Pick<StreamsClient, keyof StreamsClient> & {
  /**
   * Yields the topic's messages after `options.cursor`, or from the oldest retained (or,
   * once the topic is reset, after the reset), over the invocation socket. Ends when the
   * engine hangs up an idle subscription, `options.signal` aborts, or the task stops
   * iterating, which closes the stream and frees its socket slot; throws `StreamClosedError`
   * when the stream ends any other way. Unlike the Node client it does not resubscribe: the
   * error carries the cursor to resume from.
   */
  events(topic: string, options?: StreamCallOptions): AsyncIterable<StreamEvent>;
};

/** The runs client of a task: the handler's public surface, plus a run's events over the socket. */
export type TaskRunsClient = Pick<RunsClient, keyof RunsClient> & {
  /**
   * Yields the run's events (`/Dispatcher/SubscribeToWorkflowEvents`) over the invocation
   * socket until the engine hangs up after the run's last event, `options.signal` aborts, or
   * the task stops iterating.
   */
  events(workflowRunId: string, options?: RunEventsOptions): AsyncIterable<WorkflowEvent>;
};

/**
 * The handler's client as a task sees it: the public surface of `HatchetCore` with `streams`
 * and `runs` carrying the socket-backed iterators. The value is an instance of `HatchetCore`.
 */
export type TaskClient = Omit<HatchetCore, 'streams' | 'runs'> & {
  readonly streams: TaskStreamsClient;
  readonly runs: TaskRunsClient;
};

type Relayed = { message: string } | { closed: { code: number; detail: string } };

/**
 * Reads one engine stream over the socket as the protojson documents the operator relays.
 * Ends on a stream_close with code 0 or when the signal aborts; throws `closed(code, detail)`
 * on any other close. The stream is closed, freeing its slot, however the iteration ends.
 */
async function* relayed(
  streams: SocketStreams,
  procedure: string,
  request: string,
  signal: AbortSignal | undefined,
  closed: (code: number, detail: string) => Error
): AsyncGenerator<string, void, undefined> {
  if (signal?.aborted) {
    return;
  }

  const queue: Relayed[] = [];
  let aborted = false;
  let wake: (() => void) | undefined;

  const push = (item: Relayed) => {
    queue.push(item);
    wake?.();
    wake = undefined;
  };
  const onAbort = () => {
    aborted = true;
    wake?.();
    wake = undefined;
  };

  const stream = streams.openStream(procedure, request, {
    onMessage: (message) => push({ message }),
    onClose: (code, detail) => push({ closed: { code, detail } }),
  });
  signal?.addEventListener('abort', onAbort, { once: true });

  try {
    for (;;) {
      if (aborted) {
        return;
      }

      const item = queue.shift();

      if (!item) {
        await new Promise<void>((resolve) => {
          wake = resolve;
        });
        continue;
      }

      if ('closed' in item) {
        if (item.closed.code === StreamCloseCode.OK) {
          return;
        }

        throw closed(item.closed.code, item.closed.detail);
      }

      yield item.message;
    }
  } finally {
    signal?.removeEventListener('abort', onAbort);
    stream.close();
  }
}

const TOPIC_CLOSE_HINTS: Partial<Record<number, string>> = {
  [StreamCloseCode.PERMISSION_DENIED]:
    'durable streams are not enabled for this tenant; the durable_streams entitlement enables them',
  [StreamCloseCode.UNIMPLEMENTED]:
    'the operator does not relay durable stream subscriptions over the socket; deploy an operator that does',
};

async function* topicEvents(
  streams: SocketStreams,
  topic: string,
  options: StreamCallOptions | undefined
): AsyncGenerator<StreamEvent, void, undefined> {
  const namespace = options?.namespace ?? '';
  let cursor = options?.cursor;
  const request = JSON.stringify(
    SubscribeStreamRequest.toJSON(SubscribeStreamRequest.create({ namespace, topic, cursor }))
  );
  const subject = `the subscription to topic "${topic}"`;

  for await (const raw of relayed(
    streams,
    SUBSCRIBE_STREAM,
    request,
    options?.signal,
    (code, detail) => new StreamClosedError(subject, code, detail, TOPIC_CLOSE_HINTS[code], cursor)
  )) {
    const message = StreamMessage.fromJSON(JSON.parse(raw));

    if (message.hangup) {
      return;
    }

    for (const entry of message.entries) {
      if (entry.payloadRef && entry.payload.byteLength === 0) {
        // The Node client fetches an uploaded payload over REST; this client has no REST path.
        throw new HatchetError(
          `message ${entry.cursor} of topic "${topic}" is an uploaded payload (over the inline limit), which the serverless client cannot fetch; read the topic with the Node client`
        );
      }

      ({ cursor } = entry);
      yield { payload: entry.payload, cursor: entry.cursor, createdAt: entry.createdAt };
    }
  }
}

async function* runEvents(
  streams: SocketStreams,
  workflowRunId: string,
  options: RunEventsOptions | undefined
): AsyncGenerator<WorkflowEvent, void, undefined> {
  const request = JSON.stringify(
    SubscribeToWorkflowEventsRequest.toJSON(
      SubscribeToWorkflowEventsRequest.create({ workflowRunId })
    )
  );

  for await (const raw of relayed(
    streams,
    SUBSCRIBE_TO_WORKFLOW_EVENTS,
    request,
    options?.signal,
    (code, detail) =>
      new StreamClosedError(`the event stream of run ${workflowRunId}`, code, detail)
  )) {
    const event = WorkflowEvent.fromJSON(JSON.parse(raw));

    yield event;

    if (event.hangup) {
      return;
    }
  }
}

function noSocket(feature: string): ServerlessLimitationError {
  return new ServerlessLimitationError(
    feature,
    'It reads an engine stream the operator relays over the invocation socket, which the operator dials for every task not declared with `hatchetClient: false`; upgrade the operator, or serve the task from an endpoint whose adapter accepts the upgrade (`durable` on). From a Node process, use the Node client',
    'this task was invoked over a POST and has no invocation socket, which happens with an operator that predates stream delivery or an endpoint reporting no durable support'
  );
}

/** An object inheriting from `base` with `members` as its own properties. */
function view<T extends object, M extends object>(base: T, members: M): T & M {
  return Object.create(
    base,
    Object.fromEntries(
      Object.entries(members).map(([name, value]) => [name, { value, enumerable: true }])
    )
  ) as T & M;
}

/**
 * The task's view of the handler's client. With `streams`, the socket of the invocation,
 * `streams.events` and `runs.events` read over it; without one they throw.
 */
export function createTaskClient(
  client: HatchetCore,
  streams: SocketStreams | undefined
): TaskClient {
  const streamsView: TaskStreamsClient = view(client.streams, {
    events: (topic: string, options?: StreamCallOptions) => {
      if (!streams) {
        throw noSocket('clientFromContext(ctx).streams.events(topic)');
      }

      return topicEvents(streams, topic, options);
    },
  });
  const runsView: TaskRunsClient = view(client.runs, {
    events: (workflowRunId: string, options?: RunEventsOptions) => {
      if (!streams) {
        throw noSocket('clientFromContext(ctx).runs.events(runId)');
      }

      return runEvents(streams, workflowRunId, options);
    },
  });

  return view(client, { streams: streamsView, runs: runsView });
}
