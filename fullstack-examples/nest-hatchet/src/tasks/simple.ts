// ❓ Declaring a Task
import { HatchetClient } from '@hatchet-dev/typescript-sdk';

// (optional) Define the input type for the workflow
export type SimpleInput = {
  Message: string;
};

export const simple = (client: HatchetClient) =>
  client.task({
    name: 'simple',
    fn: (input: SimpleInput) => {
      return {
        TransformedMessage: input.Message.toLowerCase(),
      };
    },
  });

// !!

// see ./worker.ts and ./run.ts for how to run the workflow
