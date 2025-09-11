/* eslint-disable no-console */
import { hatchet } from '../hatchet-client';
import { simple, SimpleInput } from './workflow';

async function main() {
  // > Bulk Run a Task
  const res = await simple.run([
    {
      Message: 'HeLlO WoRlD',
    },
    {
      Message: 'Hello MoOn',
    },
  ]);

  // 👀 Access the results of the Task
  console.log(res[0].TransformedMessage);
  console.log(res[1].TransformedMessage);
  // !!

  // > Bulk Run Tasks from within a Task
  const parent = hatchet.task({
    name: 'simple',
    fn: async (input: SimpleInput, ctx) => {
      // Bulk run two tasks in parallel
      const child = await ctx.bulkRunChildren([
        {
          workflow: simple,
          input: {
            Message: 'Hello, World!',
          },
        },
        {
          workflow: simple,
          input: {
            Message: 'Hello, Moon!',
          },
        },
      ]);

      return {
        TransformedMessage: `${child[0].TransformedMessage} ${child[1].TransformedMessage}`,
      };
    },
  });
  // !!
}

if (require.main === module) {
  main();
}
