// > Hatchet worker
import { HatchetClient } from '@hatchet/v1';

export const hatchet = HatchetClient.init();

async function main() {
  const worker = await hatchet.worker('order-worker', {
    workflows: [validateOrder, chargeOrder, fulfillOrder, processOrder],
    slots: 10,
  });

  await worker.start();
}
// !!

export type OrderInput = {
  orderId: string;
  // A CEL-safe id you generate, used to correlate events back to this run.
  correlationId: string;
};

export type ValidateOutput = {
  valid: boolean;
};

export type FulfillOutput = {
  fulfilled: boolean;
  shipmentId: string;
};

export type SignupInput = {
  userId: string;
  email: string;
};

export type EmailOutput = {
  sent: boolean;
};

const payments = {
  charge: async (orderId: string): Promise<number> => 2500 + orderId.length,
};

const warehouse = {
  reserve: async (orderId: string): Promise<boolean> => orderId.length > 0,
  ship: async (orderId: string): Promise<string> => `shp_${orderId}`,
  pack: async (item: string): Promise<boolean> => item.length > 0,
};

const emails = {
  send: async (address: string, template: string): Promise<boolean> =>
    address.includes('@') && template.length > 0,
};

const models = {
  complete: async (prompt: string): Promise<string> => prompt.toUpperCase(),
};

const crm = {
  sync: async (customerId: string): Promise<number> => customerId.length,
};

const reports = {
  build: async (kind: string): Promise<number> => kind.length,
};

export const validateOrder = hatchet.task({
  name: 'validate-order',
  fn: async (input: OrderInput, ctx): Promise<ValidateOutput> => {
    // > Hatchet context logging
    ctx.logger.info(`validating order ${input.orderId}`);
    // !!

    return { valid: await warehouse.reserve(input.orderId) };
  },
});

// > Hatchet task definition
export type ChargeOutput = {
  charged: boolean;
  amountCents: number;
};

export const chargeOrder = hatchet.task({
  name: 'charge-order',
  fn: async (input: OrderInput): Promise<ChargeOutput> => {
    const amountCents = await payments.charge(input.orderId);

    return { charged: amountCents > 0, amountCents };
  },
});
// !!

export const fulfillOrder = hatchet.task({
  name: 'fulfill-order',
  fn: async (input: OrderInput): Promise<FulfillOutput> => {
    const shipmentId = await warehouse.ship(input.orderId);

    return { fulfilled: true, shipmentId };
  },
});

// > Hatchet workflow as durable task
export const processOrder = hatchet.durableTask({
  name: 'ProcessOrder',
  executionTimeout: '10m',
  fn: async (input: OrderInput): Promise<FulfillOutput> => {
    await validateOrder.run(input);
    await chargeOrder.run(input);

    return fulfillOrder.run(input);
  },
});
// !!

export const sendWelcomeEmail = hatchet.task({
  name: 'send-welcome-email',
  fn: async (input: SignupInput): Promise<EmailOutput> => ({
    sent: await emails.send(input.email, 'welcome'),
  }),
});

export const sendFollowupEmail = hatchet.task({
  name: 'send-followup-email',
  fn: async (input: SignupInput): Promise<EmailOutput> => ({
    sent: await emails.send(input.email, 'followup'),
  }),
});

// > Hatchet durable task with sleep
export const onboardingFlow = hatchet.durableTask({
  name: 'OnboardingFlow',
  // The timeout has to cover the whole wall-clock span of the run, sleeps included.
  executionTimeout: '168h',
  fn: async (input: SignupInput, ctx): Promise<EmailOutput> => {
    await sendWelcomeEmail.run(input);

    await ctx.sleepFor('72h');

    return sendFollowupEmail.run(input);
  },
});
// !!

async function invoke(input: OrderInput) {
  // > Hatchet task invocation
  const run = await processOrder.runNoWait(input);

  // It may be helpful to store this run id somewhere durable.
  const runId = await run.getWorkflowRunId();

  const result = await run.output;
  // !!

  return { runId, result };
}

// > Hatchet retries and timeouts
export const chargeOrderWithRetries = hatchet.task({
  name: 'charge-order-with-retries',
  retries: 10,
  backoff: {
    // Factor to increase the wait time between retries.
    factor: 2,
    // Maximum number of seconds to wait between retries.
    maxSeconds: 10,
  },
  executionTimeout: '30s',
  scheduleTimeout: '10m',
  fn: async (input: OrderInput): Promise<ChargeOutput> => {
    const amountCents = await payments.charge(input.orderId);

    return { charged: amountCents > 0, amountCents };
  },
});
// !!

export const orderFollowUp = hatchet.durableTask({
  name: 'OrderFollowUp',
  executionTimeout: '48h',
  fn: async (input: OrderInput, ctx): Promise<EmailOutput> => {
    // > Hatchet durable sleep
    await ctx.sleepFor('24h');
    // !!

    return { sent: await emails.send(`${input.orderId}@example.com`, 'order-followup') };
  },
});

