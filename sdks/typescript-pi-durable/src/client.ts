// Talking to sessions from anywhere: a web server, a CLI, another task. A client never reaches a session's host
// directly: it publishes to the session's inbox stream, wakes the session, and reads the session's view stream.
import { randomUUID } from 'node:crypto';
import type { HatchetClient } from '@hatchet-dev/typescript-sdk/v1';
import {
  ClientMessage,
  type InboxMessage,
  SessionId,
  type SessionView,
  sessionTopics,
} from './protocol.js';

export type SessionClientOptions = {
  /** The session task's name, as given to `defineSessionTask`. Default `pi-session`. */
  taskName?: string;
};

const decoder = new TextDecoder();

export class SessionClient {
  readonly taskName: string;

  constructor(
    private readonly hatchet: HatchetClient,
    options: SessionClientOptions = {}
  ) {
    this.taskName = options.taskName ?? 'pi-session';
  }

  /** Starts a session under a new id, and returns the id. */
  async create(sessionId: string = randomUUID().replaceAll('-', '').slice(0, 12)): Promise<string> {
    SessionId.parse(sessionId);
    await this.wake(sessionId);
    return sessionId;
  }

  /** Wakes the session: a run queues behind a live owner, and finds nothing to do if that owner already applied it. */
  async wake(sessionId: string): Promise<void> {
    SessionId.parse(sessionId);
    await this.hatchet.runNoWait(this.taskName, { sessionId }, {});
  }

  /** Puts a message in the session's inbox and wakes the session. Returns the request id the host dedupes on. */
  async send(sessionId: string, message: ClientMessage): Promise<string> {
    SessionId.parse(sessionId);
    const inbox: InboxMessage = { ...ClientMessage.parse(message), requestId: randomUUID() };
    await this.hatchet.streams.publish(
      sessionTopics(this.taskName, sessionId, this.hatchet.config.namespace).inbox,
      JSON.stringify(inbox)
    );
    await this.wake(sessionId);
    return inbox.requestId;
  }

  /**
   * The session's views, newest first and then each one the host publishes, until `signal` aborts. The host resets
   * the topic before every view, so reading without a cursor starts at the newest one.
   */
  async *views(sessionId: string, signal?: AbortSignal): AsyncGenerator<SessionView> {
    SessionId.parse(sessionId);
    const topic = sessionTopics(this.taskName, sessionId, this.hatchet.config.namespace).view;
    while (!signal?.aborted) {
      try {
        for await (const event of this.hatchet.streams.events(topic, { signal })) {
          yield JSON.parse(decoder.decode(event.payload)) as SessionView;
        }
      } catch (error) {
        if (signal?.aborted) return;
        throw error;
      }
      // the subscription ended without an abort: follow the topic again from its newest view
    }
  }
}
