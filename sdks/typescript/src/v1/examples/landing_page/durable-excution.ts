/* eslint-disable @typescript-eslint/no-unused-vars */
import { Or } from '@hatchet/v1/conditions';
import { hatchet } from '../hatchet-client';

// (optional) Define the input type for the workflow
export type SimpleInput = {
  Message: string;
};
async function main() {
  // ❓ Declaring a Durable Task
  const simple = hatchet.durableTask({
    name: 'simple',
    fn: async (input: SimpleInput, ctx) => {
      await ctx.waitFor(
        Or(
          {
            eventKey: 'user:pay',
            expression: 'input.Status == "PAID"',
          },
          {
            sleepFor: '24h',
          }
        )
      );

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
