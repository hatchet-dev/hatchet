import { hatchet } from '../hatchet-client';
import { supportAgent, REPLY_EVENT_KEY, SupportTicketInput } from './workflow';

// > Trigger the workflow
async function main() {
  const input: SupportTicketInput = {
    ticketId: 'ticket-42',
    customerEmail: 'alice@example.com',
    subject: 'Login broken',
    body: "I can't log in since this morning.",
  };

  // Start the support agent workflow
  const ref = await supportAgent.runNoWait(input);
  const runId = await ref.getWorkflowRunId();
  console.log(`Started workflow run: ${runId}`);

  // Push a customer reply event (scoped to this ticket)
  console.log('Pushing customer reply event...');
  await hatchet.events.push(
    REPLY_EVENT_KEY,
    { message: 'I cleared my cookies and it works now. Thanks!' },
    { scope: input.ticketId }
  );

  // Wait for the workflow to complete
  const result = await ref.output;
  console.log('Workflow completed:', result);
}

if (require.main === module) {
  main()
    .catch(console.error)
    .finally(() => {
      process.exit(0);
    });
}
