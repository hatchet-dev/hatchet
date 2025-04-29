import { Snippet } from '@/next/lib/docs/generated/snips/types';

const snippet: Snippet = {
  'language': 'typescript ',
  'content': '// ❓ Declaring a Task\nimport { hatchet } from \'../hatchet-client\';\n\n// (optional) Define the input type for the workflow\nexport type ChildInput = {\n  Message: string;\n};\n\nexport type ParentInput = {\n  Message: string;\n};\n\nexport const child = hatchet.task({\n  name: \'child\',\n  fn: (input: ChildInput) => {\n    const largePayload = new Array(1024 * 1024).fill(\'a\').join(\'\');\n\n    return {\n      TransformedMessage: largePayload,\n    };\n  },\n});\n\nexport const parent = hatchet.task({\n  name: \'parent\',\n  timeout: \'10m\',\n  fn: async (input: ParentInput, ctx) => {\n    // lets generate large payload 1 mb\n    const largePayload = new Array(1024 * 1024).fill(\'a\').join(\'\');\n\n    // Send the large payload 100 times\n    const num = 1000;\n\n    const children = [];\n    for (let i = 0; i < num; i += 1) {\n      children.push({\n        workflow: child,\n        input: {\n          Message: `Iteration ${i + 1}: ${largePayload}`,\n        },\n      });\n    }\n\n    await ctx.bulkRunNoWaitChildren(children);\n\n    return {\n      TransformedMessage: \'done\',\n    };\n  },\n});\n\n\n// see ./worker.ts and ./run.ts for how to run the workflow\n',
  'source': 'out/typescript/high-memory/workflow-with-child.ts',
  'blocks': {},
  'highlights': {}
};  // Then replace double quotes with single quotes

export default snippet;
