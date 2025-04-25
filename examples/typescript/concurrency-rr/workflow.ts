import { ConcurrencyLimitStrategy } from '@hatchet-dev/typescript-sdk/workflow';
import { hatchet } from '../hatchet-client';

type SimpleInput = {
  Message: string;
  GroupKey: string;
};

type SimpleOutput = {
  'to-lower': {
    TransformedMessage: string;
  };
};

const sleep = (ms: number) =>
  new Promise((resolve) => {
    setTimeout(resolve, ms);
  });

// ❓ Concurrency Strategy With Key
export const simpleConcurrency = hatchet.workflow<SimpleInput, SimpleOutput>({
  name: 'simple-concurrency',
  concurrency: {
    maxRuns: 1,
    limitStrategy: ConcurrencyLimitStrategy.GROUP_ROUND_ROBIN,
    expression: 'input.GroupKey',
  },
});

simpleConcurrency.task({
  name: 'to-lower',
  fn: async (input) => {
    await sleep(Math.floor(Math.random() * (1000 - 200 + 1)) + 200);
    return {
      TransformedMessage: input.Message.toLowerCase(),
    };
  },
});

// ❓ Multiple Concurrency Keys
export const multipleConcurrencyKeys = hatchet.workflow<SimpleInput, SimpleOutput>({
  name: 'simple-concurrency',
  concurrency: [
    {
      maxRuns: 1,
      limitStrategy: ConcurrencyLimitStrategy.GROUP_ROUND_ROBIN,
      expression: 'input.Tier',
    },
    {
      maxRuns: 1,
      limitStrategy: ConcurrencyLimitStrategy.GROUP_ROUND_ROBIN,
      expression: 'input.Account',
    },
  ],
});

multipleConcurrencyKeys.task({
  name: 'to-lower',
  fn: async (input) => {
    await sleep(Math.floor(Math.random() * (1000 - 200 + 1)) + 200);
    return {
      TransformedMessage: input.Message.toLowerCase(),
    };
  },
});
