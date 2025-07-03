import { Snippet } from '@/next/lib/docs/generated/snips/types';

const snippet: Snippet = {
  language: 'typescript ',
  content:
    "import { hatchet } from '../hatchet-client';\nimport { simple } from './workflow';\nimport { parent } from './workflow-with-child';\n\nasync function main() {\n  // > Running a Task\n  const res = await parent.run(\n    {\n      Message: 'HeLlO WoRlD',\n    },\n    {\n      additionalMetadata: {\n        test: 'test',\n      },\n    }\n  );\n\n  // 👀 Access the results of the Task\n  console.log(res.TransformedMessage);\n}\n\nexport async function extra() {\n  // > Running Multiple Tasks\n  const res1 = simple.run({\n    Message: 'HeLlO WoRlD',\n  });\n\n  const res2 = simple.run({\n    Message: 'Hello MoOn',\n  });\n\n  const results = await Promise.all([res1, res2]);\n\n  console.log(results[0].TransformedMessage);\n  console.log(results[1].TransformedMessage);\n\n  // > Spawning Tasks from within a Task\n  const parentTask = hatchet.task({\n    name: 'parent',\n    fn: async (input, ctx) => {\n      // Simply the task and it will be spawned from the parent task\n      const child = await simple.run({\n        Message: 'HeLlO WoRlD',\n      });\n\n      return {\n        result: child.TransformedMessage,\n      };\n    },\n  });\n}\n\nif (require.main === module) {\n  main()\n    .catch(console.error)\n    .finally(() => {\n      process.exit(0);\n    });\n}\n",
  source: 'out/typescript/simple/run.ts',
  blocks: {
    running_a_task: {
      start: 7,
      stop: 19,
    },
    running_multiple_tasks: {
      start: 24,
      stop: 35,
    },
    spawning_tasks_from_within_a_task: {
      start: 38,
      stop: 50,
    },
  },
  highlights: {},
};

export default snippet;
