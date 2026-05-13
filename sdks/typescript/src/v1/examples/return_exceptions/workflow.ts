import { hatchet } from '../hatchet-client';

export type ReturnExceptionsInput = {
  index: number;
};

export const returnExceptionsTask = hatchet.task({
  name: 'return-exceptions-task',
  fn: async (input: ReturnExceptionsInput) => {
    if (input.index % 2 === 0) {
      throw new Error(`error in task with index ${input.index}`);
    }
    return { message: 'this is a successful task.' };
  },
});
