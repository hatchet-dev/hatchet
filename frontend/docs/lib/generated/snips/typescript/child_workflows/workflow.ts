import { Snippet } from '@/lib/generated/snips/types';

const snippet: Snippet = {
  "language": "typescript ",
  "content": "// > Declaring a Child\nimport { hatchet } from '../hatchet-client';\n\ntype ChildInput = {\n  N: number;\n};\n\nexport const child = hatchet.task({\n  name: 'child',\n  fn: (input: ChildInput) => {\n    return {\n      Value: input.N,\n    };\n  },\n});\n\n// > Declaring a Parent\n\ntype ParentInput = {\n  N: number;\n};\n\nexport const parent = hatchet.task({\n  name: 'parent',\n  fn: async (input: ParentInput, ctx) => {\n    const n = input.N;\n    const promises = [];\n\n    for (let i = 0; i < n; i++) {\n      promises.push(ctx.runChild(child, { N: i }));\n    }\n\n    const childRes = await Promise.all(promises);\n    const sum = childRes.reduce((acc, curr) => acc + curr.Value, 0);\n\n    return {\n      Result: sum,\n    };\n  },\n});\n",
  "source": "out/typescript/child_workflows/workflow.ts",
  "blocks": {
    "declaring_a_child": {
      "start": 2,
      "stop": 15
    },
    "declaring_a_parent": {
      "start": 18,
      "stop": 40
    }
  },
  "highlights": {}
};

export default snippet;
