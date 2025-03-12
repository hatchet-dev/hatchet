import { hatchet } from '../client';

type SimpleInput = {
  Message: string;
};

export const simple = hatchet.workflow<SimpleInput>({
  name: 'simple',
});

simple.task({
  name: 'to-lower',
  fn: (input) => {
    return {
      TransformedMessage: input.Message.toLowerCase(),
    };
  },
});
