import { Snippet } from '@/lib/generated/snips/types';

const snippet: Snippet = {
  "language": "typescript ",
  "content": "import { hatchet } from '../hatchet-client';\n\nexport type Input = {\n  Message: string;\n  ShouldSkip: boolean;\n};\n\nexport const SIMPLE_EVENT = 'simple-event:create';\n\ntype LowerOutput = {\n  lower: {\n    TransformedMessage: string;\n  };\n};\n\n// > Run workflow on event\nexport const lower = hatchet.workflow<Input, LowerOutput>({\n  name: 'lower',\n  // 👀 Declare the event that will trigger the workflow\n  onEvents: ['simple-event:create'],\n});\n\n// > Workflow with filter\nexport const lowerWithFilter = hatchet.workflow<Input, LowerOutput>({\n  name: 'lower',\n  // 👀 Declare the event that will trigger the workflow\n  onEvents: ['simple-event:create'],\n  defaultFilters: [\n    {\n      expression: 'true',\n      scope: 'example-scope',\n      payload: {\n        mainCharacter: 'Anna',\n        supportingCharacter: 'Stiva',\n        location: 'Moscow',\n      },\n    },\n  ],\n});\n\nlower.task({\n  name: 'lower',\n  fn: (input) => {\n    return {\n      TransformedMessage: input.Message.toLowerCase(),\n    };\n  },\n});\n\ntype UpperOutput = {\n  upper: {\n    TransformedMessage: string;\n  };\n};\n\nexport const upper = hatchet.workflow<Input, UpperOutput>({\n  name: 'upper',\n  on: {\n    event: SIMPLE_EVENT,\n  },\n});\n\nupper.task({\n  name: 'upper',\n  fn: (input) => {\n    return {\n      TransformedMessage: input.Message.toUpperCase(),\n    };\n  },\n});\n\n// > Accessing the filter payload\nlowerWithFilter.task({\n  name: 'lowerWithFilter',\n  fn: (input, ctx) => {\n    console.log(ctx.filterPayload());\n  },\n});\n",
  "source": "out/typescript/on_event/workflow.ts",
  "blocks": {
    "run_workflow_on_event": {
      "start": 17,
      "stop": 21
    },
    "workflow_with_filter": {
      "start": 24,
      "stop": 39
    },
    "accessing_the_filter_payload": {
      "start": 73,
      "stop": 78
    }
  },
  "highlights": {}
};

export default snippet;
