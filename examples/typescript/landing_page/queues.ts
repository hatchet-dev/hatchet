/* eslint-disable @typescript-eslint/no-unused-vars */
import { hatchet } from '../hatchet-client';

// (optional) Define the input type for the workflow
export type SimpleInput = {
  Message: string;
};
async function main() {
  // ❓ Declaring a Task
  const simple = hatchet.task({
    name: 'simple',
    fn: (input: SimpleInput) => {
      return {
        TransformedMessage: input.Message.toLowerCase(),
      };
    },
  });
  // !!

  // ❓ Running a Task
  const result = await simple.run({ Message: 'Hello, World!' });
  // !!
}

if (require.main === module) {
  main();
}
