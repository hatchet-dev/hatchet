import { Snippet } from '@/next/lib/docs/generated/snips/types';

const snippet: Snippet = {
  language: 'typescript ',
  content:
    "// > Declaring a Task\nimport { hatchet } from '../hatchet-client';\n\n// (optional) Define the input type for the workflow\nexport type ChildInput = {\n  Message: string;\n};\n\nexport type ParentInput = {\n  Message: string;\n};\n\nexport const child = hatchet.workflow<ChildInput>({\n  name: 'child',\n});\n\nexport const child1 = child.task({\n  name: 'child1',\n  fn: (input: ChildInput, ctx) => {\n    ctx.log('hello from the child1');\n    return {\n      TransformedMessage: input.Message.toLowerCase(),\n    };\n  },\n});\n\nexport const child2 = child.task({\n  name: 'child2',\n  fn: (input: ChildInput, ctx) => {\n    ctx.log('hello from the child2');\n    return {\n      TransformedMessage: input.Message.toLowerCase(),\n    };\n  },\n});\n\nexport const parent = hatchet.task({\n  name: 'parent',\n  fn: async (input: ParentInput, ctx) => {\n    const c = await ctx.runChild(child, {\n      Message: input.Message,\n    });\n\n    return {\n      TransformedMessage: 'not implemented',\n    };\n  },\n});\n\n\n\n// see ./worker.ts and ./run.ts for how to run the workflow\n",
  source: 'out/typescript/simple/workflow-with-child.ts',
  blocks: {
    declaring_a_task: {
      start: 2,
      stop: 49,
    },
  },
  highlights: {},
}; // Then replace double quotes with single quotes

export default snippet;
