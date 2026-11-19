import { describe, expect, it } from 'vitest';
import { ClientMessage, InboxMessage, SessionId, sessionTopics } from '../src/protocol.js';

describe('SessionId', () => {
  it('accepts short URL-safe ids', () => {
    expect(SessionId.safeParse('abc_DEF-123').success).toBe(true);
  });

  it.each(['', 'a b', 'a/b', 'é', 'x'.repeat(65)])('refuses %j', (id) => {
    expect(SessionId.safeParse(id).success).toBe(false);
  });
});

describe('sessionTopics', () => {
  it('names both topics after the task and the session', () => {
    expect(sessionTopics('pi-session', 'abc')).toEqual({
      inbox: 'pi-session-inbox-abc',
      view: 'pi-session-view-abc',
    });
  });

  it('prefixes both topics with the namespace', () => {
    expect(sessionTopics('pi-session', 'abc', 'team_')).toEqual({
      inbox: 'team_pi-session-inbox-abc',
      view: 'team_pi-session-view-abc',
    });
  });
});

describe('ClientMessage', () => {
  it('queues input behind a busy conversation by default', () => {
    expect(ClientMessage.parse({ type: 'input', conversationId: 0, text: '  hi  ' })).toEqual({
      type: 'input',
      conversationId: 0,
      text: 'hi',
      whenBusy: 'followUp',
    });
  });

  it('refuses blank input', () => {
    expect(ClientMessage.safeParse({ type: 'input', conversationId: 0, text: '   ' }).success).toBe(
      false
    );
  });

  it('accepts forks and aborts', () => {
    expect(ClientMessage.safeParse({ type: 'fork', conversationId: 0, entryId: 4 }).success).toBe(
      true
    );
    expect(ClientMessage.safeParse({ type: 'abort', conversationId: 2 }).success).toBe(true);
  });

  it('refuses unknown types and negative ids', () => {
    expect(ClientMessage.safeParse({ type: 'delete', conversationId: 0 }).success).toBe(false);
    expect(ClientMessage.safeParse({ type: 'abort', conversationId: -1 }).success).toBe(false);
  });
});

describe('InboxMessage', () => {
  it('needs a request id', () => {
    const message = { type: 'abort', conversationId: 0 };
    expect(InboxMessage.safeParse(message).success).toBe(false);
    expect(InboxMessage.safeParse({ ...message, requestId: crypto.randomUUID() }).success).toBe(
      true
    );
  });
});
