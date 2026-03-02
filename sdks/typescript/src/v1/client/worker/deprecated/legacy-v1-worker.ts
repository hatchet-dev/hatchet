/**
 * Legacy V1Worker subclass that registers with the old `slots` proto field
 * instead of `slotConfig`. Used when connected to pre-slot-config engines.
 */

/* eslint-disable no-underscore-dangle */
import { ActionListener } from '@clients/dispatcher/action-listener';
import { HatchetClient } from '@hatchet/v1';
import { InternalWorker } from '../worker-internal';
import { legacyGetActionListener } from './legacy-registration';

export class LegacyV1Worker extends InternalWorker {
  private _legacySlotCount: number;

  constructor(
    client: HatchetClient,
    options: ConstructorParameters<typeof InternalWorker>[1],
    legacySlots: number
  ) {
    super(client, options);
    this._legacySlotCount = legacySlots;
  }

  /**
   * Override registration to use the legacy `slots` proto field.
   */
  protected override async createListener(): Promise<ActionListener> {
    return legacyGetActionListener(this.client.dispatcher, {
      workerName: this.name,
      services: ['default'],
      actions: Object.keys(this.action_registry),
      slots: this._legacySlotCount,
      labels: this.labels,
    });
  }
}
