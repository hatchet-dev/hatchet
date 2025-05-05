/* eslint-disable no-console */
import { hatchet } from '../hatchet-client';
import { simple } from './workflow';

async function main() {
  // > Running a Task
  const res = await simple.run(
    {
      Message: 'HeLlO WoRlD',
    },
    {
      additionalMetadata: {
        test: 'test',
      },
    }
  );

  // 👀 Access the results of the Task
  console.log(res.TransformedMessage);
  // !!
}

export async function extra() {
  // > Running Multiple Tasks
  const res1 = simple.run({
    Message: 'HeLlO WoRlD',
  });

  const res2 = simple.run({
    Message: 'Hello MoOn',
  });

  const results = await Promise.all([res1, res2]);

  console.log(results[0].TransformedMessage);
  console.log(results[1].TransformedMessage);
  // !!

  // > Spawning Tasks from within a Task
  const parent = hatchet.task({
    name: 'parent',
    fn: async (input, ctx) => {
      // Simply call ctx.runChild with the task you want to run
      const child = await ctx.runChild(simple, {
        Message: 'HeLlO WoRlD',
      });

      return {
        result: child.TransformedMessage,
      };
    },
  });
  // !!
}

if (require.main === module) {
  main();
}
