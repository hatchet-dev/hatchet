import { HatchetClient } from '@hatchet-dev/typescript-sdk';

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
