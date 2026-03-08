import { randomUUID } from 'crypto';
import {
  V1WebhookSourceName,
  V1WebhookAPIKeyAuth,
} from '../../../clients/rest/generated/data-contracts';
import { makeE2EClient, poll } from '../__e2e__/harness';
import { webhookWorkflow } from './workflow';

const TEST_API_KEY_HEADER = 'X-API-Key';
const TEST_API_KEY_VALUE = 'test_api_key_123';

describe('webhooks-e2e', () => {
  const hatchet = makeE2EClient();
  let webhookName: string;

  beforeAll(async () => {
    webhookName = `test-webhook-e2e-${randomUUID()}`;
    await hatchet.webhooks.create({
      sourceName: V1WebhookSourceName.GENERIC,
      name: webhookName,
      eventKeyExpression: "'webhook:' + input.type",
      auth: {
        headerName: TEST_API_KEY_HEADER,
        apiKey: TEST_API_KEY_VALUE,
      } as V1WebhookAPIKeyAuth,
    });
  });

  afterAll(async () => {
    try {
      await hatchet.webhooks.delete(webhookName);
    } catch {
      // ignore cleanup errors
    }
  });

  xit('webhook receive triggers workflow run', async () => {
    const testStart = new Date();
    const payload = { type: 'test', message: 'Hello, world!' };

    // await hatchet.webhooks.receive(webhookName, payload);

    const runsResp = await poll(
      async () => {
        const r = await hatchet.runs.list({
          since: testStart,
          workflowNames: [webhookWorkflow.definition.name],
          onlyTasks: false,
        });
        return r;
      },
      {
        timeoutMs: 30_000,
        intervalMs: 100,
        label: 'webhook-triggered runs',
        shouldStop: (r) => (r.rows?.length ?? 0) > 0,
      }
    );

    expect(runsResp.rows?.length).toBeGreaterThan(0);
    const [run] = runsResp.rows!;
    expect(run.status).toBe('COMPLETED');
    expect(run.additionalMetadata).toBeDefined();
    expect((run.additionalMetadata as Record<string, string>).hatchet__event_key).toBe(
      'webhook:test'
    );
  }, 60_000);
});
