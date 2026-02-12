/**
 * Legacy dual-worker implementation for pre-slot-config engines.
 *
 * When connected to an older Hatchet engine that does not support multiple slot types,
 * this module provides the old worker start flow which creates separate durable and
 * non-durable workers, each registered with the legacy `slots` proto field.
 */

/* eslint-disable no-underscore-dangle */
import { Workflow as V0Workflow } from '@hatchet/workflow';
import { Status } from 'nice-grpc';
import { BaseWorkflowDeclaration } from '../../../declaration';
import { HatchetClient } from '../../..';
import { CreateWorkerOpts } from '../worker';
import { LegacyV1Worker } from './legacy-v1-worker';
import { emitDeprecationNotice } from './deprecation';

const DEFAULT_DEFAULT_SLOTS = 100;
const DEFAULT_DURABLE_SLOTS = 1_000;

/** The date when slot_config support was released. */
const LEGACY_ENGINE_START = new Date('2026-02-12T00:00:00Z');

const LEGACY_ENGINE_MESSAGE =
  'Connected to an older Hatchet engine that does not support multiple slot types. ' +
  'Falling back to legacy worker registration. ' +
  'Please upgrade your Hatchet engine to the latest version.';

/**
 * Checks if the connected engine is legacy (does not implement GetVersion).
 * Returns true if the engine is legacy, false otherwise.
 * Emits a time-aware deprecation notice when a legacy engine is detected.
 */
export async function isLegacyEngine(v1: HatchetClient): Promise<boolean> {
  try {
    await v1._v0.dispatcher.getVersion();
    return false;
  } catch (e: any) {
    if (e?.code === Status.UNIMPLEMENTED) {
      const logger = v1._v0.config.logger('Worker', v1._v0.config.log_level);
      emitDeprecationNotice('legacy-engine', LEGACY_ENGINE_MESSAGE, LEGACY_ENGINE_START, logger, {
        errorDays: 180,
      });
      return true;
    }
    // For other errors, assume new engine and let registration fail naturally
    return false;
  }
}

/**
 * LegacyDualWorker manages two V1Worker instances (nonDurable + durable)
 * for engines that don't support slot_config.
 * Uses the legacy `slots` proto field (maxRuns) instead of `slotConfig`.
 */
export class LegacyDualWorker {
  private nonDurable: LegacyV1Worker;
  private durable: LegacyV1Worker | undefined;
  private name: string;

  constructor(name: string, nonDurable: LegacyV1Worker, durable?: LegacyV1Worker) {
    this.name = name;
    this.nonDurable = nonDurable;
    this.durable = durable;
  }

  /**
   * Creates a legacy dual-worker setup from the given options.
   * Workers are created with legacy registration (old `slots` proto field).
   */
  static async create(
    v1: HatchetClient,
    name: string,
    options: CreateWorkerOpts
  ): Promise<LegacyDualWorker> {
    const defaultSlots = options.slots || options.maxRuns || DEFAULT_DEFAULT_SLOTS;
    const durableSlots = options.durableSlots || DEFAULT_DURABLE_SLOTS;

    // Create the non-durable worker with legacy registration
    const nonDurable = new LegacyV1Worker(
      v1,
      { name, labels: options.labels, handleKill: options.handleKill },
      defaultSlots
    );

    // Check if any workflows have durable tasks
    let hasDurableTasks = false;
    for (const wf of options.workflows || []) {
      if (wf instanceof BaseWorkflowDeclaration) {
        if (wf.definition._durableTasks.length > 0) {
          hasDurableTasks = true;
          break;
        }
      }
    }

    let durableWorker: LegacyV1Worker | undefined;
    if (hasDurableTasks) {
      // Create the durable worker with legacy registration
      durableWorker = new LegacyV1Worker(
        v1,
        { name: `${name}-durable`, labels: options.labels, handleKill: options.handleKill },
        durableSlots
      );
    }

    const legacyWorker = new LegacyDualWorker(name, nonDurable, durableWorker);

    // Register workflows on appropriate workers
    for (const wf of options.workflows || []) {
      if (wf instanceof BaseWorkflowDeclaration) {
        if (wf.definition._durableTasks.length > 0 && durableWorker) {
          await durableWorker.registerWorkflowV1(wf);
          durableWorker.registerDurableActionsV1(wf.definition);
        } else {
          await nonDurable.registerWorkflowV1(wf);
        }
      } else {
        // fallback to v0 client for backwards compatibility
        await nonDurable.registerWorkflow(wf as V0Workflow);
      }
    }

    return legacyWorker;
  }

  /**
   * Starts both workers using Promise.all.
   */
  async start(): Promise<void> {
    const promises: Promise<void>[] = [this.nonDurable.start()];
    if (this.durable) {
      promises.push(this.durable.start());
    }
    await Promise.all(promises);
  }

  /**
   * Stops both workers.
   */
  async stop(): Promise<void> {
    const promises: Promise<void>[] = [this.nonDurable.stop()];
    if (this.durable) {
      promises.push(this.durable.stop());
    }
    await Promise.all(promises);
  }
}
