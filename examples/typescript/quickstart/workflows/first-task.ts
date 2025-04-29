import { hatchet } from '../../hatchet-client';

type SimpleInput = {
  Message: string;
};

type SimpleOutput = {
  TransformedMessage: string;
};

export const firstTask = hatchet.task({
  name: 'first-task',
  fn: (input: SimpleInput, ctx): SimpleOutput => {
    return {
      TransformedMessage: input.Message.toLowerCase(),
    };
  },
});
