import { Snippet } from '@/next/lib/docs/generated/snips/types';

const snippet: Snippet = {
  'language': 'typescript ',
  'content': 'import { hatchet } from \'../hatchet-client\';\n\ntype DagInput = {\n  Message: string;\n};\n\ntype DagOutput = {\n  reverse: {\n    Original: string;\n    Transformed: string;\n  };\n};\n\n// > Declaring a DAG Workflow\n// First, we declare the workflow\nexport const dag = hatchet.workflow<DagInput, DagOutput>({\n  name: \'simple\',\n});\n\n// Next, we declare the tasks bound to the workflow\nconst toLower = dag.task({\n  name: \'to-lower\',\n  fn: (input) => {\n    return {\n      TransformedMessage: input.Message.toLowerCase(),\n    };\n  },\n});\n\n// Next, we declare the tasks bound to the workflow\ndag.task({\n  name: \'reverse\',\n  parents: [toLower],\n  fn: async (input, ctx) => {\n    const lower = await ctx.parentOutput(toLower);\n    return {\n      Original: input.Message,\n      Transformed: lower.TransformedMessage.split(\'\').reverse().join(\'\'),\n    };\n  },\n});\n\n',
  'source': 'out/typescript/dag/workflow.ts',
  'blocks': {
    'declaring_a_dag_workflow': {
      'start': 15,
      'stop': 41
    }
  },
  'highlights': {}
};  // Then replace double quotes with single quotes

export default snippet;
