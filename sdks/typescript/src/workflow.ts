import { ConcurrencyLimitStrategy, StickyStrategy } from '@hatchet/v1';
import { emitV0RemovedWarning } from './util/v0-deprecation-warning';

export * from './legacy/workflow';

export { ConcurrencyLimitStrategy, StickyStrategy };

emitV0RemovedWarning(
  'workflow',
  'ConcurrencyLimitStrategy and StickyStrategy have been moved to @hatchet-dev/typescript-sdk/v1.'
);
