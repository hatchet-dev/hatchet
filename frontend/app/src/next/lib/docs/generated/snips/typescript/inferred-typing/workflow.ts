import { Snippet } from '@/next/lib/docs/generated/snips/types';

const snippet: Snippet = {
  language: 'typescript ',
  content:
    "import { hatchet } from '../hatchet-client';\n\ntype SimpleInput = {\n  Message: string;\n};\n\ntype SimpleOutput = {\n  TransformedMessage: string;\n};\n\nexport const declaredType = hatchet.task<SimpleInput, SimpleOutput>({\n  name: 'declared-type',\n  fn: (input) => {\n    return {\n      TransformedMessage: input.Message.toLowerCase(),\n    };\n  },\n});\n\nexport const inferredType = hatchet.task({\n  name: 'inferred-type',\n  fn: (input: SimpleInput) => {\n    return {\n      TransformedMessage: input.Message.toUpperCase(),\n    };\n  },\n});\n\nexport const inferredTypeDurable = hatchet.durableTask({\n  name: 'inferred-type-durable',\n  fn: async (input: SimpleInput, ctx) => {\n    // await ctx.sleepFor('5s');\n\n    return {\n      TransformedMessage: input.Message.toUpperCase(),\n    };\n  },\n});\n\nexport const crazyWorkflow = hatchet.workflow<any, any>({\n  name: 'crazy-workflow',\n});\n\nconst step1 = crazyWorkflow.task(declaredType);\n// crazyWorkflow.task(inferredTypeDurable);\n\ncrazyWorkflow.task({\n  parents: [step1],\n  ...inferredType.taskDef,\n});\n",
  source: 'out/typescript/inferred-typing/workflow.ts',
  blocks: {},
  highlights: {},
};

export default snippet;
