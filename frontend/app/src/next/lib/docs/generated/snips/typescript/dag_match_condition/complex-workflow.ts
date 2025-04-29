import { Snippet } from '@/next/lib/docs/generated/snips/types';

const snippet: Snippet = {
  language: 'typescript ',
  content:
    "// > Create a workflow\nimport { Or, SleepCondition, UserEventCondition } from '@hatchet-dev/typescript-sdk/v1/conditions';\nimport { ParentCondition } from '@hatchet-dev/typescript-sdk/v1/conditions/parent-condition';\nimport { Context } from '@hatchet-dev/typescript-sdk/step';\nimport { hatchet } from '../hatchet-client';\n\nexport const taskConditionWorkflow = hatchet.workflow({\n  name: 'TaskConditionWorkflow',\n});\n\n// > Add base task\nconst start = taskConditionWorkflow.task({\n  name: 'start',\n  fn: () => {\n    return {\n      randomNumber: Math.floor(Math.random() * 100) + 1,\n    };\n  },\n});\n\n// > Add wait for sleep\nconst waitForSleep = taskConditionWorkflow.task({\n  name: 'waitForSleep',\n  parents: [start],\n  waitFor: [new SleepCondition('10s')],\n  fn: () => {\n    return {\n      randomNumber: Math.floor(Math.random() * 100) + 1,\n    };\n  },\n});\n\n// > Add skip on event\nconst skipOnEvent = taskConditionWorkflow.task({\n  name: 'skipOnEvent',\n  parents: [start],\n  waitFor: [new SleepCondition('10s')],\n  skipIf: [new UserEventCondition('skip_on_event:skip', 'true')],\n  fn: () => {\n    return {\n      randomNumber: Math.floor(Math.random() * 100) + 1,\n    };\n  },\n});\n\n// > Add branching\nconst leftBranch = taskConditionWorkflow.task({\n  name: 'leftBranch',\n  parents: [waitForSleep],\n  skipIf: [new ParentCondition(waitForSleep, 'output.randomNumber > 50')],\n  fn: () => {\n    return {\n      randomNumber: Math.floor(Math.random() * 100) + 1,\n    };\n  },\n});\n\nconst rightBranch = taskConditionWorkflow.task({\n  name: 'rightBranch',\n  parents: [waitForSleep],\n  skipIf: [new ParentCondition(waitForSleep, 'output.randomNumber <= 50')],\n  fn: () => {\n    return {\n      randomNumber: Math.floor(Math.random() * 100) + 1,\n    };\n  },\n});\n\n// > Add wait for event\nconst waitForEvent = taskConditionWorkflow.task({\n  name: 'waitForEvent',\n  parents: [start],\n  waitFor: [Or(new SleepCondition('1m'), new UserEventCondition('wait_for_event:start', 'true'))],\n  fn: () => {\n    return {\n      randomNumber: Math.floor(Math.random() * 100) + 1,\n    };\n  },\n});\n\n// > Add sum\ntaskConditionWorkflow.task({\n  name: 'sum',\n  parents: [start, waitForSleep, waitForEvent, skipOnEvent, leftBranch, rightBranch],\n  fn: async (_, ctx: Context<any, any>) => {\n    const one = (await ctx.parentOutput(start)).randomNumber;\n    const two = (await ctx.parentOutput(waitForEvent)).randomNumber;\n    const three = (await ctx.parentOutput(waitForSleep)).randomNumber;\n    const four = (await ctx.parentOutput(skipOnEvent))?.randomNumber || 0;\n    const five = (await ctx.parentOutput(leftBranch))?.randomNumber || 0;\n    const six = (await ctx.parentOutput(rightBranch))?.randomNumber || 0;\n\n    return {\n      sum: one + two + three + four + five + six,\n    };\n  },\n});\n",
  source: 'out/typescript/dag_match_condition/complex-workflow.ts',
  blocks: {
    create_a_workflow: {
      start: 2,
      stop: 9,
    },
    add_base_task: {
      start: 12,
      stop: 19,
    },
    add_wait_for_sleep: {
      start: 22,
      stop: 31,
    },
    add_skip_on_event: {
      start: 34,
      stop: 44,
    },
    add_branching: {
      start: 47,
      stop: 67,
    },
    add_wait_for_event: {
      start: 70,
      stop: 79,
    },
    add_sum: {
      start: 82,
      stop: 97,
    },
  },
  highlights: {},
}; // Then replace double quotes with single quotes

export default snippet;
