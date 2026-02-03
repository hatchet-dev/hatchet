import { Workflow as V0Workflow } from '@hatchet/workflow';
import { BaseWorkflowDeclaration } from '../../declaration';
import { SlotConfig, SlotType } from '../../slot-types';

const DEFAULT_DEFAULT_SLOTS = 100;
const DEFAULT_DURABLE_SLOTS = 1_000;

export interface WorkerSlotOptions {
  /** (optional) Slot config for this worker (slot_type -> units). Defaults to { [SlotType.Default]: 100 }. */
  slotConfig?: SlotConfig;
  /** (optional) Maximum number of concurrent runs on this worker, defaults to 100 */
  slots?: number;
  /** (optional) Maximum number of concurrent durable tasks, defaults to 1,000 */
  durableSlots?: number;
  /** (optional) Array of workflows to register */
  workflows?: BaseWorkflowDeclaration<any, any>[] | V0Workflow[];
  /** @deprecated Use slots instead */
  maxRuns?: number;
}

export function resolveWorkerOptions<T extends WorkerSlotOptions>(
  options: T
): T & {
  slots?: number;
  durableSlots?: number;
  slotConfig: SlotConfig;
} {
  const requiredSlotTypes = options.workflows
    ? getRequiredSlotTypes(options.workflows)
    : new Set<SlotType>();

  const slotConfig: SlotConfig =
    options.slotConfig ||
    (options.slots || options.durableSlots || options.maxRuns
      ? {
          ...(options.slots || options.maxRuns
            ? { [SlotType.Default]: options.slots || options.maxRuns || 0 }
            : {}),
          ...(options.durableSlots ? { [SlotType.Durable]: options.durableSlots } : {}),
        }
      : {});

  if (requiredSlotTypes.has(SlotType.Default) && slotConfig[SlotType.Default] == null) {
    slotConfig[SlotType.Default] = DEFAULT_DEFAULT_SLOTS;
  }
  if (requiredSlotTypes.has(SlotType.Durable) && slotConfig[SlotType.Durable] == null) {
    slotConfig[SlotType.Durable] = DEFAULT_DURABLE_SLOTS;
  }

  if (Object.keys(slotConfig).length === 0) {
    slotConfig[SlotType.Default] = DEFAULT_DEFAULT_SLOTS;
  }

  return {
    ...options,
    slots:
      options.slots ||
      options.maxRuns ||
      (slotConfig[SlotType.Default] != null ? slotConfig[SlotType.Default] : undefined),
    durableSlots:
      options.durableSlots ||
      (slotConfig[SlotType.Durable] != null ? slotConfig[SlotType.Durable] : undefined),
    slotConfig,
  };
}

export const __testing = {
  resolveWorkerOptions,
};

function getRequiredSlotTypes(
  workflows: Array<BaseWorkflowDeclaration<any, any> | V0Workflow>
): Set<SlotType> {
  const required = new Set<SlotType>();
  const addFromRequests = (
    requests: Record<string, number> | undefined,
    fallbackType: SlotType
  ) => {
    if (requests && Object.keys(requests).length > 0) {
      if (requests[SlotType.Default] !== undefined) {
        required.add(SlotType.Default);
      }
      if (requests[SlotType.Durable] !== undefined) {
        required.add(SlotType.Durable);
      }
    } else {
      required.add(fallbackType);
    }
  };

  for (const wf of workflows) {
    if (wf instanceof BaseWorkflowDeclaration) {
      for (const task of wf.definition._tasks) {
        addFromRequests(task.slotRequests, SlotType.Default);
      }
      for (const task of wf.definition._durableTasks) {
        required.add(SlotType.Durable);
      }

      if (wf.definition.onFailure) {
        const opts =
          typeof wf.definition.onFailure === 'object' ? wf.definition.onFailure : undefined;
        addFromRequests(opts?.slotRequests, SlotType.Default);
      }

      if (wf.definition.onSuccess) {
        const opts =
          typeof wf.definition.onSuccess === 'object' ? wf.definition.onSuccess : undefined;
        addFromRequests(opts?.slotRequests, SlotType.Default);
      }
    }
  }

  return required;
}