async function grantApproval(input: OrderInput) {
  // > Hatchet event push
  await hatchet.events.push('approval:granted', {
    correlationId: input.correlationId,
  });
  // !!
}

// > Hatchet durable event wait
export const approvalFlow = hatchet.durableTask({
  name: 'ApprovalFlow',
  executionTimeout: '10m',
  fn: async (input: OrderInput, ctx): Promise<FulfillOutput> => {
    // The expression is compiled as CEL, so correlate on an id that cannot contain a quote.
    await ctx.waitForEvent('approval:granted', `input.correlationId == '${input.correlationId}'`);

    return fulfillOrder.run(input);
  },
});
// !!

export type ItemInput = {
  item: string;
};

export type ItemOutput = {
  packed: boolean;
};

export const processItem = hatchet.task({
  name: 'process-item',
  fn: async (input: ItemInput): Promise<ItemOutput> => ({
    packed: await warehouse.pack(input.item),
  }),
});

// > Hatchet fan out children
export const packOrder = hatchet.durableTask({
  name: 'PackOrder',
  executionTimeout: '30m',
  fn: async (input: { items: string[] }): Promise<{ packed: number }> => {
    const results = await Promise.all(input.items.map((item) => processItem.run({ item })));

    return { packed: results.filter((result) => result.packed).length };
  },
});
// !!

export type ReportInput = {
  kind: string;
};

export type ReportOutput = {
  generate: { rows: number };
};

// > Hatchet cron declaration
export const weeklyReport = hatchet.workflow<ReportInput, ReportOutput>({
  name: 'weekly-report',
  on: {
    cron: '0 9 * * 1',
  },
});

weeklyReport.task({
  name: 'generate',
  fn: async (input) => {
    return { rows: await reports.build(input.kind) };
  },
});
// !!

async function schedule() {
  // > Hatchet runtime schedules
  // A recurring schedule, created at runtime.
  await hatchet.crons.create('weekly-report', {
    name: 'weekly-report-acme',
    expression: '0 9 * * 1',
    input: { kind: 'weekly' },
  });

  // A one-shot future run.
  await hatchet.schedules.create('weekly-report', {
    triggerAt: new Date(Date.now() + 24 * 60 * 60 * 1000),
    input: { kind: 'weekly' },
  });
  // !!
}

export type SyncInput = {
  customerId: string;
};

export type SyncOutput = {
  sync: { synced: number };
};

export type PromptInput = {
  prompt: string;
};

// > Hatchet concurrency and rate limits
import { ConcurrencyLimitStrategy } from '@hatchet/v1';

// One in-flight run per customer, newest cancels the oldest.
export const syncCustomer = hatchet.workflow<SyncInput, SyncOutput>({
  name: 'SyncCustomer',
  concurrency: {
    expression: 'input.customerId',
    maxRuns: 1,
    limitStrategy: ConcurrencyLimitStrategy.CANCEL_IN_PROGRESS,
  },
});

syncCustomer.task({
  name: 'sync',
  fn: async (input) => {
    return { synced: await crm.sync(input.customerId) };
  },
});

// A global budget shared by every worker, not per-process.
export const callModel = hatchet.task({
  name: 'call-model',
  rateLimits: [
    {
      staticKey: 'openai',
      units: 1,
    },
  ],
  fn: async (input: PromptInput) => {
    return { completion: await models.complete(input.prompt) };
  },
});
// !!

export type OrderDagOutput = {
  validate: ValidateOutput;
  charge: ChargeOutput;
  fulfill: FulfillOutput;
};

// > Hatchet DAG workflow
export const orderWorkflow = hatchet.workflow<OrderInput, OrderDagOutput>({
  name: 'ProcessOrderDag',
});

const validate = orderWorkflow.task({
  name: 'validate',
  executionTimeout: '30s',
  fn: async (input) => {
    return { valid: await warehouse.reserve(input.orderId) };
  },
});

const charge = orderWorkflow.task({
  name: 'charge',
  parents: [validate],
  executionTimeout: '30s',
  fn: async (input, ctx) => {
    const validated = await ctx.parentOutput(validate);

    if (!validated.valid) {
      return { charged: false, amountCents: 0 };
    }

    const amountCents = await payments.charge(input.orderId);

    return { charged: true, amountCents };
  },
});

orderWorkflow.task({
  name: 'fulfill',
  parents: [charge],
  executionTimeout: '30s',
  fn: async (input, ctx) => {
    const charged = await ctx.parentOutput(charge);

    if (!charged.charged) {
      return { fulfilled: false, shipmentId: '' };
    }

    return { fulfilled: true, shipmentId: await warehouse.ship(input.orderId) };
  },
});
// !!
