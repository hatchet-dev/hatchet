import { hatchet } from '../hatchet-client';

export type WebhookInput = {
  type: string;
  message: string;
};

export const webhookWorkflow = hatchet.workflow<WebhookInput>({
  name: 'webhook-workflow',
  onEvents: ['webhook:test'],
});

webhookWorkflow.task({
  name: 'webhook-task',
  fn: async (input: WebhookInput) => input,
});
