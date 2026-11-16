import HatchetError, { getErrorMessage } from './hatchet-error';

/**
 * A bulk trigger that failed after at least one of its batches was accepted. The runs the
 * accepted batches created exist and are running; a caller that retries the whole request
 * would create them again, so `refs` carries their references, in input order, for the
 * caller to keep or cancel. `cause` is the error the failed batch raised, which is a
 * `BulkTriggerIdempotencyCollisionError` when that batch collided on an idempotency key.
 */
export class BulkTriggerPartialError<Ref = unknown> extends HatchetError {
  /** The references of every run created before the failed batch, in input order. */
  refs: Ref[];
  /** The zero-based index of the batch that failed. */
  failedBatchIndex: number;
  /** The error the failed batch raised. */
  cause: unknown;

  constructor(refs: Ref[], failedBatchIndex: number, cause: unknown) {
    super(
      `bulk trigger failed at batch ${failedBatchIndex} after ${refs.length} runs were created: ${getErrorMessage(cause)}`,
      { cause }
    );
    this.name = 'BulkTriggerPartialError';
    this.refs = refs;
    this.failedBatchIndex = failedBatchIndex;
    this.cause = cause;
    Object.setPrototypeOf(this, new.target.prototype);
  }
}
