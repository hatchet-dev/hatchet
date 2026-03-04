import { hatchet } from '../../hatchet-client';

// > Step 03 Push Event
// Push an event to trigger the workflow. Use the same key as onEvents.
hatchet.event.push('order:created', {
  message: 'Order #1234',
  source: 'webhook',
});
