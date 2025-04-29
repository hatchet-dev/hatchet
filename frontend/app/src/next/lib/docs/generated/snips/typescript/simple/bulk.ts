import { Snippet } from '@/next/lib/docs/generated/snips/types';

const snippet: Snippet = {
  'language': 'typescript ',
  'content': '\nimport { hatchet } from \'../hatchet-client\';\nimport { simple, SimpleInput } from \'./workflow\';\n\nasync function main() {\n  // > Bulk Run a Task\n  const res = await simple.run([\n    {\n      Message: \'HeLlO WoRlD\',\n    },\n    {\n      Message: \'Hello MoOn\',\n    },\n  ]);\n\n  // 👀 Access the results of the Task\n  console.log(res[0].TransformedMessage);\n  console.log(res[1].TransformedMessage);\n  \n\n  // > Bulk Run Tasks from within a Task\n  const parent = hatchet.task({\n    name: \'simple\',\n    fn: async (input: SimpleInput, ctx) => {\n      // Bulk run two tasks in parallel\n      const child = await ctx.bulkRunChildren([\n        {\n          workflow: simple,\n          input: {\n            Message: \'Hello, World!\',\n          },\n        },\n        {\n          workflow: simple,\n          input: {\n            Message: \'Hello, Moon!\',\n          },\n        },\n      ]);\n\n      return {\n        TransformedMessage: `${child[0].TransformedMessage} ${child[1].TransformedMessage}`,\n      };\n    },\n  });\n  \n}\n\nif (require.main === module) {\n  main();\n}\n',
  'source': 'out/typescript/simple/bulk.ts',
  'blocks': {
    'bulk_run_a_task': {
      'start': 6,
      'stop': 17
    },
    'bulk_run_tasks_from_within_a_task': {
      'start': 20,
      'stop': 43
    }
  },
  'highlights': {}
};  // Then replace double quotes with single quotes

export default snippet;
