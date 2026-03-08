import { hatchet } from '../../hatchet-client';

type EventInput = { message: string; source?: string };

// > Step 01 Define Event Task
const eventWf = hatchet.workflow<EventInput>({
  name: 'EventDrivenWorkflow',
  onEvents: ['order:created', 'user:signup'],
});

eventWf.task({
  name: 'process-event',
  fn: async (input) => ({
    processed: input.message,
    source: input.source ?? 'api',
  }),
});

// > Step 02 Register Event Trigger
// Push an event from your app to trigger the workflow. Use the same key as onEvents.
hatchet.event.push('order:created', { message: 'Order #1234', source: 'webhook' });

export { eventWf };
