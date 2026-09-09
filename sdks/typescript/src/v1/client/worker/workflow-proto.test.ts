import { z } from 'zod/v4';
import { CreateWorkflowVersionRequest, IdempotencyMethod } from '@hatchet/protoc/v1/workflows';
import { StickyStrategy as PbStickyStrategy } from '@hatchet/protoc/workflows';
import { ConcurrencyLimitStrategy } from '@hatchet/v1/task';
import { RateLimitDuration } from '@hatchet/protoc/v1/workflows';
import { Action as ConditionAction } from '@hatchet/protoc/v1/shared/condition';
import { CreateDurableTaskWorkflow, CreateTaskWorkflow, CreateWorkflow } from '../../declaration';
import { InternalWorker } from './worker-internal';
import {
  ON_FAILURE_TASK_NAME,
  ON_SUCCESS_TASK_NAME,
  normalizeWorkflowDefinition,
  workflowToProto,
} from './workflow-proto';

const NAMESPACE = 'Acme_';

// Condition groups get a fresh random id on every conversion.
function withoutGroupIds<T>(value: T): T {
  return JSON.parse(JSON.stringify(value, (key, v) => (key === 'orGroupId' ? '<group>' : v)));
}

function fixtureWorkflow() {
  const workflow = CreateWorkflow<{ id: string; tier: string }>({
    name: 'Order-Pipeline',
    description: 'processes orders',
    version: 'v2',
    on: { event: ['order:created', 'order:Updated'], cron: '*/5 * * * *' },
    onCrons: ['0 0 * * *'],
    onEvents: ['nightly'],
    sticky: 'hard',
    defaultPriority: 2,
    inputValidator: z.object({ id: z.string(), tier: z.string() }),
    concurrency: [
      {
        expression: 'input.tier',
        maxRuns: 3,
        limitStrategy: ConcurrencyLimitStrategy.GROUP_ROUND_ROBIN,
      },
      {
        expression: 'input.id',
        maxRuns: "input.tier == 'gold' ? 10 : 1",
        name: 'shared',
        isTenantScoped: true,
      },
    ],
    taskDefaults: {
      executionTimeout: '2m',
      scheduleTimeout: '10m',
      retries: 2,
      backoff: { factor: 2, maxSeconds: 30 },
      rateLimits: [{ staticKey: 'default-limit', units: 1, duration: RateLimitDuration.MINUTE }],
      workerLabels: { region: { value: 'eu' } },
    },
    defaultFilters: [{ scope: 'tenant', expression: 'true', payload: { a: 1 } }],
    idempotency: { strategy: 'status', expression: 'input.id', fallbackTtlMs: 5000 },
  });

  const validate = workflow.task({
    name: 'Validate',
    fn: () => ({ ok: true }),
    executionTimeout: '30s',
    retries: 5,
    rateLimits: [{ dynamicKey: 'input.tier', units: 'input.units', limit: 'input.limit' }],
    desiredWorkerLabels: {
      region: { value: 'us', required: true, weight: 10 },
      shard: { value: 4 },
    },
    concurrency: { expression: 'input.id', maxRuns: 1 },
    slotCost: 2,
  });

  const charge = workflow.task({
    name: 'Charge',
    parents: [validate],
    fn: () => ({ charged: true }),
    waitFor: [{ eventKey: 'payment:Confirmed' }, { sleepFor: '5s' }],
    skipIf: { eventKey: 'skip:charge' },
    cancelIf: [{ parent: validate, expression: 'output.ok == false' }],
  });

  workflow.durableTask({
    name: 'Settle',
    parents: [charge],
    fn: async () => ({ settled: true }),
    executionTimeout: '1h',
  });

  workflow.onFailure({
    fn: () => ({ failed: true }),
    retries: 1,
    executionTimeout: '45s',
    rateLimits: [{ staticKey: 'failure', units: 1 }],
    desiredWorkerLabels: { region: { value: 'eu' } },
  });

  workflow.onSuccess({
    fn: () => ({ done: true }),
    executionTimeout: '20s',
    retries: 3,
  });

  return workflow;
}

