import sleep from '@hatchet/util/sleep';
import { hatchet } from '../hatchet-client';

type SimpleInput = {
  Message: string;
};

export const simple = hatchet.workflow<SimpleInput>({
  name: 'simple',
});

simple.task({
  name: 'to-lower',
  fn: async (input) => {
    await sleep(2000);
    return {
      TransformedMessage: input.Message.toLowerCase(),
    };
  },
});
