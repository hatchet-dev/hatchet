// ❓ Declaring a Task
import { JsonObject } from '@hatchet/v1/types';
import { hatchet } from './hatchet-client';

// (optional) Define the input type for the workflow
export type SimpleInput = {
  Message: string;
};

export type SimpleOutput = {
  TransformedMessage: string;
};

export const withMiddleware = hatchet.task({
  name: 'with-middleware',
  middleware: [
    {
      input: {
        deserialize: (input: JsonObject): SimpleInput => {
          console.log('task-input-deserialize', input);
          return input as SimpleInput;
        },
        serialize: (input: unknown): JsonObject => {
          console.log('task-input-serialize', input);
          return input as JsonObject;
        },
      },
      output: {
        deserialize: (input: JsonObject): SimpleOutput => {
          console.log('task-output-deserialize', input);
          return input as SimpleOutput;
        },
        serialize: (input: unknown): JsonObject => {
          console.log('task-output-serialize', input);
          return input as JsonObject;
        },
      },
    },
  ],
  fn: (input: SimpleInput) => {
    return {
      TransformedMessage: input.Message.toLowerCase(),
    };
  },
});

// !!
