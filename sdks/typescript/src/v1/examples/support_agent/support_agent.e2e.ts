import { randomUUID } from 'crypto';
import { makeE2EClient } from '../__e2e__/harness';
import { supportAgent, REPLY_EVENT_KEY, SupportTicketInput } from './workflow';

describe('support-agent-e2e', () => {
  const hatchet = makeE2EClient();

  it('resolves when customer replies before timeout', async () => {
    const ticketId = `test-${randomUUID().slice(0, 8)}`;
    const input: SupportTicketInput = {
      ticketId,
      customerEmail: 'alice@example.com',
      subject: 'Login broken',
      body: "I can't log in since this morning.",
    };

    const ref = await supportAgent.runNoWait(input);

    // The workflow uses considerEventsSince lookback, so the reply event
    // is captured even if pushed before the or-group wait becomes active.
    await hatchet.events.push(
      REPLY_EVENT_KEY,
      { message: 'I cleared my cookies and it works now.' },
      { scope: ticketId }
    );

    const result = await ref.output;

    expect((result as any).ticketId).toBe(ticketId);
    expect((result as any).status).toBe('resolved');
    expect((result as any).triageCategory).toBeTruthy();
    expect((result as any).initialReply).toBeTruthy();
  }, 120_000);

  it('escalates when no reply arrives before timeout', async () => {
    const ticketId = `test-${randomUUID().slice(0, 8)}`;
    const input: SupportTicketInput = {
      ticketId,
      customerEmail: 'bob@example.com',
      subject: 'Billing issue',
      body: 'I was charged twice for my subscription.',
    };

    const result = await supportAgent.run(input);

    expect((result as any).ticketId).toBe(ticketId);
    expect((result as any).status).toBe('escalated');
    expect((result as any).triageCategory).toBeTruthy();
    expect((result as any).initialReply).toBeTruthy();
  }, 120_000);
});
