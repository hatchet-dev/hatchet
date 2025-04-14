import { HatchetClient } from '../../client/client';

export const hatchet = HatchetClient.init({
  middleware: {
    deserialize: (input) => {
      console.log('client-deserialize', input);
      return input;
    },
    serialize: async (input) => {
      console.log('client-serialize', input);
      return input;
    },
  },
});
