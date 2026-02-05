import { BaseWorkflowDeclaration } from '../../declaration';
import { SlotConfig, SlotType } from '../../slot-types';

const DEFAULT_DEFAULT_SLOTS = 100;
const DEFAULT_DURABLE_SLOTS = 1_000;

export interface WorkerSlotOptions {
  /** (optional) Maximum number of concurrent runs on this worker, defaults to 100 */
  slots?: number;
  /** (optional) Maximum number of concurrent durable tasks, defaults to 1,000 */
  durableSlots?: number;
  /** (optional) Array of workflows to register */
  workflows?: BaseWorkflowDeclaration<any, any>[];
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
    options.slots || options.durableSlots || options.maxRuns
      ? {
          ...(options.slots || options.maxRuns
            ? { [SlotType.Default]: options.slots || options.maxRuns || 0 }
            : {}),
          ...(options.durableSlots ? { [SlotType.Durable]: options.durableSlots } : {}),
        }
      : {};

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

// eslint-disable-next-line @typescript-eslint/naming-convention
export const testingExports = {
  resolveWorkerOptions,
};

function getRequiredSlotTypes(workflows: Array<BaseWorkflowDeclaration<any, any>>): Set<SlotType> {
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
      // eslint-disable-next-line dot-notation
      const tasks = wf.definition['_tasks'] as Array<{ slotRequests?: Record<string, number> }>;
      for (const task of tasks) {
        addFromRequests(task.slotRequests, SlotType.Default);
      }
      // eslint-disable-next-line dot-notation
      const durableTasks = wf.definition['_durableTasks'] as Array<unknown>;
      if (durableTasks.length > 0) {
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
