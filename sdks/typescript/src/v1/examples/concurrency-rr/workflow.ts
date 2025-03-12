import { ConcurrencyLimitStrategy } from '@hatchet/workflow';
import { hatchet } from '../client';

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

export const simpleConcurrency = hatchet.workflow<SimpleInput, SimpleOutput>({
  name: 'simple-concurrency',
  concurrency: {
    name: 'simple-concurrency',
    maxRuns: 1,
    limitStrategy: ConcurrencyLimitStrategy.GROUP_ROUND_ROBIN,
    expression: 'input.GroupKey',
  },
});

simpleConcurrency.task({
  name: 'to-lower',
  fn: async (input) => {
    await sleep(10_000);
    return {
      TransformedMessage: input.Message.toLowerCase(),
    };
  },
});
