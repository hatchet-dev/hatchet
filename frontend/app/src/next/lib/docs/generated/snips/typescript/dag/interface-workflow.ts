import { Snippet } from '@/next/lib/docs/generated/snips/types';

const snippet: Snippet = {
  language: 'typescript ',
  content:
    "import { WorkflowInputType, WorkflowOutputType } from '@hatchet-dev/typescript-sdk/v1';\nimport { hatchet } from '../hatchet-client';\n\ninterface DagInput extends WorkflowInputType {\n  Message: string;\n}\n\ninterface DagOutput extends WorkflowOutputType {\n  reverse: {\n    Original: string;\n    Transformed: string;\n  };\n}\n\n// > Declaring a DAG Workflow\n// First, we declare the workflow\nexport const dag = hatchet.workflow<DagInput, DagOutput>({\n  name: 'simple',\n});\n\nconst reverse = dag.task({\n  name: 'reverse',\n  fn: (input) => {\n    return {\n      Original: input.Message,\n      Transformed: input.Message.split('').reverse().join(''),\n    };\n  },\n});\n\ndag.task({\n  name: 'to-lower',\n  parents: [reverse],\n  fn: async (input, ctx) => {\n    const r = await ctx.parentOutput(reverse);\n\n    return {\n      reverse: {\n        Original: r.Transformed,\n        Transformed: r.Transformed.toLowerCase(),\n      },\n    };\n  },\n});\n",
  source: 'out/typescript/dag/interface-workflow.ts',
  blocks: {},
  highlights: {},
}; // Then replace double quotes with single quotes

export default snippet;
