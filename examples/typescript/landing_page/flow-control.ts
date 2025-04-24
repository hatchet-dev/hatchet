import { ConcurrencyLimitStrategy } from '@hatchet/protoc/v1/workflows';
import { hatchet } from '../hatchet-client';

// (optional) Define the input type for the workflow
export type SimpleInput = {
  Message: string;
};

// ❓ Process what you can handle
export const simple = hatchet.task({
  name: 'simple',
  concurrency: {
    expression: 'input.user_id',
    limitStrategy: ConcurrencyLimitStrategy.GROUP_ROUND_ROBIN,
    maxRuns: 1,
  },
  rateLimits: [
    {
      key: 'api_throttle',
      units: 1,
    },
  ],
  fn: (input: SimpleInput) => {
    return {
      TransformedMessage: input.Message.toLowerCase(),
    };
  },
});
// !!
