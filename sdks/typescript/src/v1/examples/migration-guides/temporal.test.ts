import type {
  BaseWorkflowDeclaration,
  TaskWorkflowDeclaration,
  WorkflowDeclaration,
} from '@hatchet/v1';
import type { InputType, OutputType } from '@hatchet/v1/types';
import { ConcurrencyLimitStrategy } from '@hatchet/v1/task';
import type {
  ChargeOutput,
  EmailOutput,
  FulfillOutput,
  ItemInput,
  ItemOutput,
  OrderDagOutput,
  OrderInput,
  ReportInput,
  ReportOutput,
  SignupInput,
  SyncInput,
  SyncOutput,
  ValidateOutput,
} from './temporal';

// The examples call HatchetClient.init() at import time, which needs a token to resolve the
// tenant and engine addresses from its claims. Nothing here connects, so a synthetic one is enough.
process.env.HATCHET_CLIENT_TOKEN =
  'eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJncnBjX2Jyb2FkY2FzdF9hZGRyZXNzIjoiMTI3LjAuMC4xOjgwODAiLCJzZXJ2ZXJfdXJsIjoiaHR0cDovL2xvY2FsaG9zdDo4MDgwIiwic3ViIjoiNzA3ZDA4NTUtODBhYi00ZTFmLWExNTYtZjFjNDU0NmNiZjUyIn0K.abcdef';
process.env.HATCHET_CLIENT_TLS_STRATEGY = 'none';

// eslint-disable-next-line @typescript-eslint/no-require-imports
const examples = require('./temporal') as typeof import('./temporal');

const {
  approvalFlow,
  callModel,
  chargeOrder,
  chargeOrderWithRetries,
  fulfillOrder,
  onboardingFlow,
  orderFollowUp,
  orderWorkflow,
  packOrder,
  processItem,
  processOrder,
  sendFollowupEmail,
  sendWelcomeEmail,
  syncCustomer,
  validateOrder,
  weeklyReport,
} = examples;

const ORDER: OrderInput = { orderId: 'ord_1', correlationId: 'corr-123' };
const SIGNUP: SignupInput = { userId: 'usr_1', email: 'ada@example.com' };

/** The single task opts (with its `fn`) behind a standalone task or durable task declaration. */
function standaloneOpts(decl: BaseWorkflowDeclaration<any, any>) {
  const [opts] = [...decl.definition._tasks, ...decl.definition._durableTasks];

  if (!opts?.fn) {
    throw new Error(`no task function on ${decl.definition.name}`);
  }

  return opts;
}

/**
 * Typed handle on a standalone task's body. The generics come off the declaration, so a wrong
 * input or a wrong expected output is a compile error rather than a runtime surprise.
 */
function standaloneFn<I extends InputType, O extends OutputType>(
  decl: BaseWorkflowDeclaration<I, O>
): (input: I, ctx: any) => Promise<O> {
  return standaloneOpts(decl).fn as (input: I, ctx: any) => Promise<O>;
}

/** A DAG task's opts by name. Their outputs are per-task, so these stay loosely typed. */
function dagTask(decl: BaseWorkflowDeclaration<any, any>, name: string) {
  const opts = decl.definition._tasks.find((task) => task.name === name);

  if (!opts?.fn) {
    throw new Error(`no task named ${name} on ${decl.definition.name}`);
  }

  return opts;
}

function makeContext(overrides: Record<string, unknown> = {}) {
  return {
    logger: {
      info: jest.fn(),
      debug: jest.fn(),
      warn: jest.fn(),
      error: jest.fn(),
    },
    ...overrides,
  } as any;
}

/** Replaces a declaration's `run` so task bodies can be exercised without an engine. */
function mockRun(decl: BaseWorkflowDeclaration<any, any>, impl: (input: any) => Promise<any>) {
  const spy = jest.spyOn(decl, 'run') as unknown as jest.Mock;
  spy.mockImplementation(impl);
  return spy;
}

afterEach(() => {
  jest.restoreAllMocks();
});