describe('workflowToProto', () => {
  it('builds the registration request for a workflow with parents, conditions and triggers', () => {
    const request = workflowToProto(fixtureWorkflow(), { namespace: NAMESPACE });

    expect(request.name).toBe('acme_order-pipeline');
    expect(request.description).toBe('processes orders');
    expect(request.version).toBe('v2');
    expect(request.eventTriggers).toEqual([
      'Acme_nightly',
      'Acme_order:created',
      'Acme_order:Updated',
    ]);
    expect(request.cronTriggers).toEqual(['0 0 * * *', '*/5 * * * *']);
    expect(request.sticky).toBe(PbStickyStrategy.HARD);
    expect(request.defaultPriority).toBe(2);
    expect(request.concurrency).toBeUndefined();
    expect(request.concurrencyArr).toEqual([
      {
        expression: 'input.tier',
        maxRuns: 3,
        limitStrategy: ConcurrencyLimitStrategy.GROUP_ROUND_ROBIN,
        name: undefined,
        isTenantScoped: undefined,
        maxRunsExpression: undefined,
      },
      {
        expression: 'input.id',
        maxRuns: 1,
        limitStrategy: undefined,
        name: 'shared',
        isTenantScoped: true,
        maxRunsExpression: "input.tier == 'gold' ? 10 : 1",
      },
    ]);
    expect(request.defaultFilters).toEqual([
      {
        scope: 'tenant',
        expression: 'true',
        payload: new TextEncoder().encode(JSON.stringify({ a: 1 })),
      },
    ]);
    expect(request.idempotency).toEqual({
      expression: 'input.id',
      ttlMs: 5000,
      method: IdempotencyMethod.STATUS,
    });

    const jsonSchema = JSON.parse(new TextDecoder().decode(request.inputJsonSchema));
    expect(jsonSchema.type).toBe('object');
    expect(Object.keys(jsonSchema.properties)).toEqual(['id', 'tier']);

    const tasks = Object.fromEntries(request.tasks.map((t) => [t.readableId, t]));
    expect(Object.keys(tasks)).toEqual(['Validate', 'Charge', ON_SUCCESS_TASK_NAME, 'Settle']);

    const validate = tasks['Validate'];
    expect(validate.action).toBe('acme_order-pipeline:validate');
    expect(validate.timeout).toBe('30s');
    expect(validate.scheduleTimeout).toBe('10m');
    expect(validate.retries).toBe(5);
    expect(validate.parents).toEqual([]);
    expect(validate.isDurable).toBe(false);
    expect(validate.slotRequests).toEqual({ default: 2 });
    expect(validate.backoffFactor).toBe(2);
    expect(validate.backoffMaxSeconds).toBe(30);
    expect(validate.rateLimits).toEqual([
      {
        key: 'input.tier',
        keyExpr: 'input.tier',
        units: undefined,
        unitsExpr: 'input.units',
        limitValuesExpr: 'input.limit',
        duration: undefined,
      },
    ]);
    expect(validate.workerLabels).toEqual({
      region: {
        strValue: 'us',
        intValue: undefined,
        required: true,
        weight: 10,
        comparator: undefined,
      },
      shard: {
        strValue: undefined,
        intValue: 4,
        required: undefined,
        weight: undefined,
        comparator: undefined,
      },
    });
    expect(validate.concurrency).toEqual([
      {
        expression: 'input.id',
        maxRuns: 1,
        limitStrategy: undefined,
        name: undefined,
        isTenantScoped: undefined,
        maxRunsExpression: undefined,
      },
    ]);
    expect(validate.batch).toBeUndefined();

    const charge = tasks['Charge'];
    expect(charge.action).toBe('acme_order-pipeline:charge');
    expect(charge.parents).toEqual(['Validate']);
    expect(charge.timeout).toBe('2m');
    expect(charge.retries).toBe(2);
    expect(charge.rateLimits).toEqual([
      {
        key: 'default-limit',
        keyExpr: undefined,
        units: 1,
        unitsExpr: undefined,
        limitValuesExpr: '-1',
        duration: RateLimitDuration.MINUTE,
      },
    ]);
    expect(charge.workerLabels).toEqual({
      region: {
        strValue: 'eu',
        intValue: undefined,
        required: undefined,
        weight: undefined,
        comparator: undefined,
      },
    });
    expect(
      charge.conditions?.userEventConditions.map((c) => [c.userEventKey, c.base?.action])
    ).toEqual([
      ['Acme_payment:Confirmed', ConditionAction.QUEUE],
      ['Acme_skip:charge', ConditionAction.SKIP],
    ]);
    expect(charge.conditions?.sleepConditions.map((c) => [c.sleepFor, c.base?.action])).toEqual([
      ['5s', ConditionAction.QUEUE],
    ]);
    expect(
      charge.conditions?.parentOverrideConditions.map((c) => [c.parentReadableId, c.base?.action])
    ).toEqual([['Validate', ConditionAction.CANCEL]]);

    const settle = tasks['Settle'];
    expect(settle.action).toBe('acme_order-pipeline:settle');
    expect(settle.isDurable).toBe(true);
    expect(settle.slotRequests).toEqual({ durable: 1 });
    expect(settle.parents).toEqual(['Charge']);
    expect(settle.timeout).toBe('1h');

    const onSuccess = tasks[ON_SUCCESS_TASK_NAME];
    expect(onSuccess.action).toBe('acme_order-pipeline:on-success-task');
    expect(onSuccess.parents).toEqual(['Settle']);
    expect(onSuccess.timeout).toBe('20s');
    expect(onSuccess.retries).toBe(3);

    expect(request.onFailureTask).toMatchObject({
      readableId: ON_FAILURE_TASK_NAME,
      action: 'acme_order-pipeline:on-failure-task',
      timeout: '45s',
      scheduleTimeout: '10m',
      retries: 1,
      isDurable: false,
      slotRequests: { default: 1 },
      workerLabels: {
        region: {
          strValue: 'eu',
          intValue: undefined,
          required: undefined,
          weight: undefined,
          comparator: undefined,
        },
      },
    });
    expect(request.onFailureTask?.rateLimits).toEqual([
      {
        key: 'failure',
        keyExpr: undefined,
        units: 1,
        unitsExpr: undefined,
        limitValuesExpr: '-1',
        duration: undefined,
      },
    ]);
  });

  it('builds a standalone task with defaults and no namespace', () => {
    const task = CreateTaskWorkflow({
      name: 'Solo',
      fn: (input: { n: number }) => ({ n: input.n }),
    });

    const request = workflowToProto(task, {});

    expect(request).toMatchObject({
      name: 'solo',
      description: '',
      version: '',
      eventTriggers: [],
      cronTriggers: [],
      sticky: undefined,
      concurrencyArr: [],
      onFailureTask: undefined,
      defaultPriority: undefined,
      inputJsonSchema: undefined,
      concurrency: undefined,
      defaultFilters: [],
      idempotency: undefined,
    });
    expect(request.tasks).toHaveLength(1);
    expect(request.tasks[0]).toMatchObject({
      readableId: 'Solo',
      action: 'solo:solo',
      timeout: '60s',
      scheduleTimeout: undefined,
      inputs: '{}',
      userData: '{}',
      parents: [],
      retries: 0,
      rateLimits: [],
      workerLabels: {},
      isDurable: false,
      slotRequests: { default: 1 },
      concurrency: [],
    });
  });

  it('builds a standalone durable task and a function-valued on-failure handler', () => {
    const task = CreateDurableTaskWorkflow({
      name: 'Durable',
      fn: async (input: { n: number }) => ({ n: input.n }),
    });
    task.definition.onFailure = () => ({ handled: true });

    const request = workflowToProto(task, { namespace: 'ns_' });

    expect(request.tasks).toHaveLength(1);
    expect(request.tasks[0]).toMatchObject({
      action: 'ns_durable:durable',
      isDurable: true,
      slotRequests: { durable: 1 },
    });
    expect(request.onFailureTask).toMatchObject({
      readableId: ON_FAILURE_TASK_NAME,
      action: 'ns_durable:on-failure-task',
      timeout: '60s',
      retries: 0,
      slotRequests: { default: 1 },
    });
  });

  it('maps a single concurrency object onto the deprecated field and the array', () => {
    const task = CreateTaskWorkflow({
      name: 'single',
      fn: () => ({}),
      concurrency: { expression: 'input.key', maxRuns: 2 },
    });
    const workflow = CreateWorkflow({
      name: 'solo-concurrency',
      concurrency: { expression: 'input.key', maxRuns: 4 },
    });
    workflow.task(task);

    const request = workflowToProto(workflow, {});

    expect(request.concurrency).toMatchObject({ expression: 'input.key', maxRuns: 4 });
    expect(request.concurrencyArr).toEqual([]);
    expect(request.tasks[0].concurrency).toMatchObject([{ expression: 'input.key', maxRuns: 2 }]);
  });

  it('serialises to protojson with lowerCamelCase keys and base64 bytes', () => {
    const request = workflowToProto(fixtureWorkflow(), { namespace: NAMESPACE });
    const json = CreateWorkflowVersionRequest.toJSON(request) as Record<string, unknown>;

    expect(Object.keys(json)).toEqual(
      expect.arrayContaining([
        'name',
        'eventTriggers',
        'cronTriggers',
        'tasks',
        'onFailureTask',
        'concurrencyArr',
        'defaultFilters',
        'inputJsonSchema',
        'idempotency',
      ])
    );
    expect(Object.keys(json).some((k) => k.includes('_'))).toBe(false);
    expect(typeof json.inputJsonSchema).toBe('string');
    expect(Buffer.from(json.inputJsonSchema as string, 'base64').toString('utf8')).toBe(
      new TextDecoder().decode(request.inputJsonSchema)
    );
    expect(json.sticky).toBe('HARD');
    expect(
      CreateWorkflowVersionRequest.toJSON(CreateWorkflowVersionRequest.fromJSON(json))
    ).toEqual(json);
  });

  it('does not mutate the declaration and is idempotent over a normalized definition', () => {
    const workflow = fixtureWorkflow();
    const before = workflow.definition._tasks.length;

    const normalized = normalizeWorkflowDefinition(workflow, { namespace: NAMESPACE });
    const fromDeclaration = workflowToProto(workflow, { namespace: NAMESPACE });
    const fromNormalized = workflowToProto(normalized, { namespace: NAMESPACE });

    expect(workflow.definition._tasks).toHaveLength(before);
    expect(workflow.definition.name).toBe('Order-Pipeline');
    expect(normalized._tasks.map((t) => t.name)).toEqual([
      'Validate',
      'Charge',
      ON_SUCCESS_TASK_NAME,
    ]);
    expect(withoutGroupIds(fromNormalized)).toEqual(withoutGroupIds(fromDeclaration));
  });

  it('leaves the on-success task out of a durable registration', () => {
    const request = workflowToProto(fixtureWorkflow(), { namespace: NAMESPACE, durable: true });
    expect(request.tasks.map((t) => t.readableId)).toEqual(['Validate', 'Charge', 'Settle']);
  });

  it('rejects an invalid sticky strategy', () => {
    const workflow = CreateWorkflow({ name: 'bad', sticky: 'sideways' as any });
    expect(() => workflowToProto(workflow, {})).toThrow('Invalid sticky strategy: sideways');
  });
});

