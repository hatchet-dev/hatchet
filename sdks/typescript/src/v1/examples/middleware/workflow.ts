// ❓ Declaring a Task
import { hatchet } from './hatchet-client';

// (optional) Define the input type for the workflow
export type SimpleInput = {
  Message: string;
};

export const withMiddleware = hatchet.task({
  name: 'withMiddleware',
  middleware: {
    deserialize: (input) => {
      console.log('task-deserialize', input);
      return input;
    },
    serialize: (input) => {
      console.log('task-serialize', input);
      return input;
    },
  },
  fn: (input: SimpleInput) => {
    return {
      TransformedMessage: input.Message.toLowerCase(),
    };
  },
});

// !!
