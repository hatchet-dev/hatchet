import { Snippet } from '@/lib/generated/snips/types';

const snippet: Snippet = {
  "language": "typescript ",
  "content": "import { hatchet } from '../../hatchet-client';\n\ntype SimpleInput = {\n  Message: string;\n};\n\ntype SimpleOutput = {\n  TransformedMessage: string;\n};\n\nexport const firstTask = hatchet.task({\n  name: 'first-task',\n  fn: (input: SimpleInput, ctx): SimpleOutput => {\n    return {\n      TransformedMessage: input.Message.toLowerCase(),\n    };\n  },\n});\n",
  "source": "out/typescript/quickstart/workflows/first-task.ts",
  "blocks": {},
  "highlights": {}
};

export default snippet;
