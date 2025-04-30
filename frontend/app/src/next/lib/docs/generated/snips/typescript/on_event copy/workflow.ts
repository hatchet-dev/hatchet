import { Snippet } from '@/next/lib/docs/generated/snips/types';

const snippet: Snippet = {
  'language': 'typescript ',
  'content': 'import { hatchet } from \'../hatchet-client\';\n\nexport type Input = {\n  Message: string;\n};\n\ntype LowerOutput = {\n  lower: {\n    TransformedMessage: string;\n  };\n};\n\n// ❓ Run workflow on event\nexport const lower = hatchet.workflow<Input, LowerOutput>({\n  name: \'lower\',\n  on: {\n    // 👀 Declare the event that will trigger the workflow\n    event: \'simple-event:create\',\n  },\n});\n\nlower.task({\n  name: \'lower\',\n  fn: (input) => {\n    return {\n      TransformedMessage: input.Message.toLowerCase(),\n    };\n  },\n});\n\ntype UpperOutput = {\n  upper: {\n    TransformedMessage: string;\n  };\n};\n\nexport const upper = hatchet.workflow<Input, UpperOutput>({\n  name: \'upper\',\n  on: {\n    event: \'simple-event:create\',\n  },\n});\n\nupper.task({\n  name: \'upper\',\n  fn: (input) => {\n    return {\n      TransformedMessage: input.Message.toUpperCase(),\n    };\n  },\n});\n',
  'source': 'out/typescript/on_event copy/workflow.ts',
  'blocks': {},
  'highlights': {}
};  // Then replace double quotes with single quotes

export default snippet;