describe('temporal migration guide: task definitions', () => {
  it('charges an order and returns a typed output', async () => {
    const output: ChargeOutput = await standaloneFn(chargeOrder)(ORDER, makeContext());

    expect(output).toEqual({ charged: true, amountCents: 2505 });
  });

  it('writes to the run log sink from the context', async () => {
    const ctx = makeContext();

    const output: ValidateOutput = await standaloneFn(validateOrder)(ORDER, ctx);

    expect(output).toEqual({ valid: true });
    expect(ctx.logger.info).toHaveBeenCalledWith(expect.stringContaining(ORDER.orderId));
  });

  it('carries retry and timeout settings on the task definition', () => {
    const opts = standaloneOpts(chargeOrderWithRetries);

    expect(opts.retries).toBe(10);
    expect(opts.backoff).toEqual({ factor: 2, maxSeconds: 10 });
    expect(opts.executionTimeout).toBe('30s');
    expect(opts.scheduleTimeout).toBe('10m');
  });

  it('declares a global static rate limit', () => {
    expect(standaloneOpts(callModel).rateLimits).toEqual([{ staticKey: 'openai', units: 1 }]);
  });
});

describe('temporal migration guide: durable tasks', () => {
  it('runs validate, charge and fulfill in order', async () => {
    const calls: string[] = [];
    const fulfilled: FulfillOutput = { fulfilled: true, shipmentId: 'shp_ord_1' };

    mockRun(validateOrder, async () => {
      calls.push('validate');
      return { valid: true };
    });
    mockRun(chargeOrder, async () => {
      calls.push('charge');
      return { charged: true, amountCents: 2505 };
    });
    mockRun(fulfillOrder, async () => {
      calls.push('fulfill');
      return fulfilled;
    });

    const output: FulfillOutput = await standaloneFn(processOrder)(ORDER, makeContext());

    expect(calls).toEqual(['validate', 'charge', 'fulfill']);
    expect(output).toEqual(fulfilled);
  });

  it('sleeps durably between the welcome and follow-up emails', async () => {
    const calls: string[] = [];

    mockRun(sendWelcomeEmail, async () => {
      calls.push('welcome');
      return { sent: true };
    });
    mockRun(sendFollowupEmail, async () => {
      calls.push('followup');
      return { sent: true };
    });

    const sleepFor = jest.fn(async (duration: string) => {
      calls.push(`sleep:${duration}`);
      return {};
    });

    const output: EmailOutput = await standaloneFn(onboardingFlow)(
      SIGNUP,
      makeContext({ sleepFor })
    );

    expect(calls).toEqual(['welcome', 'sleep:72h', 'followup']);
    expect(output).toEqual({ sent: true });
    // The execution timeout has to outlast the sleep, or the run is killed mid-wait.
    expect(standaloneOpts(onboardingFlow).executionTimeout).toBe('168h');
  });

  it('sleeps for a day on the durable context', async () => {
    const sleepFor = jest.fn(async () => ({}));

    const output: EmailOutput = await standaloneFn(orderFollowUp)(ORDER, makeContext({ sleepFor }));

    expect(sleepFor).toHaveBeenCalledWith('24h');
    expect(output).toEqual({ sent: true });
    expect(standaloneOpts(orderFollowUp).executionTimeout).toBe('48h');
  });

  it('waits for a correlated event before fulfilling', async () => {
    const fulfilled: FulfillOutput = { fulfilled: true, shipmentId: 'shp_ord_1' };
    const waitForEvent = jest.fn(async () => ({}));
    const fulfillRun = mockRun(fulfillOrder, async () => fulfilled);

    const output: FulfillOutput = await standaloneFn(approvalFlow)(
      ORDER,
      makeContext({ waitForEvent })
    );

    expect(waitForEvent).toHaveBeenCalledWith(
      'approval:granted',
      "input.correlationId == 'corr-123'"
    );
    expect(fulfillRun).toHaveBeenCalledWith(ORDER);
    expect(output).toEqual(fulfilled);
  });

  it('spawns one child run per item, concurrently', async () => {
    const events: string[] = [];

    mockRun(processItem, async ({ item }: ItemInput): Promise<ItemOutput> => {
      events.push(`start:${item}`);
      await Promise.resolve();
      events.push(`end:${item}`);
      return { packed: item !== 'torn-box' };
    });

    const output = await standaloneFn(packOrder)(
      { items: ['mug', 'torn-box', 'poster'] },
      makeContext()
    );

    expect(output).toEqual({ packed: 2 });
    // Sequential spawning would interleave start/end pairs instead of starting all three first.
    expect(events.slice(0, 3)).toEqual(['start:mug', 'start:torn-box', 'start:poster']);
  });
});

