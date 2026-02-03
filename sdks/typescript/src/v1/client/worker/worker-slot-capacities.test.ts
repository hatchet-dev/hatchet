import { WorkflowDeclaration } from '../../declaration';
import { SlotType } from '../../slot-types';
import { __testing } from '@hatchet/v1/client/worker/worker';

const { resolveWorkerOptions } = __testing;

describe('resolveWorkerOptions slot capacities', () => {
  it('sets default slots for non-durable tasks', () => {
    const workflow = new WorkflowDeclaration({ name: 'default-wf' });
    workflow.task({
      name: 'task1',
      fn: async () => undefined,
    });

    const resolved = resolveWorkerOptions({ workflows: [workflow] });

    expect(resolved.slotCapacities[SlotType.Default]).toBe(100);
    expect(resolved.slotCapacities[SlotType.Durable]).toBeUndefined();
  });

  it('sets durable slots for durable-only workflows without default slots', () => {
    const workflow = new WorkflowDeclaration({ name: 'durable-wf' });
    workflow.durableTask({
      name: 'durable-task',
      fn: async () => undefined,
    });

    const resolved = resolveWorkerOptions({ workflows: [workflow] });

    expect(resolved.slotCapacities[SlotType.Durable]).toBe(1000);
    expect(resolved.slotCapacities[SlotType.Default]).toBeUndefined();
    expect(resolved.slots).toBeUndefined();
  });

  it('sets both default and durable slots for mixed workflows', () => {
    const workflow = new WorkflowDeclaration({ name: 'mixed-wf' });
    workflow.task({
      name: 'task1',
      fn: async () => undefined,
    });
    workflow.durableTask({
      name: 'durable-task',
      fn: async () => undefined,
    });

    const resolved = resolveWorkerOptions({ workflows: [workflow] });

    expect(resolved.slotCapacities[SlotType.Default]).toBe(100);
    expect(resolved.slotCapacities[SlotType.Durable]).toBe(1000);
  });
});
