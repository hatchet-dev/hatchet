/* eslint-disable no-underscore-dangle */
import { WorkerLabels } from '@hatchet/clients/dispatcher/dispatcher-client';
import { LegacyHatchetClient } from '@hatchet/clients/hatchet-client';
import { Workflow as V0Workflow } from '@hatchet/workflow';
import { WebhookWorkerCreateRequest } from '@hatchet/clients/rest/generated/data-contracts';
import { BaseWorkflowDeclaration } from '../../declaration';
import { HatchetClient } from '../..';
import { V1Worker } from './worker-internal';
import { SlotCapacities, SlotType } from '../../slot-types';

const DEFAULT_DEFAULT_SLOTS = 100;
const DEFAULT_DURABLE_SLOTS = 1_000;

/**
 * Options for creating a new hatchet worker
 * @interface CreateWorkerOpts
 */
export interface CreateWorkerOpts {
  /** (optional) Slot capacities for this worker (slot_type -> units). Defaults to { [SlotType.Default]: 100 }. */
  slotCapacities?: SlotCapacities;
  /** (optional) Maximum number of concurrent runs on this worker, defaults to 100 */
  slots?: number;
  /** (optional) Maximum number of concurrent durable tasks, defaults to 1,000 */
  durableSlots?: number;
  /** (optional) Array of workflows to register */
  workflows?: BaseWorkflowDeclaration<any, any>[] | V0Workflow[];
  /** (optional) Worker labels for affinity-based assignment */
  labels?: WorkerLabels;
  /** (optional) Whether to handle kill signals */
  handleKill?: boolean;
  /** @deprecated Use slots instead */
  maxRuns?: number;
}

/**
 * HatchetWorker class for workflow execution runtime
 */
export class Worker {
  config: CreateWorkerOpts;
  name: string;
  _v1: HatchetClient;
  _v0: LegacyHatchetClient;

  /** Internal reference to the underlying V0 worker implementation */
  nonDurable: V1Worker;

  /**
   * Creates a new HatchetWorker instance
   * @param nonDurable - The V0 worker implementation
   */
  constructor(
    v1: HatchetClient,
    v0: LegacyHatchetClient,
    nonDurable: V1Worker,
    config: CreateWorkerOpts,
    name: string
  ) {
    this._v1 = v1;
    this._v0 = v0;
    this.nonDurable = nonDurable;
    this.config = config;
    this.name = name;
  }

  /**
   * Creates and initializes a new HatchetWorker
   * @param v0 - The HatchetClient instance
   * @param options - Worker creation options
   * @returns A new HatchetWorker instance
   */
  static async create(
    v1: HatchetClient,
    v0: LegacyHatchetClient,
    name: string,
    options: CreateWorkerOpts
  ) {
    const hasSlotCapacities = options.slotCapacities !== undefined;
    const hasLegacySlots =
      options.slots !== undefined ||
      options.durableSlots !== undefined ||
      options.maxRuns !== undefined;
    if (hasSlotCapacities && hasLegacySlots) {
      throw new Error(
        'Cannot set both slotCapacities and slots/durableSlots. Use slotCapacities only.'
      );
    }

    const resolvedOptions = resolveWorkerOptions(options);
    const opts = {
      name,
      ...resolvedOptions,
    };

    const internalWorker = new V1Worker(v1, opts);
    const worker = new Worker(v1, v0, internalWorker, options, name);
    await worker.registerWorkflows(options.workflows);
    return worker;
  }

  /**
   * Registers workflows with the worker
   * @param workflows - Array of workflows to register
   * @returns Array of registered workflow promises
   */
  async registerWorkflows(workflows?: Array<BaseWorkflowDeclaration<any, any> | V0Workflow>) {
    for (const wf of workflows || []) {
      if (wf instanceof BaseWorkflowDeclaration) {
        // TODO check if tenant is V1
        await this.nonDurable.registerWorkflowV1(wf);

        if (wf.definition._durableTasks.length > 0) {
          this.nonDurable.registerDurableActionsV1(wf.definition);
        }
      } else {
        // fallback to v0 client for backwards compatibility
        await this.nonDurable.registerWorkflow(wf);
      }
    }
  }

  /**
   * Registers a single workflow with the worker
   * @param workflow - The workflow to register
   * @returns A promise that resolves when the workflow is registered
   * @deprecated use registerWorkflows instead
   */
  registerWorkflow(workflow: BaseWorkflowDeclaration<any, any> | V0Workflow) {
    return this.registerWorkflows([workflow]);
  }