describe('temporal migration guide: DAG workflow', () => {
  it('passes each task output to the next through the context', async () => {
    const validate = dagTask(orderWorkflow, 'validate');
    const charge = dagTask(orderWorkflow, 'charge');
    const fulfill = dagTask(orderWorkflow, 'fulfill');

    const validated = await validate.fn!(ORDER, makeContext());
    expect(validated).toEqual({ valid: true });

    const charged = await charge.fn!(
      ORDER,
      makeContext({ parentOutput: jest.fn(async () => validated) })
    );
    expect(charged).toEqual({ charged: true, amountCents: 2505 });

    const fulfilled = await fulfill.fn!(
      ORDER,
      makeContext({ parentOutput: jest.fn(async () => charged) })
    );
    expect(fulfilled).toEqual({ fulfilled: true, shipmentId: 'shp_ord_1' });
  });

  it('short-circuits downstream tasks when validation fails', async () => {
    const charge = dagTask(orderWorkflow, 'charge');
    const fulfill = dagTask(orderWorkflow, 'fulfill');

    const charged = await charge.fn!(
      ORDER,
      makeContext({ parentOutput: jest.fn(async () => ({ valid: false })) })
    );
    expect(charged).toEqual({ charged: false, amountCents: 0 });

    const fulfilled = await fulfill.fn!(
      ORDER,
      makeContext({ parentOutput: jest.fn(async () => charged) })
    );
    expect(fulfilled).toEqual({ fulfilled: false, shipmentId: '' });
  });

  it('chains the three tasks with parents', () => {
    const validate = dagTask(orderWorkflow, 'validate');
    const charge = dagTask(orderWorkflow, 'charge');
    const fulfill = dagTask(orderWorkflow, 'fulfill');

    expect(validate.parents).toBeUndefined();
    expect(charge.parents).toEqual([validate]);
    expect(fulfill.parents).toEqual([charge]);
    expect([validate, charge, fulfill].map((task) => task.executionTimeout)).toEqual([
      '30s',
      '30s',
      '30s',
    ]);
  });
});

describe('temporal migration guide: triggers and flow control', () => {
  it('declares a cron on the workflow', () => {
    expect(weeklyReport.definition.on).toEqual({ cron: '0 9 * * 1' });
    expect(dagTask(weeklyReport, 'generate')).toBeDefined();
  });

  it('runs the report task', async () => {
    const output = await dagTask(weeklyReport, 'generate').fn!({ kind: 'weekly' }, makeContext());

    expect(output).toEqual({ rows: 6 });
  });

  it('limits a workflow to one in-flight run per customer', () => {
    expect(syncCustomer.definition.concurrency).toEqual({
      expression: 'input.customerId',
      maxRuns: 1,
      limitStrategy: ConcurrencyLimitStrategy.CANCEL_IN_PROGRESS,
    });
  });
});

describe('temporal migration guide: declaration types', () => {
  it('propagates input and output types onto every declaration', () => {
    const typedProcessOrder: TaskWorkflowDeclaration<OrderInput, FulfillOutput> = processOrder;
    const typedChargeOrder: TaskWorkflowDeclaration<OrderInput, ChargeOutput> = chargeOrder;
    const typedOnboarding: TaskWorkflowDeclaration<SignupInput, EmailOutput> = onboardingFlow;
    const typedApproval: TaskWorkflowDeclaration<OrderInput, FulfillOutput> = approvalFlow;
    const typedProcessItem: TaskWorkflowDeclaration<ItemInput, ItemOutput> = processItem;
    const typedOrderWorkflow: WorkflowDeclaration<OrderInput, OrderDagOutput> = orderWorkflow;
    const typedWeeklyReport: WorkflowDeclaration<ReportInput, ReportOutput> = weeklyReport;
    const typedSyncCustomer: WorkflowDeclaration<SyncInput, SyncOutput> = syncCustomer;

    expect(
      [
        typedProcessOrder,
        typedChargeOrder,
        typedOnboarding,
        typedApproval,
        typedProcessItem,
        typedOrderWorkflow,
        typedWeeklyReport,
        typedSyncCustomer,
      ].map((decl) => decl.definition.name)
    ).toEqual([
      'ProcessOrder',
      'charge-order',
      'OnboardingFlow',
      'ApprovalFlow',
      'process-item',
      'ProcessOrderDag',
      'weekly-report',
      'SyncCustomer',
    ]);
  });
});
