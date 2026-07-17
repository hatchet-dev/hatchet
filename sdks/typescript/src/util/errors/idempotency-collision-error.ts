import HatchetError from './hatchet-error';

export class IdempotencyCollisionError extends HatchetError {
  existingRunExternalId: string;

  constructor(existingRunExternalId: string) {
    super(`idempotency key collision: existing run ${existingRunExternalId} already exists`);
    this.name = 'IdempotencyCollisionError';
    this.existingRunExternalId = existingRunExternalId;
    Object.setPrototypeOf(this, new.target.prototype);
  }
}
