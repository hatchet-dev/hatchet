import type { AssistantMessage, Message } from '@earendil-works/pi-ai';
import type {
  ConversationView,
  InboxState,
  LiveState,
  UsageState,
} from '@earendil-works/pi-durable';
import type { SandboxState } from './extension.js';
import type { ConversationSummary, ItemView } from './protocol.js';

const MAX_TEXT_CHARS = 4_000;

const clip = (text: string) =>
  text.length > MAX_TEXT_CHARS
    ? `${text.slice(0, MAX_TEXT_CHARS)}\n… ${text.length - MAX_TEXT_CHARS} more characters`
    : text;

type Content = Message['content'] | undefined;

function textOf(content: Content): string {
  if (typeof content === 'string') return content;
  return (content ?? []).flatMap((block) => (block.type === 'text' ? [block.text] : [])).join('\n');
}

function thinkingOf(content: Content): string {
  if (typeof content === 'string') return '';
  return (content ?? [])
    .flatMap((block) => (block.type === 'thinking' ? [block.thinking] : []))
    .join('\n');
}

function toItem(id: number, kind: string, message: Message | undefined): ItemView | undefined {
  if (kind === 'pi.compaction') return { id, kind: 'note', text: 'Older messages were compacted.' };
  switch (message?.role) {
    case 'user':
      return { id, kind: 'user', text: clip(textOf(message.content)) };
    case 'assistant':
      return {
        id,
        kind: 'assistant',
        text: clip(textOf(message.content)),
        thinking: clip(thinkingOf(message.content)),
        toolCalls: message.content.flatMap((block) =>
          block.type === 'toolCall'
            ? [{ id: block.id, name: block.name, arguments: block.arguments }]
            : []
        ),
        model: message.model,
      };
    case 'toolResult':
      return {
        id,
        kind: 'tool',
        callId: message.toolCallId,
        name: message.toolName,
        isError: message.isError,
        output: clip(textOf(message.content)),
      };
    default:
      return undefined; // system prompt changes and app entries the view does not show
  }
}

export function summarize(view: ConversationView, sandbox: SandboxState): ConversationSummary {
  const items = view.entries.flatMap((entry) => {
    const item = toItem(entry.id, entry.kind, (entry.model as Message[] | undefined)?.[0]);
    return item ? [item] : [];
  });

  // The built-in documents, keyed by kind; each is absent until first written.
  const live = view.docs['pi.live'] as LiveState | undefined;
  const inbox = view.docs['pi.inbox'] as InboxState | undefined;
  const spend = view.docs['pi.usage'] as UsageState | undefined;

  const usage = { input: 0, output: 0, cacheRead: 0, cost: 0 };
  for (const u of Object.values(spend?.models ?? {})) {
    usage.input += u.input;
    usage.output += u.output;
    usage.cacheRead += u.cacheRead;
    usage.cost += u.cost.total;
  }

  const partial = live?.generation?.message as AssistantMessage | undefined;
  const { parent } = view.conversation;
  return {
    id: view.conversation.id,
    parent: parent && { conversationId: parent.conversationId, entryId: parent.at },
    sandboxId: sandbox.sandboxId,
    snapshotId: sandbox.snapshotId,
    busy: live?.run !== undefined,
    inbox: inbox?.items.length ?? 0,
    usage,
    items,
    before: view.before,
    failure: live?.runFailure && { reason: live.runFailure.reason, detail: live.runFailure.detail },
    live: live?.run && {
      text: clip(textOf(partial?.content)),
      thinking: clip(thinkingOf(partial?.content)),
      retry: live.generation?.retry?.error,
      tools: (live.tools ?? []).map((tool) => ({
        callId: tool.callId,
        name: tool.name,
        status: tool.status,
        output: clip(tool.output ?? ''),
      })),
    },
  };
}