describe('action id casing', () => {
  function fakeWorker(namespace?: string) {
    const putWorkflow = jest.fn(async (request: CreateWorkflowVersionRequest) => request);
    const logger = { info: jest.fn(), warn: jest.fn(), debug: jest.fn(), error: jest.fn() };
    const client = {
      config: { namespace, logger: () => logger, log_level: 'OFF' },
      admin: { putWorkflow },
    } as any;
    return {
      worker: new InternalWorker(client, { name: 'Test-Worker', handleKill: false }),
      putWorkflow,
    };
  }

  it('registers the same lowercase action ids the worker keys its registries by', async () => {
    const workflow = fixtureWorkflow();
    const { worker, putWorkflow } = fakeWorker(NAMESPACE);

    await worker.registerWorkflow(workflow);
    worker.registerDurableActions(workflow.definition);

    const [[request]] = putWorkflow.mock.calls as [[CreateWorkflowVersionRequest]];
    const registered = [
      ...request.tasks.map((t) => t.action),
      request.onFailureTask?.action,
    ].filter((a): a is string => !!a);

    expect(registered).toEqual([
      'acme_order-pipeline:validate',
      'acme_order-pipeline:charge',
      'acme_order-pipeline:on-success-task',
      'acme_order-pipeline:settle',
      'acme_order-pipeline:on-failure-task',
    ]);
    expect(registered.every((a) => a === a.toLowerCase())).toBe(true);
    expect(Object.keys(worker.action_registry).sort()).toEqual([...registered].sort());
    expect([...worker.durable_action_set]).toEqual(['acme_order-pipeline:settle']);
    expect(worker.workflow_registry[0].name).toBe('acme_order-pipeline');
  });
});
