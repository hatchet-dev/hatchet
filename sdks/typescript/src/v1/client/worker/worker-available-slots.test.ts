import { InternalWorker } from '@hatchet/v1/client/worker/worker-internal';
import { WorkflowDeclaration } from '../../declaration';
import { SlotType } from '../../slot-types';

const getAvailableSlots = (fakeThis: any): number =>
  (InternalWorker.prototype as any).getAvailableSlots.call({
    ...fakeThis,
    // getAvailableSlots delegates to the per-pool breakdown.
    getAvailableSlotsByPool: (InternalWorker.prototype as any).getAvailableSlotsByPool,
  });

const getAvailableSlotsByPool = (fakeThis: any): Record<string, number> =>
  (InternalWorker.prototype as any).getAvailableSlotsByPool.call(fakeThis);

const slotRequestsForAction = (fakeThis: any, actionId: string): Record<string, number> =>
  (InternalWorker.prototype as any).slotRequestsForAction.call(fakeThis, actionId);

describe('InternalWorker available slot reporting', () => {
  it('charges a running task its actual slot cost', () => {
    const fakeThis: any = {
      slotConfig: { default: 10 },
      running_slot_requests: { 'expensive-task-run/0': { default: 5 } },
    };

    expect(getAvailableSlots(fakeThis)).toBe(5);
    expect(getAvailableSlotsByPool(fakeThis)).toEqual({ default: 5 });
  });

  it('does not conflate the default and durable pools', () => {
    const fakeThis: any = {
      slotConfig: { default: 5, durable: 3 },
      running_slot_requests: {
        'run-1/0': { default: 1 },
        'run-2/0': { default: 1 },
        'run-3/0': { default: 1 },
        'run-4/0': { default: 1 },
        'run-5/0': { default: 1 },
      },
    };

    // The default pool is full, so no queued non-durable task can be picked up, even
    // though the durable pool still has room.
    expect(getAvailableSlots(fakeThis)).toBe(0);
    expect(getAvailableSlotsByPool(fakeThis)).toEqual({ default: 0, durable: 3 });
  });

  it('reports the durable pool independently', () => {
    const fakeThis: any = {
      slotConfig: { default: 5, durable: 3 },
      running_slot_requests: {
        'run-1/0': { durable: 1 },
        'run-2/0': { durable: 1 },
      },
    };

    expect(getAvailableSlotsByPool(fakeThis)).toEqual({ default: 5, durable: 1 });
    expect(getAvailableSlots(fakeThis)).toBe(1);
  });

  it('reports full capacity when nothing is running', () => {
    const fakeThis: any = {
      slotConfig: { default: 100, durable: 1000 },
      running_slot_requests: {},
    };

    expect(getAvailableSlots(fakeThis)).toBe(100);
    expect(getAvailableSlotsByPool(fakeThis)).toEqual({ default: 100, durable: 1000 });
  });

  it('never reports negative capacity when running costs exceed the pool', () => {
    const fakeThis: any = {
      slotConfig: { default: 2 },
      running_slot_requests: {
        'run-1/0': { default: 5 },
      },
    };

    expect(getAvailableSlots(fakeThis)).toBe(0);
    expect(getAvailableSlotsByPool(fakeThis)).toEqual({ default: 0 });
  });

  it('ignores costs charged against a pool the worker has not configured', () => {
    const fakeThis: any = {
      slotConfig: { default: 5 },
      running_slot_requests: {
        'run-1/0': { durable: 1 },
      },
    };

    expect(getAvailableSlotsByPool(fakeThis)).toEqual({ default: 5 });
  });

  it('reports zero when no slot pools are configured', () => {
    const fakeThis: any = { slotConfig: {}, running_slot_requests: {} };

    expect(getAvailableSlots(fakeThis)).toBe(0);
    expect(getAvailableSlotsByPool(fakeThis)).toEqual({});
  });
});

describe('InternalWorker.slotRequestsForAction', () => {
  it('uses the slot requests recorded when the action was registered', () => {
    const fakeThis: any = {
      action_slot_requests: { 'wf:heavy': { default: 5 } },
      durable_action_set: new Set<string>(),
    };

    expect(slotRequestsForAction(fakeThis, 'wf:heavy')).toEqual({ default: 5 });
  });

  it('falls back to one default slot for an unregistered action', () => {
    const fakeThis: any = {
      action_slot_requests: {},
      durable_action_set: new Set<string>(),
    };

    expect(slotRequestsForAction(fakeThis, 'wf:unknown')).toEqual({ [SlotType.Default]: 1 });
  });

  it('falls back to one durable slot for an unregistered durable action', () => {
    const fakeThis: any = {
      action_slot_requests: {},
      durable_action_set: new Set(['wf:durable']),
    };

    expect(slotRequestsForAction(fakeThis, 'wf:durable')).toEqual({ [SlotType.Durable]: 1 });
  });
});

describe('InternalWorker action registration records slot requests', () => {
  it('records slotCost for tasks and the durable pool for durable tasks', () => {
    const workflow = new WorkflowDeclaration({ name: 'slots-wf' });
    workflow.task({ name: 'cheap', fn: async () => undefined });
    workflow.task({ name: 'heavy', slotCost: 5, fn: async () => undefined });
    workflow.durableTask({ name: 'durable-task', fn: async () => undefined });

    const fakeThis: any = {
      action_registry: {},
      action_slot_requests: {},
      durable_action_set: new Set<string>(),
      eviction_policies: new Map(),
      client: { config: { namespace: '' } },
    };

    (InternalWorker.prototype as any).registerActions.call(fakeThis, {
      ...workflow.definition,
      name: 'slots-wf',
    });
    InternalWorker.prototype.registerDurableActions.call(fakeThis, {
      ...workflow.definition,
      name: 'slots-wf',
    } as any);

    expect(fakeThis.action_slot_requests['slots-wf:cheap']).toEqual({ default: 1 });
    expect(fakeThis.action_slot_requests['slots-wf:heavy']).toEqual({ default: 5 });
    expect(fakeThis.action_slot_requests['slots-wf:durable-task']).toEqual({ durable: 1 });
  });
});

describe('InternalWorker.cleanupRun releases held slots', () => {
  it('drops the run from the slot bookkeeping', () => {
    const fakeThis: any = {
      slotConfig: { default: 10 },
      futures: { 'run-1/0': {} },
      contexts: {},
      running_slot_requests: { 'run-1/0': { default: 5 } },
      evictionManager: undefined,
      client: { durableListener: { cleanupTaskState: jest.fn() } },
    };

    expect(getAvailableSlots(fakeThis)).toBe(5);

    (InternalWorker.prototype as any).cleanupRun.call(fakeThis, 'run-1/0');

    expect(fakeThis.running_slot_requests['run-1/0']).toBeUndefined();
    expect(getAvailableSlots(fakeThis)).toBe(10);
  });
});
