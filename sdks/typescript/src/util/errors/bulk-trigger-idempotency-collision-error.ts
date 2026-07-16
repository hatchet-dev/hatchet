import { IdempotencyCollisionError } from './idempotency-collision-error';

export class BulkTriggerIdempotencyCollisionError extends Error {
  successfulWorkflowRunExternalIds: string[];
  collisions: IdempotencyCollisionError[];

  constructor(successfulWorkflowRunExternalIds: string[], collisions: IdempotencyCollisionError[]) {
    super('idempotency key collision in bulk trigger');
    this.name = 'BulkTriggerIdempotencyCollisionError';
    this.successfulWorkflowRunExternalIds = successfulWorkflowRunExternalIds;
    this.collisions = collisions;
    Object.setPrototypeOf(this, new.target.prototype);
  }
}
