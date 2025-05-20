import { Snippet } from '@/next/lib/docs/generated/snips/types';

const snippet: Snippet = {
  language: 'typescript ',
  content:
    "import { Priority } from '@hatchet-dev/typescript-sdk/v1';\nimport { hatchet } from '../hatchet-client';\n\n// > Simple Task Priority\nexport const priority = hatchet.task({\n  name: 'priority',\n  defaultPriority: Priority.MEDIUM,\n  fn: async (_, ctx) => {\n    return {\n      priority: ctx.priority(),\n    };\n  },\n});\n\n// > Task Priority in a Workflow\nexport const priorityWf = hatchet.workflow({\n  name: 'priorityWf',\n  defaultPriority: Priority.LOW,\n});\n\npriorityWf.task({\n  name: 'child-medium',\n  fn: async (_, ctx) => {\n    return {\n      priority: ctx.priority(),\n    };\n  },\n});\n\npriorityWf.task({\n  name: 'child-high',\n  // will inherit the default priority from the workflow\n  fn: async (_, ctx) => {\n    return {\n      priority: ctx.priority(),\n    };\n  },\n});\n\nexport const priorityTasks = [priority, priorityWf];\n",
  source: 'out/typescript/priority/workflow.ts',
  blocks: {
    simple_task_priority: {
      start: 5,
      stop: 13,
    },
    task_priority_in_a_workflow: {
      start: 16,
      stop: 19,
    },
  },
  highlights: {},
};

export default snippet;