  /**
   * Starts the worker
   * @returns Promise that resolves when the worker is stopped or killed
   */
  start() {
    return this.nonDurable.start();
  }

  /**
   * Stops the worker
   * @returns Promise that resolves when the worker stops
   */
  stop() {
    return this.nonDurable.stop();
  }

  /**
   * Updates or inserts worker labels
   * @param labels - Worker labels to update
   * @returns Promise that resolves when labels are updated
   */
  upsertLabels(labels: WorkerLabels) {
    return this.nonDurable.upsertLabels(labels);
  }

  /**
   * Get the labels for the worker
   * @returns The labels for the worker
   */
  getLabels() {
    return this.nonDurable.labels;
  }

  /**
   * Register a webhook with the worker
   * @param webhook - The webhook to register
   * @returns A promise that resolves when the webhook is registered
   */
  registerWebhook(webhook: WebhookWorkerCreateRequest) {
    return this.nonDurable.registerWebhook(webhook);
  }

  async isPaused() {
    if (!this.nonDurable?.workerId) {
      return false;
    }

    return this._v1.workers.isPaused(this.nonDurable.workerId);
  }

  // TODO docstrings
  pause() {
    if (!this.nonDurable?.workerId) {
      return Promise.resolve();
    }

    return this._v1.workers.pause(this.nonDurable.workerId);
  }

  unpause() {
    if (!this.nonDurable?.workerId) {
      return Promise.resolve();
    }

    return this._v1.workers.unpause(this.nonDurable.workerId);
  }
}

function resolveWorkerOptions(options: CreateWorkerOpts) {
  const requiredSlotTypes =
    options.workflows
      ? getRequiredSlotTypes(options.workflows)
      : new Set<SlotType>();

  const slotCapacities: SlotCapacities =
    options.slotCapacities ||
    (options.slots || options.durableSlots || options.maxRuns
      ? {
          ...(options.slots || options.maxRuns
            ? { [SlotType.Default]: options.slots || options.maxRuns || 0 }
            : {}),
          ...(options.durableSlots ? { [SlotType.Durable]: options.durableSlots } : {}),
        }
      : {});

  if (requiredSlotTypes.has(SlotType.Default) && slotCapacities[SlotType.Default] == null) {
    slotCapacities[SlotType.Default] = DEFAULT_DEFAULT_SLOTS;
  }
  if (requiredSlotTypes.has(SlotType.Durable) && slotCapacities[SlotType.Durable] == null) {
    slotCapacities[SlotType.Durable] = DEFAULT_DURABLE_SLOTS;
  }

  if (Object.keys(slotCapacities).length === 0) {
    slotCapacities[SlotType.Default] = DEFAULT_DEFAULT_SLOTS;
  }

  return {
    ...options,
    slots:
      options.slots ||
      options.maxRuns ||
      (slotCapacities[SlotType.Default] != null ? slotCapacities[SlotType.Default] : undefined),
    durableSlots:
      options.durableSlots ||
      (slotCapacities[SlotType.Durable] != null ? slotCapacities[SlotType.Durable] : undefined),
    slotCapacities,
  };
}

export const __testing = {
  resolveWorkerOptions,
};

function getRequiredSlotTypes(
  workflows: Array<BaseWorkflowDeclaration<any, any> | V0Workflow>
): Set<SlotType> {
  const required = new Set<SlotType>();
  const addFromRequirements = (
    requirements: Record<string, number> | undefined,
    fallbackType: SlotType
  ) => {
    if (requirements && Object.keys(requirements).length > 0) {
      if (requirements[SlotType.Default] !== undefined) {
        required.add(SlotType.Default);
      }
      if (requirements[SlotType.Durable] !== undefined) {
        required.add(SlotType.Durable);
      }
    } else {
      required.add(fallbackType);
    }
  };

  for (const wf of workflows) {
    if (wf instanceof BaseWorkflowDeclaration) {
      for (const task of wf.definition._tasks) {
        addFromRequirements(task.slotRequirements, SlotType.Default);
      }
      for (const task of wf.definition._durableTasks) {
        required.add(SlotType.Durable);
      }

      if (wf.definition.onFailure) {
        const opts = typeof wf.definition.onFailure === 'object' ? wf.definition.onFailure : undefined;
        addFromRequirements(opts?.slotRequirements, SlotType.Default);
      }

      if (wf.definition.onSuccess) {
        const opts = typeof wf.definition.onSuccess === 'object' ? wf.definition.onSuccess : undefined;
        addFromRequirements(opts?.slotRequirements, SlotType.Default);
      }
    }
  }

  return required;
}
