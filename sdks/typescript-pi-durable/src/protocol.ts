import { z } from 'zod';

/** Session ids appear in stream topic names and concurrency keys, so they stay short and URL-safe. */
export const SessionId = z.string().regex(/^[A-Za-z0-9_-]{1,64}$/);

/**
 * The two topics of one session of the task `taskName`. `namespace` is the client's namespace, so deployments in
 * different namespaces of one tenant never share a session's topics.
 * - `inbox`: clients publish here; the host reads it from the cursor it last committed. Never reset.
 * - `view`: the host publishes a full SessionView here after commits, and resets the topic before each one, so a
 *   reader that connects without a cursor starts at the newest view.
 */
export function sessionTopics(taskName: string, sessionId: string, namespace = '') {
  const prefix = `${namespace}${taskName}`;
  return { inbox: `${prefix}-inbox-${sessionId}`, view: `${prefix}-view-${sessionId}` };
}

const conversationId = z.number().int().nonnegative();

/** What a client may ask for. The client adds the request id that makes redelivery harmless. */
export const ClientMessage = z.discriminatedUnion('type', [
  z.object({
    type: z.literal('input'),
    conversationId,
    text: z.string().trim().min(1).max(100_000),
    whenBusy: z.enum(['steer', 'followUp']).default('followUp'),
  }),
  z.object({ type: z.literal('fork'), conversationId, entryId: z.number().int().nonnegative() }),
  z.object({ type: z.literal('abort'), conversationId }),
]);
export type ClientMessage = z.input<typeof ClientMessage>;

export const InboxMessage = ClientMessage.and(z.object({ requestId: z.uuid() }));
export type InboxMessage = z.infer<typeof InboxMessage>;

export type ToolCallView = { id: string; name: string; arguments: unknown };

export type ItemView =
  | { id: number; kind: 'user'; text: string }
  | {
      id: number;
      kind: 'assistant';
      text: string;
      thinking: string;
      toolCalls: ToolCallView[];
      model?: string;
    }
  | { id: number; kind: 'tool'; callId: string; name: string; isError: boolean; output: string }
  | { id: number; kind: 'note'; text: string };

export type LiveToolView = { callId: string; name: string; status: string; output: string };

export type ConversationSummary = {
  id: number;
  parent?: { conversationId: number; entryId: number };
  sandboxId?: string;
  snapshotId?: string;
  busy: boolean;
  inbox: number;
  usage: { input: number; output: number; cacheRead: number; cost: number };
  /** The newest entries, 50 to 100 of them. `before` is the entry just before them, absent when they start the transcript. */
  items: ItemView[];
  before?: number;
  /** Why the newest run ended without an answer, until the next run starts: `aborted`, `model_error`, `interrupted`, … */
  failure?: { reason: string; detail?: string };
  live?: { text: string; thinking: string; retry?: string; tools: LiveToolView[] };
};

/** A message the host could not apply, such as an input for a conversation that does not exist. */
export type RejectedMessage = {
  requestId: string;
  type: ClientMessage['type'];
  conversationId: number;
  reason: string;
  at: number;
};

export type SessionView = {
  sessionId: string;
  /** The Hatchet task run that owns the session, and its attempt; a retry after a crash has a higher attempt. */
  host: {
    runId: string;
    attempt: number;
    state: 'starting' | 'running' | 'stopped';
    startedAt: number;
  };
  recoveredTasks: number;
  conversations: ConversationSummary[];
  /** The most recent messages the host could not apply, oldest first. */
  rejected: RejectedMessage[];
  publishedAt: number;
};
