import { hatchet } from '../../hatchet-client';

type WebhookPayload = { event_id: string; type: string; data: Record<string, unknown> };

// > Step 01 Define Webhook Task
const processWebhook = hatchet.task<WebhookPayload>({
  name: 'process-webhook',
  onEvents: ['webhook:stripe', 'webhook:github'],
  fn: async (input) => ({
    processed: input.event_id,
    type: input.type,
  }),
});

// > Step 02 Register Webhook
// Call from your webhook endpoint to trigger the task.
function forwardWebhook(eventKey: string, payload: WebhookPayload) {
  hatchet.event.push(eventKey, payload);
}
// forwardWebhook('webhook:stripe', { event_id: 'evt_123', type: 'payment', data: {} });

// > Step 03 Process Payload
// Validate event_id for deduplication; process idempotently.
function validateAndProcess(input: WebhookPayload) {
  if (!input.event_id) throw new Error('event_id required for deduplication');
  return { processed: input.event_id, type: input.type };
}

export { processWebhook };
