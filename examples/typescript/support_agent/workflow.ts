import { Or, SleepCondition, UserEventCondition } from '@hatchet-dev/typescript-sdk/v1/conditions';
import { durationToMs } from '@hatchet-dev/typescript-sdk/v1/client/duration';
import { hatchet } from '../hatchet-client';

export const REPLY_EVENT_KEY = 'support:customer-reply';
const REPLY_LABEL = 'reply' as const;
const TIMEOUT_LABEL = 'timeout' as const;
export const TIMEOUT_SECONDS = 5;
const LOOKBACK_WINDOW = '5m' as const;

// > Models
export type SupportTicketInput = {
  ticketId: string;
  customerEmail: string;
  subject: string;
  body: string;
};

// > Triage task
// Classify the ticket into a category and priority.
export const triageTicket = hatchet.task({
  name: 'triage-ticket',
  fn: async (input: SupportTicketInput) => {
    const text = `${input.subject} ${input.body}`.toLowerCase();

    let category: string;
    if (['bill', 'charge', 'payment', 'invoice'].some((w) => text.includes(w))) {
      category = 'billing';
    } else if (['login', 'password', 'auth', 'access'].some((w) => text.includes(w))) {
      category = 'account';
    } else {
      category = 'technical';
    }

    let priority: string;
    if (['urgent', 'critical', 'down', 'outage'].some((w) => text.includes(w))) {
      priority = 'high';
    } else if (['twice', 'broken', 'error'].some((w) => text.includes(w))) {
      priority = 'medium';
    } else {
      priority = 'low';
    }

    return { category, priority };
  },
});

// > Generate reply task
// Generate an initial support reply (fallback-only in TypeScript v1).
export const generateReply = hatchet.task({
  name: 'generate-reply',
  fn: async (input: SupportTicketInput) => {
    return {
      message: `Thank you for contacting support about: ${input.subject}. We are looking into this and will get back to you shortly.`,
    };
  },
});

// > Escalate task
// Escalate an unresolved ticket to the human support team.
export const escalateTicket = hatchet.task({
  name: 'escalate-ticket',
  fn: async (input: SupportTicketInput) => {
    return {
      reason: `No customer reply within ${TIMEOUT_SECONDS}s timeout`,
      assignedTo: 'support-team@example.com',
    };
  },
});

// > Support agent workflow
export const supportAgent = hatchet.durableTask({
  name: 'support-agent',
  executionTimeout: '10m',
  fn: async (input: SupportTicketInput, ctx) => {
    // Step 1: Triage the ticket
    const triage = await triageTicket.run(input);

    // Step 2: Generate an initial reply
    const reply = await generateReply.run(input);

    // Step 3: Wait for a customer reply or timeout
    const now = await ctx.now();
    const considerEventsSince = new Date(
      now.getTime() - durationToMs(LOOKBACK_WINDOW)
    ).toISOString();

    const waitResult = await ctx.waitFor(
      Or(
        new SleepCondition(`${TIMEOUT_SECONDS}s`, TIMEOUT_LABEL),
        new UserEventCondition(
          REPLY_EVENT_KEY,
          '',
          REPLY_LABEL,
          undefined,
          input.ticketId,
          considerEventsSince
        )
      )
    );

    // Determine which condition fired. ctx.waitFor returns
    // { CREATE: { <label>: ... } } where <label> is the readableDataKey
    // we assigned above ('timeout' or 'reply').
    const create = (waitResult as Record<string, Record<string, unknown>>)['CREATE'] ?? waitResult;
    const resolvedLabel = Object.keys(create as Record<string, unknown>)[0] ?? '';
    const customerReplied = resolvedLabel === REPLY_LABEL;

    if (!customerReplied) {
      // Step 4a: Timeout -> escalate
      await escalateTicket.run(input);
      return {
        ticketId: input.ticketId,
        status: 'escalated' as const,
        triageCategory: triage.category,
        triagePriority: triage.priority,
        initialReply: reply.message,
      };
    }

    // Step 4b: Customer replied -> resolve
    return {
      ticketId: input.ticketId,
      status: 'resolved' as const,
      triageCategory: triage.category,
      triagePriority: triage.priority,
      initialReply: reply.message,
    };
  },
});
