export * from './legacy/workflow';

import { ConcurrencyLimitStrategy, StickyStrategy } from '@hatchet/v1';

export { ConcurrencyLimitStrategy, StickyStrategy };

console.warn(
  '\x1b[31mDeprecation warning: The v0 sdk, including the workflow module has been deprecated and has been removed in release v1.12.0.\x1b[0m'
);
console.warn(
  '\x1b[31mPlease migrate to v1 SDK instead: https://docs.hatchet.run/home/v1-sdk-improvements\x1b[0m'
);
console.warn(
  'ConcurrencyLimitStrategy, StickyStrategy have been moved to @hatchet-dev/typescript-sdk/v1'
);
console.warn('--------------------------------');
