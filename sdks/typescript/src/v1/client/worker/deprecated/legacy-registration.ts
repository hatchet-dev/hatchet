/**
 * Legacy worker registration using the deprecated `slots` proto field
 * instead of `slotConfig`. For backward compatibility with engines
 * that do not support multiple slot types.
 */

import {
  DispatcherClient,
  mapLabels,
  WorkerLabels,
} from '@hatchet/clients/dispatcher/dispatcher-client';
import { ActionListener } from '@clients/dispatcher/action-listener';

export interface LegacyRegistrationOptions {
  workerName: string;
  services: string[];
  actions: string[];
  slots: number;
  labels: WorkerLabels;
}

/**
 * Registers a worker using the legacy `slots` proto field instead of `slotConfig`.
 */
export async function legacyGetActionListener(
  dispatcher: DispatcherClient,
  options: LegacyRegistrationOptions
): Promise<ActionListener> {
  const registration = await dispatcher.client.register({
    workerName: options.workerName,
    services: options.services,
    actions: options.actions,
    slots: options.slots,
    labels: options.labels ? mapLabels(options.labels) : undefined,
    runtimeInfo: dispatcher.getRuntimeInfo(),
  });

  return new ActionListener(dispatcher, registration.workerId);
}
