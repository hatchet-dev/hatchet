// ❓ Declaring a Task
import { hatchet } from '../hatchet-client';

// (optional) Define the input type for the workflow
export type SimpleInput = {
  Message: string;
};

export const simple = hatchet.task({
  name: 'simple',
  fn: (input: SimpleInput) => {
    return {
      TransformedMessage: input.Message.toLowerCase(),
    };
  },
});
// !!

// see ./worker.ts and ./run.ts for how to run the workflow
