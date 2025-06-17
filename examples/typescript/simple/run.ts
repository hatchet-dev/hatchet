import { hatchet } from '../hatchet-client';
import { simple } from './workflow';
import { parent } from './workflow-with-child';

async function main() {
  // > Running a Task
  const res = await parent.run(
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

  // > Spawning Tasks from within a Task
  const parentTask = hatchet.task({
    name: 'parent',
    fn: async (input, ctx) => {
      // Simply the task and it will be spawned from the parent task
      const child = await simple.run({
        Message: 'HeLlO WoRlD',
      });

      return {
        result: child.TransformedMessage,
      };
    },
  });
}

if (require.main === module) {
  main()
    .catch(console.error)
    .finally(() => {
      process.exit(0);
    });
}
