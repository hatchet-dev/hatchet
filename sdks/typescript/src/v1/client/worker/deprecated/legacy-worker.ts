/**
 * Legacy dual-worker implementation for pre-slot-config engines.
 *
 * When connected to an older Hatchet engine that does not support multiple slot types,
 * this module provides the old worker start flow which creates separate durable and
 * non-durable workers, each registered with the legacy `slots` proto field.
 */

import { Status } from 'nice-grpc';
import { BaseWorkflowDeclaration } from '../../../declaration';
import { HatchetClient } from '../../..';
import { CreateWorkerOpts } from '../worker';
import { LegacyV1Worker } from './legacy-v1-worker';
import { emitDeprecationNotice, semverLessThan } from './deprecation';
import { transformLegacyWorkflow } from '../../../../legacy/legacy-transformer';

import { MinEngineVersion } from '../engine-version';
import { getGrpcErrorCode } from '@hatchet-dev/typescript-sdk/util/grpc-error';

const DEFAULT_DEFAULT_SLOTS = 100;
const DEFAULT_DURABLE_SLOTS = 1_000;

/** The date when slot_config support was released. */
export const LEGACY_ENGINE_START = new Date('2026-02-12T00:00:00Z');

export const LEGACY_ENGINE_MESSAGE =
  'Connected to an older Hatchet engine that does not support multiple slot types. ' +
  'Falling back to legacy worker registration. ' +
  'Please upgrade your Hatchet engine to the latest version.';

/**
 * Fetches the engine version from the dispatcher.
 * Returns the semver string, or undefined if the engine is too old to support GetVersion.
 */
export async function fetchEngineVersion(v1: HatchetClient): Promise<string | undefined> {
  try {
    const version = await v1.dispatcher.getVersion();
    return version || undefined;
  } catch (e: unknown) {
    if (getGrpcErrorCode(e) == Status.UNIMPLEMENTED) {
      return undefined;
    }
    throw e;
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
          await durableWorker.registerWorkflow(wf);
          durableWorker.registerDurableActions(wf.definition);
        } else {
          await nonDurable.registerWorkflow(wf);
        }
      } else {
        // fallback to v0 client for backwards compatibility
        await nonDurable.registerWorkflow(transformLegacyWorkflow(wf));
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
