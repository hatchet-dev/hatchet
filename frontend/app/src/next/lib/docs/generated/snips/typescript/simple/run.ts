import { Snippet } from '@/next/lib/docs/generated/snips/types';

const snippet: Snippet = {
  'language': 'typescript ',
  'content': '\nimport { hatchet } from \'../hatchet-client\';\nimport { simple } from \'./workflow\';\n\nasync function main() {\n  // > Running a Task\n  const res = await simple.run(\n    {\n      Message: \'HeLlO WoRlD\',\n    },\n    {\n      additionalMetadata: {\n        test: \'test\',\n      },\n    }\n  );\n\n  // 👀 Access the results of the Task\n  console.log(res.TransformedMessage);\n  \n}\n\nexport async function extra() {\n  // > Running Multiple Tasks\n  const res1 = simple.run({\n    Message: \'HeLlO WoRlD\',\n  });\n\n  const res2 = simple.run({\n    Message: \'Hello MoOn\',\n  });\n\n  const results = await Promise.all([res1, res2]);\n\n  console.log(results[0].TransformedMessage);\n  console.log(results[1].TransformedMessage);\n  \n\n  // > Spawning Tasks from within a Task\n  const parent = hatchet.task({\n    name: \'parent\',\n    fn: async (input, ctx) => {\n      // Simply call ctx.runChild with the task you want to run\n      const child = await ctx.runChild(simple, {\n        Message: \'HeLlO WoRlD\',\n      });\n\n      return {\n        result: child.TransformedMessage,\n      };\n    },\n  });\n  \n}\n\nif (require.main === module) {\n  main();\n}\n',
  'source': 'out/typescript/simple/run.ts',
  'blocks': {
    'running_a_task': {
      'start': 6,
      'stop': 18
    },
    'running_multiple_tasks': {
      'start': 23,
      'stop': 34
    },
    'spawning_tasks_from_within_a_task': {
      'start': 37,
      'stop': 49
    }
  },
  'highlights': {}
};  // Then replace double quotes with single quotes

export default snippet;
