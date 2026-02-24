import Hatchet from '../sdk';
import { V1WebhookSourceName } from '../clients/rest/generated/data-contracts';

const hatchet = Hatchet.init();

async function main() {
  const webhookName = `example-webhook-${Date.now()}`;
  const namespace = hatchet.config?.namespace ?? 'default';

  const webhook = await hatchet.webhooks.create({
    sourceName: V1WebhookSourceName.GENERIC,
    name: webhookName,
    eventKeyExpression: `'${namespace}/webhook:' + input.type`,
    scopeExpression: 'input.customer_id',
    staticPayload: { customer_id: 'cust-123', environment: 'production' },
    auth: { username: 'test_user', password: 'test_password' },
  });
  console.log('Created webhook:', webhook.name, webhook.scopeExpression, webhook.staticPayload);

  const one = await hatchet.webhooks.get(webhook.name);
  console.log('Get webhook:', one.name);

  await hatchet.webhooks.update(webhook.name, {
    scopeExpression: 'input.environment',
  });
  const updated = await hatchet.webhooks.get(webhook.name);
  console.log('Updated scope expression:', updated.scopeExpression);

  const list = await hatchet.webhooks.list({ limit: 10, offset: 0 });
  console.log('List webhooks:', list.rows?.length ?? 0);

  await hatchet.webhooks.delete(webhook.name);
  console.log('Deleted example webhook');
}

main();
