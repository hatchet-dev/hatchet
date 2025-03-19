// ❓ Declaring a Workflow
import { hatchet } from '../hatchet-client';

// (optional) Define the input type for the workflow
export type SimpleInput = {
  Message: string;
};

// (optional) Define the output type for the workflow
export type SimpleOutput = {
  'to-lower': {
    TransformedMessage: string;
  };
};

// declare the workflow
export const simple = hatchet.workflow<SimpleInput, SimpleOutput>({
  name: 'simple',
});
// !!

// ❓ Binding a Task to a Workflow
// we can bind a task to the workflow by calling the `task` method on the workflow object
simple.task({
  name: 'to-lower',
  fn: async (input, ctx) => {
    return {
      TransformedMessage: input.Message.toLowerCase(),
    };
  },
});
// !!

// see ./worker.ts and ./run.ts for how to run the workflow
