// This file is auto-generated. Do not edit directly.

// Types for snippets
type Snippet = {
  content: string;
  language: string;
  source: string;
  highlights?: {
    [key: string]: {
      lines: number[];
      strings: string[];
    };
  };
};

type Snippets = {
  [key: string]: Snippet;
};

// Snippet contents
export const snippets: Snippets = {
  L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3R5cGVzY3JpcHQvY2FuY2VsbGF0aW9ucy9ydW4udHM_:
    {
      content:
        "\n// ❓ Running a Task with Results\nimport sleep from '@hatchet-dev/typescript-sdk/util/sleep';\nimport { cancellation } from './workflow';\nimport { hatchet } from '../hatchet-client';\n// ...\nasync function main() {\n  const run = cancellation.runNoWait({});\n  const run1 = cancellation.runNoWait({});\n\n  await sleep(1000);\n\n  await run.cancel();\n\n  const res = await run.output;\n  const res1 = await run1.output;\n\n  console.log('canceled', res);\n  console.log('completed', res1);\n\n  await sleep(1000);\n\n  await run.replay();\n\n  const resReplay = await run.output;\n\n  console.log(resReplay);\n\n  const run2 = cancellation.runNoWait({}, { additionalMetadata: { test: 'abc' } });\n  const run4 = cancellation.runNoWait({}, { additionalMetadata: { test: 'test' } });\n\n  await sleep(1000);\n\n  await hatchet.runs.cancel({\n    filters: {\n      since: new Date(Date.now() - 60 * 60),\n      additionalMetadata: { test: 'test' },\n    },\n  });\n\n  const res3 = await Promise.all([run2.output, run4.output]);\n  console.log(res3);\n\n}\n\nif (require.main === module) {\n  main()\n    .catch(console.error)\n    .finally(() => process.exit(0));\n}\n",
      language: 'ts',
      source: 'examples/typescript/cancellations/run.ts',
      highlights: {},
    },
  L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3R5cGVzY3JpcHQvY2FuY2VsbGF0aW9ucy93b3JrZXIudHM_:
    {
      content:
        "// ❓ Declaring a Worker\nimport { hatchet } from '../hatchet-client';\nimport { cancellation } from './workflow';\n\nasync function main() {\n  const worker = await hatchet.worker('cancellation-worker', {\n    // 👀 Declare the workflows that the worker can execute\n    workflows: [cancellation],\n    // 👀 Declare the number of concurrent task runs the worker can accept\n    slots: 100,\n  });\n\n  await worker.start();\n}\n\nif (require.main === module) {\n  main();\n}\n",
      language: 'ts',
      source: 'examples/typescript/cancellations/worker.ts',
      highlights: {},
    },
  L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3R5cGVzY3JpcHQvY2FuY2VsbGF0aW9ucy93b3JrZmxvdy50cw__:
    {
      content:
        "import sleep from '@hatchet-dev/typescript-sdk/util/sleep';\nimport axios from 'axios';\nimport { hatchet } from '../hatchet-client';\n\n// ❓ Declaring a Task\nexport const cancellation = hatchet.task({\n  name: 'cancellation',\n  fn: async (_, { cancelled }) => {\n    await sleep(10 * 1000);\n\n    if (cancelled) {\n      throw new Error('Task was cancelled');\n    }\n\n    return {\n      Completed: true,\n    };\n  },\n});\n\n// ❓ Abort Signal\nexport const abortSignal = hatchet.task({\n  name: 'abort-signal',\n  fn: async (_, { controller }) => {\n    try {\n      const response = await axios.get('https://api.example.com/data', {\n        signal: controller.signal,\n      });\n      // Handle the response\n    } catch (error) {\n      if (axios.isCancel(error)) {\n        // Request was canceled\n        console.log('Request canceled');\n      } else {\n        // Handle other errors\n      }\n    }\n  },\n});\n\n// see ./worker.ts and ./run.ts for how to run the workflow\n",
      language: 'ts',
      source: 'examples/typescript/cancellations/workflow.ts',
      highlights: {},
    },
  L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3R5cGVzY3JpcHQvY2hpbGRfd29ya2Zsb3dzL3J1bi50cw__:
    {
      content:
        "import { parent } from './workflow';\n\nasync function main() {\n  const res = await parent.run({\n    N: 10,\n  });\n  console.log(res.Result);\n}\n\nif (require.main === module) {\n  main()\n    .then(() => process.exit(0))\n    .catch((error) => {\n      console.error('Error:', error);\n      process.exit(1);\n    });\n}\n",
      language: 'ts',
      source: 'examples/typescript/child_workflows/run.ts',
      highlights: {},
    },
  L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3R5cGVzY3JpcHQvY2hpbGRfd29ya2Zsb3dzL3dvcmtlci50cw__:
    {
      content:
        "import { hatchet } from '../hatchet-client';\nimport { parent, child } from './workflow';\n\nasync function main() {\n  const worker = await hatchet.worker('child-workflow-worker', {\n    workflows: [parent, child],\n    slots: 100,\n  });\n\n  await worker.start();\n}\n\nif (require.main === module) {\n  main();\n}\n",
      language: 'ts',
      source: 'examples/typescript/child_workflows/worker.ts',
      highlights: {},
    },
  L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3R5cGVzY3JpcHQvY2hpbGRfd29ya2Zsb3dzL3dvcmtmbG93LnRz:
    {
      content:
        "\n// ❓ Declaring a Child\nimport { hatchet } from '../hatchet-client';\n\ntype ChildInput = {\n  N: number;\n};\n\nexport const child = hatchet.task({\n  name: 'child',\n  fn: (input: ChildInput) => {\n    return {\n      Value: input.N,\n    };\n  },\n});\n\n// ❓ Declaring a Parent\n\ntype ParentInput = {\n  N: number;\n};\n\nexport const parent = hatchet.task({\n  name: 'parent',\n  fn: async (input: ParentInput, ctx) => {\n    const n = input.N;\n    const promises = [];\n\n    for (let i = 0; i < n; i++) {\n      promises.push(ctx.runChild(child, { N: i }));\n    }\n\n    const childRes = await Promise.all(promises);\n    const sum = childRes.reduce((acc, curr) => acc + curr.Value, 0);\n\n    return {\n      Result: sum,\n    };\n  },\n});\n",
      language: 'ts',
      source: 'examples/typescript/child_workflows/workflow.ts',
      highlights: {},
    },
  L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3R5cGVzY3JpcHQvY29uY3VycmVuY3ktcnIvbG9hZC50cw__:
    {
      content:
        "\nimport { hatchet } from '../hatchet-client';\nimport { simpleConcurrency } from './workflow';\n\nfunction generateRandomString(length: number): string {\n  const characters = 'ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789';\n  let result = '';\n  for (let i = 0; i < length; i++) {\n    result += characters.charAt(Math.floor(Math.random() * characters.length));\n  }\n  return result;\n}\n\nasync function main() {\n  const groupCount = 2;\n  const runsPerGroup = 20_000;\n  const BATCH_SIZE = 400;\n\n  const workflowRuns = [];\n  for (let i = 0; i < groupCount; i++) {\n    for (let j = 0; j < runsPerGroup; j++) {\n      workflowRuns.push({\n        workflowName: simpleConcurrency.definition.name,\n        input: {\n          Message: generateRandomString(10),\n          GroupKey: `group-${i}`,\n        },\n      });\n    }\n  }\n\n  // Shuffle the workflow runs array\n  for (let i = workflowRuns.length - 1; i > 0; i--) {\n    const j = Math.floor(Math.random() * (i + 1));\n    [workflowRuns[i], workflowRuns[j]] = [workflowRuns[j], workflowRuns[i]];\n  }\n\n  // Process workflows in batches\n  for (let i = 0; i < workflowRuns.length; i += BATCH_SIZE) {\n    const batch = workflowRuns.slice(i, i + BATCH_SIZE);\n    await hatchet.admin.runWorkflows(batch);\n  }\n}\n\nif (require.main === module) {\n  main().then(() => process.exit(0));\n}\n",
      language: 'ts',
      source: 'examples/typescript/concurrency-rr/load.ts',
      highlights: {},
    },
  L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3R5cGVzY3JpcHQvY29uY3VycmVuY3ktcnIvcnVuLnRz:
    {
      content:
        "import { simpleConcurrency } from './workflow';\n\nasync function main() {\n  const res = await simpleConcurrency.run([\n    {\n      Message: 'Hello World',\n      GroupKey: 'A',\n    },\n    {\n      Message: 'Goodbye Moon',\n      GroupKey: 'A',\n    },\n    {\n      Message: 'Hello World B',\n      GroupKey: 'B',\n    },\n  ]);\n  console.log(res[0]['to-lower'].TransformedMessage);\n  console.log(res[1]['to-lower'].TransformedMessage);\n  console.log(res[2]['to-lower'].TransformedMessage);\n}\n\nif (require.main === module) {\n  main().then(() => process.exit(0));\n}\n",
      language: 'ts',
      source: 'examples/typescript/concurrency-rr/run.ts',
      highlights: {},
    },
  L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3R5cGVzY3JpcHQvY29uY3VycmVuY3ktcnIvd29ya2VyLnRz:
    {
      content:
        "import { hatchet } from '../hatchet-client';\nimport { simpleConcurrency } from './workflow';\n\nasync function main() {\n  const worker = await hatchet.worker('simple-concurrency-worker', {\n    workflows: [simpleConcurrency],\n  });\n\n  await worker.start();\n}\n\nif (require.main === module) {\n  main();\n}\n",
      language: 'ts',
      source: 'examples/typescript/concurrency-rr/worker.ts',
      highlights: {},
    },
  L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3R5cGVzY3JpcHQvY29uY3VycmVuY3ktcnIvd29ya2Zsb3cudHM_:
    {
      content:
        "import { ConcurrencyLimitStrategy } from '@hatchet-dev/typescript-sdk/workflow';\nimport { hatchet } from '../hatchet-client';\n\ntype SimpleInput = {\n  Message: string;\n  GroupKey: string;\n};\n\ntype SimpleOutput = {\n  'to-lower': {\n    TransformedMessage: string;\n  };\n};\n\nconst sleep = (ms: number) =>\n  new Promise((resolve) => {\n    setTimeout(resolve, ms);\n  });\n\n// ❓ Concurrency Strategy With Key\nexport const simpleConcurrency = hatchet.workflow<SimpleInput, SimpleOutput>({\n  name: 'simple-concurrency',\n  concurrency: {\n    maxRuns: 1,\n    limitStrategy: ConcurrencyLimitStrategy.GROUP_ROUND_ROBIN,\n    expression: 'input.GroupKey',\n  },\n});\n\nsimpleConcurrency.task({\n  name: 'to-lower',\n  fn: async (input) => {\n    await sleep(Math.floor(Math.random() * (1000 - 200 + 1)) + 200);\n    return {\n      TransformedMessage: input.Message.toLowerCase(),\n    };\n  },\n});\n\n// ❓ Multiple Concurrency Keys\nexport const multipleConcurrencyKeys = hatchet.workflow<SimpleInput, SimpleOutput>({\n  name: 'simple-concurrency',\n  concurrency: [\n    {\n      maxRuns: 1,\n      limitStrategy: ConcurrencyLimitStrategy.GROUP_ROUND_ROBIN,\n      expression: 'input.Tier',\n    },\n    {\n      maxRuns: 1,\n      limitStrategy: ConcurrencyLimitStrategy.GROUP_ROUND_ROBIN,\n      expression: 'input.Account',\n    },\n  ],\n});\n\nmultipleConcurrencyKeys.task({\n  name: 'to-lower',\n  fn: async (input) => {\n    await sleep(Math.floor(Math.random() * (1000 - 200 + 1)) + 200);\n    return {\n      TransformedMessage: input.Message.toLowerCase(),\n    };\n  },\n});\n",
      language: 'ts',
      source: 'examples/typescript/concurrency-rr/workflow.ts',
      highlights: {},
    },
  L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3R5cGVzY3JpcHQvZGFnL2ludGVyZmFjZS13b3JrZmxvdy50cw__:
    {
      content:
        "import { WorkflowInputType, WorkflowOutputType } from '@hatchet-dev/typescript-sdk/v1';\nimport { hatchet } from '../hatchet-client';\n\ninterface DagInput extends WorkflowInputType {\n  Message: string;\n}\n\ninterface DagOutput extends WorkflowOutputType {\n  reverse: {\n    Original: string;\n    Transformed: string;\n  };\n}\n\n// ❓ Declaring a DAG Workflow\n// First, we declare the workflow\nexport const dag = hatchet.workflow<DagInput, DagOutput>({\n  name: 'simple',\n});\n\nconst reverse = dag.task({\n  name: 'reverse',\n  fn: (input) => {\n    return {\n      Original: input.Message,\n      Transformed: input.Message.split('').reverse().join(''),\n    };\n  },\n});\n\ndag.task({\n  name: 'to-lower',\n  parents: [reverse],\n  fn: async (input, ctx) => {\n    const r = await ctx.parentOutput(reverse);\n\n    return {\n      reverse: {\n        Original: r.Transformed,\n        Transformed: r.Transformed.toLowerCase(),\n      },\n    };\n  },\n});\n",
      language: 'ts',
      source: 'examples/typescript/dag/interface-workflow.ts',
      highlights: {},
    },
  L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3R5cGVzY3JpcHQvZGFnL3J1bi50cw__:
    {
      content:
        "import { dag } from './workflow';\n\nasync function main() {\n  const res = await dag.run({\n    Message: 'hello world',\n  });\n  console.log(res.reverse.Transformed);\n}\n\nif (require.main === module) {\n  main();\n}\n",
      language: 'ts',
      source: 'examples/typescript/dag/run.ts',
      highlights: {},
    },
  L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3R5cGVzY3JpcHQvZGFnL3dvcmtlci50cw__:
    {
      content:
        "import { hatchet } from '../hatchet-client';\nimport { dag } from './workflow';\n\nasync function main() {\n  const worker = await hatchet.worker('dag-worker', {\n    workflows: [dag],\n  });\n\n  await worker.start();\n}\n\nif (require.main === module) {\n  main();\n}\n",
      language: 'ts',
      source: 'examples/typescript/dag/worker.ts',
      highlights: {},
    },
  L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3R5cGVzY3JpcHQvZGFnL3dvcmtmbG93LnRz:
    {
      content:
        "import { hatchet } from '../hatchet-client';\n\ntype DagInput = {\n  Message: string;\n};\n\ntype DagOutput = {\n  reverse: {\n    Original: string;\n    Transformed: string;\n  };\n};\n\n// ❓ Declaring a DAG Workflow\n// First, we declare the workflow\nexport const dag = hatchet.workflow<DagInput, DagOutput>({\n  name: 'simple',\n});\n\n// Next, we declare the tasks bound to the workflow\nconst toLower = dag.task({\n  name: 'to-lower',\n  fn: (input) => {\n    return {\n      TransformedMessage: input.Message.toLowerCase(),\n    };\n  },\n});\n\n// Next, we declare the tasks bound to the workflow\ndag.task({\n  name: 'reverse',\n  parents: [toLower],\n  fn: async (input, ctx) => {\n    const lower = await ctx.parentOutput(toLower);\n    return {\n      Original: input.Message,\n      Transformed: lower.TransformedMessage.split('').reverse().join(''),\n    };\n  },\n});\n",
      language: 'ts',
      source: 'examples/typescript/dag/workflow.ts',
      highlights: {},
    },
  L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3R5cGVzY3JpcHQvZGFnX21hdGNoX2NvbmRpdGlvbi9jb21wbGV4LXdvcmtmbG93LnRz:
    {
      content:
        "// ❓ Create a workflow\nimport { Or, SleepCondition, UserEventCondition } from '@hatchet-dev/typescript-sdk/v1/conditions';\nimport { ParentCondition } from '@hatchet-dev/typescript-sdk/v1/conditions/parent-condition';\nimport { Context } from '@hatchet-dev/typescript-sdk/step';\nimport { hatchet } from '../hatchet-client';\n\nexport const taskConditionWorkflow = hatchet.workflow({\n  name: 'TaskConditionWorkflow',\n});\n\n// ❓ Add base task\nconst start = taskConditionWorkflow.task({\n  name: 'start',\n  fn: () => {\n    return {\n      randomNumber: Math.floor(Math.random() * 100) + 1,\n    };\n  },\n});\n\n// ❓ Add wait for sleep\nconst waitForSleep = taskConditionWorkflow.task({\n  name: 'waitForSleep',\n  parents: [start],\n  waitFor: [new SleepCondition('10s')],\n  fn: () => {\n    return {\n      randomNumber: Math.floor(Math.random() * 100) + 1,\n    };\n  },\n});\n\n// ❓ Add skip on event\nconst skipOnEvent = taskConditionWorkflow.task({\n  name: 'skipOnEvent',\n  parents: [start],\n  waitFor: [new SleepCondition('10s')],\n  skipIf: [new UserEventCondition('skip_on_event:skip', 'true')],\n  fn: () => {\n    return {\n      randomNumber: Math.floor(Math.random() * 100) + 1,\n    };\n  },\n});\n\n// ❓ Add branching\nconst leftBranch = taskConditionWorkflow.task({\n  name: 'leftBranch',\n  parents: [waitForSleep],\n  skipIf: [new ParentCondition(waitForSleep, 'output.randomNumber > 50')],\n  fn: () => {\n    return {\n      randomNumber: Math.floor(Math.random() * 100) + 1,\n    };\n  },\n});\n\nconst rightBranch = taskConditionWorkflow.task({\n  name: 'rightBranch',\n  parents: [waitForSleep],\n  skipIf: [new ParentCondition(waitForSleep, 'output.randomNumber <= 50')],\n  fn: () => {\n    return {\n      randomNumber: Math.floor(Math.random() * 100) + 1,\n    };\n  },\n});\n\n// ❓ Add wait for event\nconst waitForEvent = taskConditionWorkflow.task({\n  name: 'waitForEvent',\n  parents: [start],\n  waitFor: [Or(new SleepCondition('1m'), new UserEventCondition('wait_for_event:start', 'true'))],\n  fn: () => {\n    return {\n      randomNumber: Math.floor(Math.random() * 100) + 1,\n    };\n  },\n});\n\n// ❓ Add sum\ntaskConditionWorkflow.task({\n  name: 'sum',\n  parents: [start, waitForSleep, waitForEvent, skipOnEvent, leftBranch, rightBranch],\n  fn: async (_, ctx: Context<any, any>) => {\n    const one = (await ctx.parentOutput(start)).randomNumber;\n    const two = (await ctx.parentOutput(waitForEvent)).randomNumber;\n    const three = (await ctx.parentOutput(waitForSleep)).randomNumber;\n    const four = (await ctx.parentOutput(skipOnEvent))?.randomNumber || 0;\n    const five = (await ctx.parentOutput(leftBranch))?.randomNumber || 0;\n    const six = (await ctx.parentOutput(rightBranch))?.randomNumber || 0;\n\n    return {\n      sum: one + two + three + four + five + six,\n    };\n  },\n});\n",
      language: 'ts',
      source: 'examples/typescript/dag_match_condition/complex-workflow.ts',
      highlights: {},
    },
  L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3R5cGVzY3JpcHQvZGFnX21hdGNoX2NvbmRpdGlvbi9ldmVudC50cw__:
    {
      content:
        "import { hatchet } from '../hatchet-client';\n\nasync function main() {\n  const event = await hatchet.events.push('user:event', {\n    Data: { Hello: 'World' },\n  });\n}\n\nif (require.main === module) {\n  main()\n    .then(() => process.exit(0))\n    .catch((error) => {\n      console.error('Error:', error);\n      process.exit(1);\n    });\n}\n",
      language: 'ts',
      source: 'examples/typescript/dag_match_condition/event.ts',
      highlights: {},
    },
  L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3R5cGVzY3JpcHQvZGFnX21hdGNoX2NvbmRpdGlvbi9ydW4udHM_:
    {
      content:
        "\nimport { dagWithConditions } from './workflow';\n\nasync function main() {\n  const res = await dagWithConditions.run({});\n\n  console.log(res['first-task'].Completed);\n  console.log(res['second-task'].Completed);\n}\n\nif (require.main === module) {\n  main()\n    .catch(console.error)\n    .finally(() => process.exit(0));\n}\n",
      language: 'ts',
      source: 'examples/typescript/dag_match_condition/run.ts',
      highlights: {},
    },
  L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3R5cGVzY3JpcHQvZGFnX21hdGNoX2NvbmRpdGlvbi93b3JrZXIudHM_:
    {
      content:
        "import { hatchet } from '../hatchet-client';\nimport { dagWithConditions } from './workflow';\n\nasync function main() {\n  const worker = await hatchet.worker('dag-worker', {\n    workflows: [dagWithConditions],\n  });\n\n  await worker.start();\n}\n\nif (require.main === module) {\n  main();\n}\n",
      language: 'ts',
      source: 'examples/typescript/dag_match_condition/worker.ts',
      highlights: {},
    },
  L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3R5cGVzY3JpcHQvZGFnX21hdGNoX2NvbmRpdGlvbi93b3JrZmxvdy50cw__:
    {
      content:
        "import sleep from '@hatchet-dev/typescript-sdk/util/sleep';\nimport { Or } from '@hatchet-dev/typescript-sdk/v1/conditions';\nimport { hatchet } from '../hatchet-client';\n\ntype DagInput = {};\n\ntype DagOutput = {\n  'first-task': {\n    Completed: boolean;\n  };\n  'second-task': {\n    Completed: boolean;\n  };\n};\n\nexport const dagWithConditions = hatchet.workflow<DagInput, DagOutput>({\n  name: 'simple',\n});\n\nconst firstTask = dagWithConditions.task({\n  name: 'first-task',\n  fn: async () => {\n    await sleep(2000);\n    return {\n      Completed: true,\n    };\n  },\n});\n\ndagWithConditions.task({\n  name: 'second-task',\n  parents: [firstTask],\n  waitFor: Or({ eventKey: 'user:event' }, { sleepFor: '10s' }),\n  fn: async (_, ctx) => {\n    console.log('triggered by condition', ctx.triggers());\n\n    return {\n      Completed: true,\n    };\n  },\n});\n",
      language: 'ts',
      source: 'examples/typescript/dag_match_condition/workflow.ts',
      highlights: {},
    },
  L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3R5cGVzY3JpcHQvZGVlcC9ydW4udHM_:
    {
      content:
        "import { parent } from './workflow';\n\nasync function main() {\n  const res = await parent.run({\n    Message: 'hello',\n    N: 5,\n  });\n  console.log(res.parent.Sum);\n}\n\nif (require.main === module) {\n  main()\n    .catch(console.error)\n    .finally(() => process.exit(0));\n}\n",
      language: 'ts',
      source: 'examples/typescript/deep/run.ts',
      highlights: {},
    },
  L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3R5cGVzY3JpcHQvZGVlcC93b3JrZXIudHM_:
    {
      content:
        "import { hatchet } from '../hatchet-client';\nimport { parent, child1, child2, child3, child4, child5 } from './workflow';\n\nasync function main() {\n  const worker = await hatchet.worker('simple-worker', {\n    workflows: [parent, child1, child2, child3, child4, child5],\n    slots: 5000,\n  });\n\n  await worker.start();\n}\n\nif (require.main === module) {\n  main();\n}\n",
      language: 'ts',
      source: 'examples/typescript/deep/worker.ts',
      highlights: {},
    },
  L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3R5cGVzY3JpcHQvZGVlcC93b3JrZmxvdy50cw__:
    {
      content:
        "import sleep from '@hatchet-dev/typescript-sdk/util/sleep';\nimport { hatchet } from '../hatchet-client';\n\ntype SimpleInput = {\n  Message: string;\n  N: number;\n};\n\ntype Output = {\n  transformer: {\n    Sum: number;\n  };\n};\n\nexport const child1 = hatchet.workflow<SimpleInput, Output>({\n  name: 'child1',\n});\n\nchild1.task({\n  name: 'transformer',\n  fn: () => {\n    sleep(15);\n    return {\n      Sum: 1,\n    };\n  },\n});\n\nexport const child2 = hatchet.workflow<SimpleInput, Output>({\n  name: 'child2',\n});\n\nchild2.task({\n  name: 'transformer',\n  fn: async (input, ctx) => {\n    const count = input.N;\n    const promises = Array(count)\n      .fill(null)\n      .map(() => ({ workflow: child1, input }));\n\n    const results = await ctx.bulkRunChildren(promises);\n\n    sleep(15);\n    return {\n      Sum: results.reduce((acc, r) => acc + r.transformer.Sum, 0),\n    };\n  },\n});\n\nexport const child3 = hatchet.workflow<SimpleInput, Output>({\n  name: 'child3',\n});\n\nchild3.task({\n  name: 'transformer',\n  fn: async (input, ctx) => {\n    const count = input.N;\n    const promises = Array(count)\n      .fill(null)\n      .map(() => ({ workflow: child2, input }));\n\n    const results = await ctx.bulkRunChildren(promises);\n\n    return {\n      Sum: results.reduce((acc, r) => acc + r.transformer.Sum, 0),\n    };\n  },\n});\n\nexport const child4 = hatchet.workflow<SimpleInput, Output>({\n  name: 'child4',\n});\n\nchild4.task({\n  name: 'transformer',\n  fn: async (input, ctx) => {\n    const count = input.N;\n    const promises = Array(count)\n      .fill(null)\n      .map(() => ({ workflow: child3, input }));\n\n    const results = await ctx.bulkRunChildren(promises);\n\n    return {\n      Sum: results.reduce((acc, r) => acc + r.transformer.Sum, 0),\n    };\n  },\n});\n\nexport const child5 = hatchet.workflow<SimpleInput, Output>({\n  name: 'child5',\n});\n\nchild5.task({\n  name: 'transformer',\n  fn: async (input, ctx) => {\n    const count = input.N;\n    const promises = Array(count)\n      .fill(null)\n      .map(() => ({ workflow: child4, input }));\n\n    const results = await ctx.bulkRunChildren(promises);\n\n    return {\n      Sum: results.reduce((acc, r) => acc + r.transformer.Sum, 0),\n    };\n  },\n});\n\nexport const parent = hatchet.workflow<SimpleInput, { parent: Output['transformer'] }>({\n  name: 'parent',\n});\n\nparent.task({\n  name: 'parent',\n  fn: async (input, ctx) => {\n    const count = input.N; // Random number between 2-4\n    const promises = Array(count)\n      .fill(null)\n      .map(() => ({ workflow: child5, input }));\n\n    const results = await ctx.bulkRunChildren(promises);\n\n    return {\n      Sum: results.reduce((acc, r) => acc + r.transformer.Sum, 0),\n    };\n  },\n});\n",
      language: 'ts',
      source: 'examples/typescript/deep/workflow.ts',
      highlights: {},
    },
  L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3R5cGVzY3JpcHQvZHVyYWJsZS1ldmVudC9ldmVudC50cw__:
    {
      content:
        "import { hatchet } from '../hatchet-client';\n\nasync function main() {\n  const event = await hatchet.events.push('user:update', {\n    userId: '1234',\n  });\n}\n\nif (require.main === module) {\n  main()\n    .then(() => process.exit(0))\n    .catch((error) => {\n      console.error('Error:', error);\n      process.exit(1);\n    });\n}\n",
      language: 'ts',
      source: 'examples/typescript/durable-event/event.ts',
      highlights: {},
    },
  L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3R5cGVzY3JpcHQvZHVyYWJsZS1ldmVudC9ydW4udHM_:
    {
      content:
        "import { durableEvent } from './workflow';\n\nasync function main() {\n  const timeStart = Date.now();\n  const res = await durableEvent.run({});\n  const timeEnd = Date.now();\n  console.log(`Time taken: ${timeEnd - timeStart}ms`);\n}\n\nif (require.main === module) {\n  main()\n    .then(() => process.exit(0))\n    .catch((error) => {\n      console.error('Error:', error);\n      process.exit(1);\n    });\n}\n",
      language: 'ts',
      source: 'examples/typescript/durable-event/run.ts',
      highlights: {},
    },
  L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3R5cGVzY3JpcHQvZHVyYWJsZS1ldmVudC93b3JrZXIudHM_:
    {
      content:
        "import { hatchet } from '../hatchet-client';\nimport { durableEvent } from './workflow';\n\nasync function main() {\n  const worker = await hatchet.worker('durable-event-worker', {\n    workflows: [durableEvent],\n  });\n\n  await worker.start();\n}\n\nif (require.main === module) {\n  main();\n}\n",
      language: 'ts',
      source: 'examples/typescript/durable-event/worker.ts',
      highlights: {},
    },
  L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3R5cGVzY3JpcHQvZHVyYWJsZS1ldmVudC93b3JrZmxvdy50cw__:
    {
      content:
        "// import sleep from '@hatchet-dev/typescript-sdk/util/sleep';\nimport { hatchet } from '../hatchet-client';\n\n// ❓ Durable Event\nexport const durableEvent = hatchet.durableTask({\n  name: 'durable-event',\n  executionTimeout: '10m',\n  fn: async (_, ctx) => {\n    const res = ctx.waitFor({\n      eventKey: 'user:update',\n    });\n\n    console.log('res', res);\n\n    return {\n      Value: 'done',\n    };\n  },\n});\n\nexport const durableEventWithFilter = hatchet.durableTask({\n  name: 'durable-event-with-filter',\n  executionTimeout: '10m',\n  fn: async (_, ctx) => {\n    // ❓ Durable Event With Filter\n    const res = ctx.waitFor({\n      eventKey: 'user:update',\n      expression: \"input.userId == '1234'\",\n    });\n\n    console.log('res', res);\n\n    return {\n      Value: 'done',\n    };\n  },\n});\n",
      language: 'ts',
      source: 'examples/typescript/durable-event/workflow.ts',
      highlights: {},
    },
  L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3R5cGVzY3JpcHQvZHVyYWJsZS1zbGVlcC9ldmVudC50cw__:
    {
      content:
        "import { hatchet } from '../hatchet-client';\n\nasync function main() {\n  const event = await hatchet.events.push('user:event', {\n    Data: { Hello: 'World' },\n  });\n}\n\nif (require.main === module) {\n  main()\n    .then(() => process.exit(0))\n    .catch((error) => {\n      console.error('Error:', error);\n      process.exit(1);\n    });\n}\n",
      language: 'ts',
      source: 'examples/typescript/durable-sleep/event.ts',
      highlights: {},
    },
  L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3R5cGVzY3JpcHQvZHVyYWJsZS1zbGVlcC9ydW4udHM_:
    {
      content:
        "import { durableSleep } from './workflow';\n\nasync function main() {\n  const timeStart = Date.now();\n  const res = await durableSleep.run({});\n  const timeEnd = Date.now();\n  console.log(`Time taken: ${timeEnd - timeStart}ms`);\n}\n\nif (require.main === module) {\n  main()\n    .then(() => process.exit(0))\n    .catch((error) => {\n      console.error('Error:', error);\n      process.exit(1);\n    });\n}\n",
      language: 'ts',
      source: 'examples/typescript/durable-sleep/run.ts',
      highlights: {},
    },
  L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3R5cGVzY3JpcHQvZHVyYWJsZS1zbGVlcC93b3JrZXIudHM_:
    {
      content:
        "import { hatchet } from '../hatchet-client';\nimport { durableSleep } from './workflow';\n\nasync function main() {\n  const worker = await hatchet.worker('sleep-worker', {\n    workflows: [durableSleep],\n  });\n\n  await worker.start();\n}\n\nif (require.main === module) {\n  main();\n}\n",
      language: 'ts',
      source: 'examples/typescript/durable-sleep/worker.ts',
      highlights: {},
    },
  L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3R5cGVzY3JpcHQvZHVyYWJsZS1zbGVlcC93b3JrZmxvdy50cw__:
    {
      content:
        "// import sleep from '@hatchet-dev/typescript-sdk/util/sleep';\nimport { hatchet } from '../hatchet-client';\n\nexport const durableSleep = hatchet.workflow({\n  name: 'durable-sleep',\n});\n\n// ❓ Durable Sleep\ndurableSleep.durableTask({\n  name: 'durable-sleep',\n  executionTimeout: '10m',\n  fn: async (_, ctx) => {\n    console.log('sleeping for 5s');\n    const sleepRes = await ctx.sleepFor('5s');\n    console.log('done sleeping for 5s', sleepRes);\n\n    return {\n      Value: 'done',\n    };\n  },\n});\n",
      language: 'ts',
      source: 'examples/typescript/durable-sleep/workflow.ts',
      highlights: {},
    },
  L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3R5cGVzY3JpcHQvaGF0Y2hldC1jbGllbnQudHM_:
    {
      content:
        "import { HatchetClient } from '@hatchet-dev/typescript-sdk/v1';\n\nexport const hatchet = HatchetClient.init();\n",
      language: 'ts',
      source: 'examples/typescript/hatchet-client.ts',
      highlights: {},
    },
  L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3R5cGVzY3JpcHQvaW5mZXJyZWQtdHlwaW5nL3J1bi50cw__:
    {
      content:
        "\nimport { crazyWorkflow, declaredType, inferredType, inferredTypeDurable } from './workflow';\n\nasync function main() {\n  const declaredTypeRun = declaredType.run({\n    Message: 'hello',\n  });\n\n  const inferredTypeRun = inferredType.run({\n    Message: 'hello',\n  });\n\n  const crazyWorkflowRun = crazyWorkflow.run({\n    Message: 'hello',\n  });\n\n  const inferredTypeDurableRun = inferredTypeDurable.run({\n    Message: 'Durable Task',\n  });\n\n  const [declaredTypeResult, inferredTypeResult, inferredTypeDurableResult, crazyWorkflowResult] =\n    await Promise.all([declaredTypeRun, inferredTypeRun, inferredTypeDurableRun, crazyWorkflowRun]);\n\n  console.log('declaredTypeResult', declaredTypeResult);\n  console.log('inferredTypeResult', inferredTypeResult);\n  console.log('inferredTypeDurableResult', inferredTypeDurableResult);\n  console.log('crazyWorkflowResult', crazyWorkflowResult);\n  console.log('declaredTypeResult.TransformedMessage', declaredTypeResult.TransformedMessage);\n  console.log('inferredTypeResult.TransformedMessage', inferredTypeResult.TransformedMessage);\n  console.log(\n    'inferredTypeDurableResult.TransformedMessage',\n    inferredTypeDurableResult.TransformedMessage\n  );\n  console.log('crazyWorkflowResult.TransformedMessage', crazyWorkflowResult.TransformedMessage);\n}\n\nif (require.main === module) {\n  main();\n}\n",
      language: 'ts',
      source: 'examples/typescript/inferred-typing/run.ts',
      highlights: {},
    },
  L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3R5cGVzY3JpcHQvaW5mZXJyZWQtdHlwaW5nL3dvcmtlci50cw__:
    {
      content:
        "import { hatchet } from '../hatchet-client';\nimport { declaredType, inferredType, inferredTypeDurable, crazyWorkflow } from './workflow';\n\nasync function main() {\n  const worker = await hatchet.worker('simple-worker', {\n    workflows: [declaredType, inferredType, inferredTypeDurable, crazyWorkflow],\n  });\n\n  await worker.start();\n}\n\nif (require.main === module) {\n  main();\n}\n",
      language: 'ts',
      source: 'examples/typescript/inferred-typing/worker.ts',
      highlights: {},
    },
  L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3R5cGVzY3JpcHQvaW5mZXJyZWQtdHlwaW5nL3dvcmtmbG93LnRz:
    {
      content:
        "import { hatchet } from '../hatchet-client';\n\ntype SimpleInput = {\n  Message: string;\n};\n\ntype SimpleOutput = {\n  TransformedMessage: string;\n};\n\nexport const declaredType = hatchet.task<SimpleInput, SimpleOutput>({\n  name: 'declared-type',\n  fn: (input) => {\n    return {\n      TransformedMessage: input.Message.toLowerCase(),\n    };\n  },\n});\n\nexport const inferredType = hatchet.task({\n  name: 'inferred-type',\n  fn: (input: SimpleInput) => {\n    return {\n      TransformedMessage: input.Message.toUpperCase(),\n    };\n  },\n});\n\nexport const inferredTypeDurable = hatchet.durableTask({\n  name: 'inferred-type-durable',\n  fn: async (input: SimpleInput, ctx) => {\n    // await ctx.sleepFor('5s');\n\n    return {\n      TransformedMessage: input.Message.toUpperCase(),\n    };\n  },\n});\n\nexport const crazyWorkflow = hatchet.workflow<any, any>({\n  name: 'crazy-workflow',\n});\n\nconst step1 = crazyWorkflow.task(declaredType);\n// crazyWorkflow.task(inferredTypeDurable);\n\ncrazyWorkflow.task({\n  parents: [step1],\n  ...inferredType.taskDef,\n});\n",
      language: 'ts',
      source: 'examples/typescript/inferred-typing/workflow.ts',
      highlights: {},
    },
  L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3R5cGVzY3JpcHQvbGFuZGluZ19wYWdlL2R1cmFibGUtZXhjdXRpb24udHM_:
    {
      content:
        "\nimport { Or } from '@hatchet-dev/typescript-sdk/v1/conditions';\nimport { hatchet } from '../hatchet-client';\n\n// (optional) Define the input type for the workflow\nexport type SimpleInput = {\n  Message: string;\n};\nasync function main() {\n  // ❓ Declaring a Durable Task\n  const simple = hatchet.durableTask({\n    name: 'simple',\n    fn: async (input: SimpleInput, ctx) => {\n      await ctx.waitFor(\n        Or(\n          {\n            eventKey: 'user:pay',\n            expression: 'input.Status == \"PAID\"',\n          },\n          {\n            sleepFor: '24h',\n          }\n        )\n      );\n\n      return {\n        TransformedMessage: input.Message.toLowerCase(),\n      };\n    },\n  });\n\n  // ❓ Running a Task\n  const result = await simple.run({ Message: 'Hello, World!' });\n\n}\n\nif (require.main === module) {\n  main();\n}\n",
      language: 'ts',
      source: 'examples/typescript/landing_page/durable-excution.ts',
      highlights: {},
    },
  L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3R5cGVzY3JpcHQvbGFuZGluZ19wYWdlL2V2ZW50LXNpZ25hbGluZy50cw__:
    {
      content:
        "import { hatchet } from '../hatchet-client';\n\n// (optional) Define the input type for the workflow\nexport type SimpleInput = {\n  Message: string;\n};\n\n// ❓ Trigger on an event\nexport const simple = hatchet.task({\n  name: 'simple',\n  onEvents: ['user:created'],\n  fn: (input: SimpleInput) => {\n    // ...\n    return {\n      TransformedMessage: input.Message.toLowerCase(),\n    };\n  },\n});\n",
      language: 'ts',
      source: 'examples/typescript/landing_page/event-signaling.ts',
      highlights: {},
    },
  L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3R5cGVzY3JpcHQvbGFuZGluZ19wYWdlL2Zsb3ctY29udHJvbC50cw__:
    {
      content:
        "import { ConcurrencyLimitStrategy } from '@hatchet-dev/typescript-sdk/protoc/v1/workflows';\nimport { hatchet } from '../hatchet-client';\n\n// (optional) Define the input type for the workflow\nexport type SimpleInput = {\n  Message: string;\n};\n\n// ❓ Process what you can handle\nexport const simple = hatchet.task({\n  name: 'simple',\n  concurrency: {\n    expression: 'input.user_id',\n    limitStrategy: ConcurrencyLimitStrategy.GROUP_ROUND_ROBIN,\n    maxRuns: 1,\n  },\n  rateLimits: [\n    {\n      key: 'api_throttle',\n      units: 1,\n    },\n  ],\n  fn: (input: SimpleInput) => {\n    return {\n      TransformedMessage: input.Message.toLowerCase(),\n    };\n  },\n});\n",
      language: 'ts',
      source: 'examples/typescript/landing_page/flow-control.ts',
      highlights: {},
    },
  L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3R5cGVzY3JpcHQvbGFuZGluZ19wYWdlL3F1ZXVlcy50cw__:
    {
      content:
        "\nimport { hatchet } from '../hatchet-client';\n\n// (optional) Define the input type for the workflow\nexport type SimpleInput = {\n  Message: string;\n};\nasync function main() {\n  // ❓ Declaring a Task\n  const simple = hatchet.task({\n    name: 'simple',\n    fn: (input: SimpleInput) => {\n      return {\n        TransformedMessage: input.Message.toLowerCase(),\n      };\n    },\n  });\n\n  // ❓ Running a Task\n  const result = await simple.run({ Message: 'Hello, World!' });\n\n}\n\nif (require.main === module) {\n  main();\n}\n",
      language: 'ts',
      source: 'examples/typescript/landing_page/queues.ts',
      highlights: {},
    },
  L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3R5cGVzY3JpcHQvbGFuZGluZ19wYWdlL3NjaGVkdWxpbmcudHM_:
    {
      content:
        "import { simple } from './flow-control';\n\n// ❓ Schedules and Crons\nconst tomorrow = new Date(Date.now() + 1000 * 60 * 60 * 24);\nconst scheduled = simple.schedule(tomorrow, {\n  Message: 'Hello, World!',\n});\n\nconst cron = simple.cron('every-day', '0 0 * * *', {\n  Message: 'Hello, World!',\n});\n",
      language: 'ts',
      source: 'examples/typescript/landing_page/scheduling.ts',
      highlights: {},
    },
  L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3R5cGVzY3JpcHQvbGFuZGluZ19wYWdlL3Rhc2stcm91dGluZy50cw__:
    {
      content:
        "import { hatchet } from '../hatchet-client';\n\n// (optional) Define the input type for the workflow\nexport type SimpleInput = {\n  Message: string;\n};\n\n// ❓ Route tasks to workers with matching labels\nexport const simple = hatchet.task({\n  name: 'simple',\n  desiredWorkerLabels: {\n    cpu: {\n      value: '2x',\n    },\n  },\n  fn: (input: SimpleInput) => {\n    return {\n      TransformedMessage: input.Message.toLowerCase(),\n    };\n  },\n});\n\nhatchet.worker('task-routing-worker', {\n  workflows: [simple],\n  labels: {\n    cpu: process.env.CPU_LABEL,\n  },\n});\n",
      language: 'ts',
      source: 'examples/typescript/landing_page/task-routing.ts',
      highlights: {},
    },
  L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3R5cGVzY3JpcHQvbGVnYWN5L3J1bi50cw__:
    {
      content:
        "import { hatchet } from '../hatchet-client';\nimport { simple } from './workflow';\n\nasync function main() {\n  const res = await hatchet.run<{ Message: string }, { step2: string }>(simple, {\n    Message: 'hello',\n  });\n  console.log(res.step2);\n}\n\nif (require.main === module) {\n  main();\n}\n",
      language: 'ts',
      source: 'examples/typescript/legacy/run.ts',
      highlights: {},
    },
  L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3R5cGVzY3JpcHQvbGVnYWN5L3dvcmtlci50cw__:
    {
      content:
        "import { hatchet } from '../hatchet-client';\nimport { simple } from './workflow';\n\nasync function main() {\n  const worker = await hatchet.worker('legacy-worker', {\n    workflows: [simple],\n  });\n\n  await worker.start();\n}\n\nif (require.main === module) {\n  main();\n}\n",
      language: 'ts',
      source: 'examples/typescript/legacy/worker.ts',
      highlights: {},
    },
  L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3R5cGVzY3JpcHQvbGVnYWN5L3dvcmtmbG93LnRz:
    {
      content:
        "import { Workflow } from '@hatchet-dev/typescript-sdk/workflow';\n\nexport const simple: Workflow = {\n  id: 'legacy-workflow',\n  description: 'test',\n  on: {\n    event: 'user:create',\n  },\n  steps: [\n    {\n      name: 'step1',\n      run: async (ctx) => {\n        const input = ctx.workflowInput();\n\n        return { step1: `original input: ${input.Message}` };\n      },\n    },\n    {\n      name: 'step2',\n      parents: ['step1'],\n      run: (ctx) => {\n        const step1Output = ctx.stepOutput('step1');\n\n        return { step2: `step1 output: ${step1Output.step1}` };\n      },\n    },\n  ],\n};\n",
      language: 'ts',
      source: 'examples/typescript/legacy/workflow.ts',
      highlights: {},
    },
  L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3R5cGVzY3JpcHQvbWlncmF0aW9uLWd1aWRlcy9oYXRjaGV0LWNsaWVudC50cw__:
    {
      content:
        "import HatchetClient from '@hatchet-dev/typescript-sdk/sdk';\n\nexport const hatchet = HatchetClient.init();\n",
      language: 'ts',
      source: 'examples/typescript/migration-guides/hatchet-client.ts',
      highlights: {},
    },
  L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3R5cGVzY3JpcHQvbWlncmF0aW9uLWd1aWRlcy9tZXJnZW50LnRz:
    {
      content:
        "import { hatchet } from './hatchet-client';\n\nfunction processImage(\n  imageUrl: string,\n  filters: string[]\n): Promise<{ url: string; size: number; format: string }> {\n  // Do some image processing\n  return Promise.resolve({ url: imageUrl, size: 100, format: 'png' });\n}\n// ❓ Before (Mergent)\nexport async function processImageTask(req: { body: { imageUrl: string; filters: string[] } }) {\n  const { imageUrl, filters } = req.body;\n  try {\n    const result = await processImage(imageUrl, filters);\n    return { success: true, processedUrl: result.url };\n  } catch (error) {\n    console.error('Image processing failed:', error);\n    throw error;\n  }\n}\n\n// ❓ After (Hatchet)\ntype ImageProcessInput = {\n  imageUrl: string;\n  filters: string[];\n};\n\ntype ImageProcessOutput = {\n  processedUrl: string;\n  metadata: {\n    size: number;\n    format: string;\n    appliedFilters: string[];\n  };\n};\n\nexport const imageProcessor = hatchet.task({\n  name: 'image-processor',\n  retries: 3,\n  executionTimeout: '10m',\n  fn: async ({ imageUrl, filters }: ImageProcessInput): Promise<ImageProcessOutput> => {\n    // Do some image processing\n    const result = await processImage(imageUrl, filters);\n\n    if (!result.url) throw new Error('Processing failed to generate URL');\n\n    return {\n      processedUrl: result.url,\n      metadata: {\n        size: result.size,\n        format: result.format,\n        appliedFilters: filters,\n      },\n    };\n  },\n});\n\nasync function run() {\n  // ❓ Running a task (Mergent)\n  const options = {\n    method: 'POST',\n    headers: { Authorization: 'Bearer <token>', 'Content-Type': 'application/json' },\n    body: JSON.stringify({\n      name: '4cf95241-fa19-47ef-8a67-71e483747649',\n      queue: 'default',\n      request: {\n        url: 'https://example.com',\n        headers: { Authorization: 'fake-secret-token', 'Content-Type': 'application/json' },\n        body: 'Hello, world!',\n      },\n    }),\n  };\n\n  fetch('https://api.mergent.co/v2/tasks', options)\n    .then((response) => response.json())\n    .then((response) => console.log(response))\n    .catch((err) => console.error(err));\n\n  // ❓ Running a task (Hatchet)\n  const result = await imageProcessor.run({\n    imageUrl: 'https://example.com/image.png',\n    filters: ['blur'],\n  });\n\n  // you can await fully typed results\n  console.log(result);\n\n}\n\nasync function schedule() {\n  // ❓ Scheduling tasks (Mergent)\n  const options = {\n    // same options as before\n    body: JSON.stringify({\n      // same body as before\n      delay: '5m',\n    }),\n  };\n\n  // ❓ Scheduling tasks (Hatchet)\n  // Schedule the task to run at a specific time\n  const runAt = new Date(Date.now() + 1000 * 60 * 60 * 24);\n  imageProcessor.schedule(runAt, {\n    imageUrl: 'https://example.com/image.png',\n    filters: ['blur'],\n  });\n\n  // Schedule the task to run every hour\n  imageProcessor.cron('run-hourly', '0 * * * *', {\n    imageUrl: 'https://example.com/image.png',\n    filters: ['blur'],\n  });\n\n}\n",
      language: 'ts',
      source: 'examples/typescript/migration-guides/mergent.ts',
      highlights: {},
    },
  L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3R5cGVzY3JpcHQvbXVsdGlwbGVfd2ZfY29uY3VycmVuY3kvcnVuLnRz:
    {
      content:
        "import { multiConcurrency } from './workflow';\n\nasync function main() {\n  const res = await multiConcurrency.run([\n    {\n      Message: 'Hello World',\n      GroupKey: 'A',\n    },\n    {\n      Message: 'Goodbye Moon',\n      GroupKey: 'A',\n    },\n    {\n      Message: 'Hello World B',\n      GroupKey: 'B',\n    },\n  ]);\n  console.log(res[0]['to-lower'].TransformedMessage);\n  console.log(res[1]['to-lower'].TransformedMessage);\n  console.log(res[2]['to-lower'].TransformedMessage);\n}\n\nif (require.main === module) {\n  main().then(() => process.exit(0));\n}\n",
      language: 'ts',
      source: 'examples/typescript/multiple_wf_concurrency/run.ts',
      highlights: {},
    },
  L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3R5cGVzY3JpcHQvbXVsdGlwbGVfd2ZfY29uY3VycmVuY3kvd29ya2VyLnRz:
    {
      content:
        "import { hatchet } from '../hatchet-client';\nimport { multiConcurrency } from './workflow';\n\nasync function main() {\n  const worker = await hatchet.worker('simple-concurrency-worker', {\n    workflows: [multiConcurrency],\n  });\n\n  await worker.start();\n}\n\nif (require.main === module) {\n  main();\n}\n",
      language: 'ts',
      source: 'examples/typescript/multiple_wf_concurrency/worker.ts',
      highlights: {},
    },
  L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3R5cGVzY3JpcHQvbXVsdGlwbGVfd2ZfY29uY3VycmVuY3kvd29ya2Zsb3cudHM_:
    {
      content:
        "import { ConcurrencyLimitStrategy } from '@hatchet-dev/typescript-sdk/workflow';\nimport { hatchet } from '../hatchet-client';\n\ntype SimpleInput = {\n  Message: string;\n  GroupKey: string;\n};\n\ntype SimpleOutput = {\n  'to-lower': {\n    TransformedMessage: string;\n  };\n};\n\nconst sleep = (ms: number) =>\n  new Promise((resolve) => {\n    setTimeout(resolve, ms);\n  });\n\n// ❓ Concurrency Strategy With Key\nexport const multiConcurrency = hatchet.workflow<SimpleInput, SimpleOutput>({\n  name: 'simple-concurrency',\n  concurrency: [\n    {\n      maxRuns: 1,\n      limitStrategy: ConcurrencyLimitStrategy.GROUP_ROUND_ROBIN,\n      expression: 'input.GroupKey',\n    },\n    {\n      maxRuns: 1,\n      limitStrategy: ConcurrencyLimitStrategy.GROUP_ROUND_ROBIN,\n      expression: 'input.UserId',\n    },\n  ],\n});\n\nmultiConcurrency.task({\n  name: 'to-lower',\n  fn: async (input) => {\n    await sleep(Math.floor(Math.random() * (1000 - 200 + 1)) + 200);\n    return {\n      TransformedMessage: input.Message.toLowerCase(),\n    };\n  },\n});\n",
      language: 'ts',
      source: 'examples/typescript/multiple_wf_concurrency/workflow.ts',
      highlights: {},
    },
  L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3R5cGVzY3JpcHQvbm9uX3JldHJ5YWJsZS9ydW4udHM_:
    {
      content:
        "import { nonRetryableWorkflow } from './workflow';\n\nasync function main() {\n  const res = await nonRetryableWorkflow.runNoWait({});\n  console.log(res);\n}\n\nif (require.main === module) {\n  main();\n}\n",
      language: 'ts',
      source: 'examples/typescript/non_retryable/run.ts',
      highlights: {},
    },
  L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3R5cGVzY3JpcHQvbm9uX3JldHJ5YWJsZS93b3JrZXIudHM_:
    {
      content:
        "import { hatchet } from '../hatchet-client';\nimport { nonRetryableWorkflow } from './workflow';\n\nasync function main() {\n  const worker = await hatchet.worker('no-retry-worker', {\n    workflows: [nonRetryableWorkflow],\n  });\n\n  await worker.start();\n}\n\nif (require.main === module) {\n  main();\n}\n",
      language: 'ts',
      source: 'examples/typescript/non_retryable/worker.ts',
      highlights: {},
    },
  L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3R5cGVzY3JpcHQvbm9uX3JldHJ5YWJsZS93b3JrZmxvdy50cw__:
    {
      content:
        "import { NonRetryableError } from '@hatchet-dev/typescript-sdk/v1/task';\nimport { hatchet } from '../hatchet-client';\n\nexport const nonRetryableWorkflow = hatchet.workflow({\n  name: 'no-retry-workflow',\n});\n\n// ❓ Non-retrying task\nconst shouldNotRetry = nonRetryableWorkflow.task({\n  name: 'should-not-retry',\n  fn: () => {\n    throw new NonRetryableError('This task should not retry');\n  },\n  retries: 1,\n});\n\n// Create a task that should retry\nconst shouldRetryWrongErrorType = nonRetryableWorkflow.task({\n  name: 'should-retry-wrong-error-type',\n  fn: () => {\n    throw new Error('This task should not retry');\n  },\n  retries: 1,\n});\n\nconst shouldNotRetrySuccessfulTask = nonRetryableWorkflow.task({\n  name: 'should-not-retry-successful-task',\n  fn: () => {},\n});\n",
      language: 'ts',
      source: 'examples/typescript/non_retryable/workflow.ts',
      highlights: {},
    },
  L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3R5cGVzY3JpcHQvb25fY3Jvbi93b3JrZXIudHM_:
    {
      content:
        "import { hatchet } from '../hatchet-client';\nimport { onCron } from './workflow';\n\nasync function main() {\n  const worker = await hatchet.worker('on-cron-worker', {\n    workflows: [onCron],\n  });\n\n  await worker.start();\n}\n\nif (require.main === module) {\n  main();\n}\n",
      language: 'ts',
      source: 'examples/typescript/on_cron/worker.ts',
      highlights: {},
    },
  L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3R5cGVzY3JpcHQvb25fY3Jvbi93b3JrZmxvdy50cw__:
    {
      content:
        "import { hatchet } from '../hatchet-client';\n\nexport type Input = {\n  Message: string;\n};\n\ntype OnCronOutput = {\n  job: {\n    TransformedMessage: string;\n  };\n};\n\n// ❓ Run Workflow on Cron\nexport const onCron = hatchet.workflow<Input, OnCronOutput>({\n  name: 'on-cron-workflow',\n  on: {\n    // 👀 add a cron expression to run the workflow every 15 minutes\n    cron: '*/15 * * * *',\n  },\n});\n\nonCron.task({\n  name: 'job',\n  fn: (input) => {\n    return {\n      TransformedMessage: input.Message.toLowerCase(),\n    };\n  },\n});\n",
      language: 'ts',
      source: 'examples/typescript/on_cron/workflow.ts',
      highlights: {},
    },
  L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3R5cGVzY3JpcHQvb25fZXZlbnQvZXZlbnQudHM_:
    {
      content:
        "import { hatchet } from '../hatchet-client';\nimport { Input } from './workflow';\n\nasync function main() {\n  // ❓ Pushing an Event\n  const res = await hatchet.events.push<Input>('simple-event:create', {\n    Message: 'hello',\n  });\n  console.log(res.eventId);\n}\n\nif (require.main === module) {\n  main();\n}\n",
      language: 'ts',
      source: 'examples/typescript/on_event/event.ts',
      highlights: {},
    },
  L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3R5cGVzY3JpcHQvb25fZXZlbnQvd29ya2VyLnRz:
    {
      content:
        "import { hatchet } from '../hatchet-client';\nimport { lower, upper } from './workflow';\n\nasync function main() {\n  const worker = await hatchet.worker('on-event-worker', {\n    workflows: [lower, upper],\n  });\n\n  await worker.start();\n}\n\nif (require.main === module) {\n  main();\n}\n",
      language: 'ts',
      source: 'examples/typescript/on_event/worker.ts',
      highlights: {},
    },
  L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3R5cGVzY3JpcHQvb25fZXZlbnQvd29ya2Zsb3cudHM_:
    {
      content:
        "import { hatchet } from '../hatchet-client';\n\nexport type Input = {\n  Message: string;\n};\n\nexport const SIMPLE_EVENT = 'simple-event:create';\n\ntype LowerOutput = {\n  lower: {\n    TransformedMessage: string;\n  };\n};\n\n// ❓ Run workflow on event\nexport const lower = hatchet.workflow<Input, LowerOutput>({\n  name: 'lower',\n  // 👀 Declare the event that will trigger the workflow\n  onEvents: ['simple-event:create'],\n});\n\nlower.task({\n  name: 'lower',\n  fn: (input) => {\n    return {\n      TransformedMessage: input.Message.toLowerCase(),\n    };\n  },\n});\n\ntype UpperOutput = {\n  upper: {\n    TransformedMessage: string;\n  };\n};\n\nexport const upper = hatchet.workflow<Input, UpperOutput>({\n  name: 'upper',\n  on: {\n    event: SIMPLE_EVENT,\n  },\n});\n\nupper.task({\n  name: 'upper',\n  fn: (input) => {\n    return {\n      TransformedMessage: input.Message.toUpperCase(),\n    };\n  },\n});\n",
      language: 'ts',
      source: 'examples/typescript/on_event/workflow.ts',
      highlights: {},
    },
  L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3R5cGVzY3JpcHQvb25fZXZlbnQgY29weS9ldmVudC50cw__:
    {
      content:
        "import { hatchet } from '../hatchet-client';\nimport { Input } from './workflow';\n\nasync function main() {\n  // ❓ Pushing an Event\n  const res = await hatchet.event.push<Input>('simple-event:create', {\n    Message: 'hello',\n  });\n  console.log(res.eventId);\n}\n\nif (require.main === module) {\n  main();\n}\n",
      language: 'ts',
      source: 'examples/typescript/on_event copy/event.ts',
      highlights: {},
    },
  L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3R5cGVzY3JpcHQvb25fZXZlbnQgY29weS93b3JrZXIudHM_:
    {
      content:
        "import { hatchet } from '../hatchet-client';\nimport { lower, upper } from './workflow';\n\nasync function main() {\n  const worker = await hatchet.worker('on-event-worker', {\n    workflows: [lower, upper],\n  });\n\n  await worker.start();\n}\n\nif (require.main === module) {\n  main();\n}\n",
      language: 'ts',
      source: 'examples/typescript/on_event copy/worker.ts',
      highlights: {},
    },
  L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3R5cGVzY3JpcHQvb25fZXZlbnQgY29weS93b3JrZmxvdy50cw__:
    {
      content:
        "import { hatchet } from '../hatchet-client';\n\nexport type Input = {\n  Message: string;\n};\n\ntype LowerOutput = {\n  lower: {\n    TransformedMessage: string;\n  };\n};\n\n// ❓ Run workflow on event\nexport const lower = hatchet.workflow<Input, LowerOutput>({\n  name: 'lower',\n  on: {\n    // 👀 Declare the event that will trigger the workflow\n    event: 'simple-event:create',\n  },\n});\n\nlower.task({\n  name: 'lower',\n  fn: (input) => {\n    return {\n      TransformedMessage: input.Message.toLowerCase(),\n    };\n  },\n});\n\ntype UpperOutput = {\n  upper: {\n    TransformedMessage: string;\n  };\n};\n\nexport const upper = hatchet.workflow<Input, UpperOutput>({\n  name: 'upper',\n  on: {\n    event: 'simple-event:create',\n  },\n});\n\nupper.task({\n  name: 'upper',\n  fn: (input) => {\n    return {\n      TransformedMessage: input.Message.toUpperCase(),\n    };\n  },\n});\n",
      language: 'ts',
      source: 'examples/typescript/on_event copy/workflow.ts',
      highlights: {},
    },
  L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3R5cGVzY3JpcHQvb25fZmFpbHVyZS9ydW4udHM_:
    {
      content:
        "\nimport { failureWorkflow } from './workflow';\n\nasync function main() {\n  try {\n    const res = await failureWorkflow.run({});\n    console.log(res);\n  } catch (e) {\n    console.log('error', e);\n  }\n}\n\nif (require.main === module) {\n  main()\n    .catch(console.error)\n    .finally(() => process.exit(0));\n}\n",
      language: 'ts',
      source: 'examples/typescript/on_failure/run.ts',
      highlights: {},
    },
  L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3R5cGVzY3JpcHQvb25fZmFpbHVyZS93b3JrZXIudHM_:
    {
      content:
        "import { hatchet } from '../hatchet-client';\nimport { failureWorkflow } from './workflow';\n\nasync function main() {\n  const worker = await hatchet.worker('always-fail-worker', {\n    workflows: [failureWorkflow],\n  });\n\n  await worker.start();\n}\n\nif (require.main === module) {\n  main();\n}\n",
      language: 'ts',
      source: 'examples/typescript/on_failure/worker.ts',
      highlights: {},
    },
  L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3R5cGVzY3JpcHQvb25fZmFpbHVyZS93b3JrZmxvdy50cw__:
    {
      content:
        "\nimport { hatchet } from '../hatchet-client';\n\n// ❓ On Failure Task\nexport const failureWorkflow = hatchet.workflow({\n  name: 'always-fail',\n});\n\nfailureWorkflow.task({\n  name: 'always-fail',\n  fn: async () => {\n    throw new Error('intentional failure');\n  },\n});\n\nfailureWorkflow.onFailure({\n  name: 'on-failure',\n  fn: async (input, ctx) => {\n    console.log('onFailure for run:', ctx.workflowRunId());\n    return {\n      'on-failure': 'success',\n    };\n  },\n});\n",
      language: 'ts',
      source: 'examples/typescript/on_failure/workflow.ts',
      highlights: {},
    },
  L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3R5cGVzY3JpcHQvb25fc3VjY2Vzcy9ydW4udHM_:
    {
      content:
        "\nimport { onSuccessDag } from './workflow';\n\nasync function main() {\n  try {\n    const res2 = await onSuccessDag.run({});\n    console.log(res2);\n  } catch (e) {\n    console.log('error', e);\n  }\n}\n\nif (require.main === module) {\n  main()\n    .catch(console.error)\n    .finally(() => process.exit(0));\n}\n",
      language: 'ts',
      source: 'examples/typescript/on_success/run.ts',
      highlights: {},
    },
  L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3R5cGVzY3JpcHQvb25fc3VjY2Vzcy93b3JrZXIudHM_:
    {
      content:
        "import { hatchet } from '../hatchet-client';\nimport { onSuccessDag } from './workflow';\n\nasync function main() {\n  const worker = await hatchet.worker('always-succeed-worker', {\n    workflows: [onSuccessDag],\n  });\n\n  await worker.start();\n}\n\nif (require.main === module) {\n  main();\n}\n",
      language: 'ts',
      source: 'examples/typescript/on_success/worker.ts',
      highlights: {},
    },
  L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3R5cGVzY3JpcHQvb25fc3VjY2Vzcy93b3JrZmxvdy50cw__:
    {
      content:
        "\nimport { hatchet } from '../hatchet-client';\n\n// ❓ On Success DAG\nexport const onSuccessDag = hatchet.workflow({\n  name: 'on-success-dag',\n});\n\nonSuccessDag.task({\n  name: 'always-succeed',\n  fn: async () => {\n    return {\n      'always-succeed': 'success',\n    };\n  },\n});\nonSuccessDag.task({\n  name: 'always-succeed2',\n  fn: async () => {\n    return {\n      'always-succeed': 'success',\n    };\n  },\n});\n\n// 👀 onSuccess handler will run if all tasks in the workflow succeed\nonSuccessDag.onSuccess({\n  fn: (_, ctx) => {\n    console.log('onSuccess for run:', ctx.workflowRunId());\n    return {\n      'on-success': 'success',\n    };\n  },\n});\n",
      language: 'ts',
      source: 'examples/typescript/on_success/workflow.ts',
      highlights: {},
    },
  L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3R5cGVzY3JpcHQvcHJpb3JpdHkvcnVuLnRz:
    {
      content:
        "import { Priority } from '@hatchet-dev/typescript-sdk/v1';\nimport { priority } from './workflow';\n\nasync function main() {\n  try {\n    console.log('running priority workflow');\n\n    // ❓ Run a Task with a Priority\n    const run = priority.run(new Date(Date.now() + 60 * 60 * 1000), { priority: Priority.HIGH });\n\n    // ❓ Schedule and cron\n    const scheduled = priority.schedule(\n      new Date(Date.now() + 60 * 60 * 1000),\n      {},\n      { priority: Priority.HIGH }\n    );\n    const delayed = priority.delay(60 * 60 * 1000, {}, { priority: Priority.HIGH });\n    const cron = priority.cron(\n      `daily-cron-${Math.random()}`,\n      '0 0 * * *',\n      {},\n      { priority: Priority.HIGH }\n    );\n\n    const [scheduledResult, delayedResult] = await Promise.all([scheduled, delayed]);\n    console.log('scheduledResult', scheduledResult);\n    console.log('delayedResult', delayedResult);\n\n  } catch (e) {\n    console.log('error', e);\n  }\n}\n\nif (require.main === module) {\n  main()\n    .catch(console.error)\n    .finally(() => process.exit(0));\n}\n",
      language: 'ts',
      source: 'examples/typescript/priority/run.ts',
      highlights: {},
    },
  L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3R5cGVzY3JpcHQvcHJpb3JpdHkvd29ya2VyLnRz:
    {
      content:
        "import { hatchet } from '../hatchet-client';\nimport { priorityTasks } from './workflow';\n\nasync function main() {\n  const worker = await hatchet.worker('priority-worker', {\n    workflows: [...priorityTasks],\n  });\n\n  await worker.start();\n}\n\nif (require.main === module) {\n  main();\n}\n",
      language: 'ts',
      source: 'examples/typescript/priority/worker.ts',
      highlights: {},
    },
  L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3R5cGVzY3JpcHQvcHJpb3JpdHkvd29ya2Zsb3cudHM_:
    {
      content:
        "\nimport { Priority } from '@hatchet-dev/typescript-sdk/v1';\nimport { hatchet } from '../hatchet-client';\n\n// ❓ Simple Task Priority\nexport const priority = hatchet.task({\n  name: 'priority',\n  defaultPriority: Priority.MEDIUM,\n  fn: async (_, ctx) => {\n    return {\n      priority: ctx.priority(),\n    };\n  },\n});\n\n// ❓ Task Priority in a Workflow\nexport const priorityWf = hatchet.workflow({\n  name: 'priorityWf',\n  defaultPriority: Priority.LOW,\n});\n\npriorityWf.task({\n  name: 'child-medium',\n  fn: async (_, ctx) => {\n    return {\n      priority: ctx.priority(),\n    };\n  },\n});\n\npriorityWf.task({\n  name: 'child-high',\n  // will inherit the default priority from the workflow\n  fn: async (_, ctx) => {\n    return {\n      priority: ctx.priority(),\n    };\n  },\n});\n\nexport const priorityTasks = [priority, priorityWf];\n",
      language: 'ts',
      source: 'examples/typescript/priority/workflow.ts',
      highlights: {},
    },
  L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3R5cGVzY3JpcHQvcmF0ZV9saW1pdC93b3JrZmxvdy50cw__:
    {
      content:
        "import { RateLimitDuration } from '@hatchet-dev/typescript-sdk/protoc/v1/workflows';\nimport { hatchet } from '../hatchet-client';\n\n// ❓ Upsert Rate Limit\nhatchet.ratelimits.upsert({\n  key: 'api-service-rate-limit',\n  limit: 10,\n  duration: RateLimitDuration.SECOND,\n});\n\n// ❓ Static\nconst RATE_LIMIT_KEY = 'api-service-rate-limit';\n\nconst task1 = hatchet.task({\n  name: 'task1',\n  rateLimits: [\n    {\n      staticKey: RATE_LIMIT_KEY,\n      units: 1,\n    },\n  ],\n  fn: (input) => {\n    console.log('executed task1');\n  },\n});\n\n// ❓ Dynamic\nconst task2 = hatchet.task({\n  name: 'task2',\n  fn: (input: { userId: string }) => {\n    console.log('executed task2 for user: ', input.userId);\n  },\n  rateLimits: [\n    {\n      dynamicKey: 'input.userId',\n      units: 1,\n      limit: 10,\n      duration: RateLimitDuration.MINUTE,\n    },\n  ],\n});\n",
      language: 'ts',
      source: 'examples/typescript/rate_limit/workflow.ts',
      highlights: {},
    },
  L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3R5cGVzY3JpcHQvcmV0cmllcy9ydW4udHM_:
    {
      content:
        "\nimport { retries } from './workflow';\n\nasync function main() {\n  try {\n    const res = await retries.run({});\n    console.log(res);\n  } catch (e) {\n    console.log('error', e);\n  }\n}\n\nif (require.main === module) {\n  main()\n    .catch(console.error)\n    .finally(() => process.exit(0));\n}\n",
      language: 'ts',
      source: 'examples/typescript/retries/run.ts',
      highlights: {},
    },
  L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3R5cGVzY3JpcHQvcmV0cmllcy93b3JrZXIudHM_:
    {
      content:
        "import { hatchet } from '../hatchet-client';\nimport { retries } from './workflow';\n\nasync function main() {\n  const worker = await hatchet.worker('always-fail-worker', {\n    workflows: [retries],\n  });\n\n  await worker.start();\n}\n\nif (require.main === module) {\n  main();\n}\n",
      language: 'ts',
      source: 'examples/typescript/retries/worker.ts',
      highlights: {},
    },
  L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3R5cGVzY3JpcHQvcmV0cmllcy93b3JrZmxvdy50cw__:
    {
      content:
        "\nimport { hatchet } from '../hatchet-client';\n\n// ❓ Simple Step Retries\nexport const retries = hatchet.task({\n  name: 'retries',\n  retries: 3,\n  fn: async (_, ctx) => {\n    throw new Error('intentional failure');\n  },\n});\n\n// ❓ Retries with Count\nexport const retriesWithCount = hatchet.task({\n  name: 'retriesWithCount',\n  retries: 3,\n  fn: async (_, ctx) => {\n    // ❓ Get the current retry count\n    const retryCount = ctx.retryCount();\n\n    console.log(`Retry count: ${retryCount}`);\n\n    if (retryCount < 2) {\n      throw new Error('intentional failure');\n    }\n\n    return {\n      message: 'success',\n    };\n  },\n});\n\n// ❓ Retries with Backoff\nexport const withBackoff = hatchet.task({\n  name: 'withBackoff',\n  retries: 10,\n  backoff: {\n    // 👀 Maximum number of seconds to wait between retries\n    maxSeconds: 10,\n    // 👀 Factor to increase the wait time between retries.\n    // This sequence will be 2s, 4s, 8s, 10s, 10s, 10s... due to the maxSeconds limit\n    factor: 2,\n  },\n  fn: async () => {\n    throw new Error('intentional failure');\n  },\n});\n",
      language: 'ts',
      source: 'examples/typescript/retries/workflow.ts',
      highlights: {},
    },
  L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3R5cGVzY3JpcHQvc2ltcGxlL2J1bGsudHM_:
    {
      content:
        "\nimport { hatchet } from '../hatchet-client';\nimport { simple, SimpleInput } from './workflow';\n\nasync function main() {\n  // ❓ Bulk Run a Task\n  const res = await simple.run([\n    {\n      Message: 'HeLlO WoRlD',\n    },\n    {\n      Message: 'Hello MoOn',\n    },\n  ]);\n\n  // 👀 Access the results of the Task\n  console.log(res[0].TransformedMessage);\n  console.log(res[1].TransformedMessage);\n\n  // ❓ Bulk Run Tasks from within a Task\n  const parent = hatchet.task({\n    name: 'simple',\n    fn: async (input: SimpleInput, ctx) => {\n      // Bulk run two tasks in parallel\n      const child = await ctx.bulkRunChildren([\n        {\n          workflow: simple,\n          input: {\n            Message: 'Hello, World!',\n          },\n        },\n        {\n          workflow: simple,\n          input: {\n            Message: 'Hello, Moon!',\n          },\n        },\n      ]);\n\n      return {\n        TransformedMessage: `${child[0].TransformedMessage} ${child[1].TransformedMessage}`,\n      };\n    },\n  });\n\n}\n\nif (require.main === module) {\n  main();\n}\n",
      language: 'ts',
      source: 'examples/typescript/simple/bulk.ts',
      highlights: {},
    },
  L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3R5cGVzY3JpcHQvc2ltcGxlL2NsaWVudC1ydW4udHM_:
    {
      content:
        "// ❓ Client Run Methods\nimport { hatchet } from '../hatchet-client';\n\nhatchet.run('simple', { Message: 'Hello, World!' });\n\nhatchet.runNoWait('simple', { Message: 'Hello, World!' }, {});\n\nhatchet.schedules.create('simple', {\n  triggerAt: new Date(Date.now() + 1000 * 60 * 60 * 24),\n  input: { Message: 'Hello, World!' },\n});\n\nhatchet.crons.create('simple', {\n  name: 'my-cron',\n  expression: '0 0 * * *',\n  input: { Message: 'Hello, World!' },\n});\n",
      language: 'ts',
      source: 'examples/typescript/simple/client-run.ts',
      highlights: {},
    },
  L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3R5cGVzY3JpcHQvc2ltcGxlL2Nyb24udHM_:
    {
      content:
        "import { hatchet } from '../hatchet-client';\nimport { simple } from './workflow';\n\nasync function main() {\n  // ❓ Create\n  const cron = await simple.cron('simple-daily', '0 0 * * *', {\n    Message: 'hello',\n  });\n\n  // it may be useful to save the cron id for later\n  const cronId = cron.metadata.id;\n  console.log(cron.metadata.id);\n\n  // ❓ Delete\n  await hatchet.crons.delete(cronId);\n\n  // ❓ List\n  const crons = await hatchet.crons.list({\n    workflowId: simple.id,\n  });\n  console.log(crons);\n}\n\nif (require.main === module) {\n  main();\n}\n",
      language: 'ts',
      source: 'examples/typescript/simple/cron.ts',
      highlights: {},
    },
  L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3R5cGVzY3JpcHQvc2ltcGxlL2RlbGF5LnRz:
    {
      content:
        "import { hatchet } from '../hatchet-client';\nimport { simple } from './workflow';\n\nasync function main() {\n  const tomorrow = 24 * 60 * 60; // 1 day\n  const scheduled = await simple.delay(tomorrow, {\n    Message: 'hello',\n  });\n  console.log(scheduled.metadata.id);\n\n  await hatchet.schedules.delete(scheduled);\n}\n\nif (require.main === module) {\n  main();\n}\n",
      language: 'ts',
      source: 'examples/typescript/simple/delay.ts',
      highlights: {},
    },
  L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3R5cGVzY3JpcHQvc2ltcGxlL2VucXVldWUudHM_:
    {
      content:
        "\n\nimport { hatchet } from '../hatchet-client';\nimport { SimpleOutput } from './stub-workflow';\n// ❓ Enqueuing a Workflow (Fire and Forget)\nimport { simple } from './workflow';\n// ...\n\nasync function main() {\n  // 👀 Enqueue the workflow\n  const run = await simple.runNoWait({\n    Message: 'hello',\n  });\n\n  // 👀 Get the run ID of the workflow\n  const runId = await run.getWorkflowRunId();\n  // It may be helpful to store the run ID of the workflow\n  // in a database or other persistent storage for later use\n  console.log(runId);\n\n  // ❓ Subscribing to results\n  // the return object of the enqueue method is a WorkflowRunRef which includes a listener for the result of the workflow\n  const result = await run.result();\n  console.log(result);\n\n  // if you need to subscribe to the result of the workflow at a later time, you can use the runRef method and the stored runId\n  const ref = hatchet.runRef<SimpleOutput>(runId);\n  const result2 = await ref.result();\n  console.log(result2);\n\n}\n\nif (require.main === module) {\n  main();\n}\n",
      language: 'ts',
      source: 'examples/typescript/simple/enqueue.ts',
      highlights: {},
    },
  L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3R5cGVzY3JpcHQvc2ltcGxlL3J1bi50cw__:
    {
      content:
        "\nimport { hatchet } from '../hatchet-client';\nimport { simple } from './workflow';\n\nasync function main() {\n  // ❓ Running a Task\n  const res = await simple.run(\n    {\n      Message: 'HeLlO WoRlD',\n    },\n    {\n      additionalMetadata: {\n        test: 'test',\n      },\n    }\n  );\n\n  // 👀 Access the results of the Task\n  console.log(res.TransformedMessage);\n\n}\n\nexport async function extra() {\n  // ❓ Running Multiple Tasks\n  const res1 = simple.run({\n    Message: 'HeLlO WoRlD',\n  });\n\n  const res2 = simple.run({\n    Message: 'Hello MoOn',\n  });\n\n  const results = await Promise.all([res1, res2]);\n\n  console.log(results[0].TransformedMessage);\n  console.log(results[1].TransformedMessage);\n\n  // ❓ Spawning Tasks from within a Task\n  const parent = hatchet.task({\n    name: 'parent',\n    fn: async (input, ctx) => {\n      // Simply call ctx.runChild with the task you want to run\n      const child = await ctx.runChild(simple, {\n        Message: 'HeLlO WoRlD',\n      });\n\n      return {\n        result: child.TransformedMessage,\n      };\n    },\n  });\n\n}\n\nif (require.main === module) {\n  main();\n}\n",
      language: 'ts',
      source: 'examples/typescript/simple/run.ts',
      highlights: {},
    },
  L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3R5cGVzY3JpcHQvc2ltcGxlL3NjaGVkdWxlLnRz:
    {
      content:
        "\nimport { hatchet } from '../hatchet-client';\nimport { simple } from './workflow';\n\nasync function main() {\n  // ❓ Create a Scheduled Run\n\n  const runAt = new Date(new Date().setHours(12, 0, 0, 0) + 24 * 60 * 60 * 1000);\n\n  const scheduled = await simple.schedule(runAt, {\n    Message: 'hello',\n  });\n\n  // 👀 Get the scheduled run ID of the workflow\n  // it may be helpful to store the scheduled run ID of the workflow\n  // in a database or other persistent storage for later use\n  const scheduledRunId = scheduled.metadata.id;\n  console.log(scheduledRunId);\n\n  // ❓ Delete a Scheduled Run\n  await hatchet.schedules.delete(scheduled);\n\n  // ❓ List Scheduled Runs\n  const scheduledRuns = await hatchet.schedules.list({\n    workflowId: simple.id,\n  });\n  console.log(scheduledRuns);\n\n}\n\nif (require.main === module) {\n  main();\n}\n",
      language: 'ts',
      source: 'examples/typescript/simple/schedule.ts',
      highlights: {},
    },
  L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3R5cGVzY3JpcHQvc2ltcGxlL3N0dWItd29ya2Zsb3cudHM_:
    {
      content:
        "// ❓ Declaring an External Workflow Reference\nimport { hatchet } from '../hatchet-client';\n\n// (optional) Define the input type for the workflow\nexport type SimpleInput = {\n  Message: string;\n};\n\n// (optional) Define the output type for the workflow\nexport type SimpleOutput = {\n  'to-lower': {\n    TransformedMessage: string;\n  };\n};\n\n// declare the workflow with the same name as the\n// workflow name on the worker\nexport const simple = hatchet.workflow<SimpleInput, SimpleOutput>({\n  name: 'simple',\n});\n\n// you can use all the same run methods on the stub\n// with full type-safety\nsimple.run({ Message: 'Hello, World!' });\nsimple.runNoWait({ Message: 'Hello, World!' });\nsimple.schedule(new Date(), { Message: 'Hello, World!' });\nsimple.cron('my-cron', '0 0 * * *', { Message: 'Hello, World!' });\n",
      language: 'ts',
      source: 'examples/typescript/simple/stub-workflow.ts',
      highlights: {},
    },
  L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3R5cGVzY3JpcHQvc2ltcGxlL3dvcmtlci50cw__:
    {
      content:
        "// ❓ Declaring a Worker\nimport { hatchet } from '../hatchet-client';\nimport { simple } from './workflow';\nimport { parent, child } from './workflow-with-child';\n\nasync function main() {\n  const worker = await hatchet.worker('simple-worker', {\n    // 👀 Declare the workflows that the worker can execute\n    workflows: [simple, parent, child],\n    // 👀 Declare the number of concurrent task runs the worker can accept\n    slots: 100,\n  });\n\n  await worker.start();\n}\n\nif (require.main === module) {\n  main();\n}\n",
      language: 'ts',
      source: 'examples/typescript/simple/worker.ts',
      highlights: {},
    },
  L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3R5cGVzY3JpcHQvc2ltcGxlL3dvcmtmbG93LXdpdGgtY2hpbGQudHM_:
    {
      content:
        "// ❓ Declaring a Task\nimport { hatchet } from '../hatchet-client';\n\n// (optional) Define the input type for the workflow\nexport type ChildInput = {\n  Message: string;\n};\n\nexport type ParentInput = {\n  Message: string;\n};\n\nexport const child = hatchet.workflow<ChildInput>({\n  name: 'child',\n});\n\nexport const child1 = child.task({\n  name: 'child1',\n  fn: (input: ChildInput, ctx) => {\n    ctx.log('hello from the child1');\n    return {\n      TransformedMessage: input.Message.toLowerCase(),\n    };\n  },\n});\n\nexport const child2 = child.task({\n  name: 'child2',\n  fn: (input: ChildInput, ctx) => {\n    ctx.log('hello from the child2');\n    return {\n      TransformedMessage: input.Message.toLowerCase(),\n    };\n  },\n});\n\nexport const parent = hatchet.task({\n  name: 'parent',\n  fn: async (input: ParentInput, ctx) => {\n    const c = await ctx.runChild(child, {\n      Message: input.Message,\n    });\n\n    return {\n      TransformedMessage: 'not implemented',\n    };\n  },\n});\n\n// see ./worker.ts and ./run.ts for how to run the workflow\n",
      language: 'ts',
      source: 'examples/typescript/simple/workflow-with-child.ts',
      highlights: {},
    },
  L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3R5cGVzY3JpcHQvc2ltcGxlL3dvcmtmbG93LnRz:
    {
      content:
        "// ❓ Declaring a Task\nimport sleep from '@hatchet-dev/typescript-sdk/util/sleep';\nimport { hatchet } from '../hatchet-client';\n\n// (optional) Define the input type for the workflow\n\nexport type SimpleInput = {\n  Message: string;\n};\n\nexport const simple = hatchet.task({\n  name: 'simple',\n  retries: 3,\n  fn: async (input: SimpleInput, ctx) => {\n    ctx.log('hello from the workflow');\n    await sleep(100);\n    ctx.log('goodbye from the workflow');\n    await sleep(100);\n    if (ctx.retryCount() < 2) {\n      throw new Error('test error');\n    }\n    return {\n      TransformedMessage: input.Message.toLowerCase(),\n    };\n  },\n});\n\n// see ./worker.ts and ./run.ts for how to run the workflow\n",
      language: 'ts',
      source: 'examples/typescript/simple/workflow.ts',
      highlights: {
        input: {
          lines: [6],
          strings: ['input'],
        },
        retries: {
          lines: [9, 10, 11, 12],
          strings: [],
        },
        func: {
          lines: [9, 10, 11],
          strings: ['input', 'ctx'],
        },
      },
    },
  L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3R5cGVzY3JpcHQvc3RpY2t5L3J1bi50cw__:
    {
      content:
        "\nimport { retries } from '../retries/workflow';\n\nasync function main() {\n  try {\n    const res = await retries.run({});\n    console.log(res);\n  } catch (e) {\n    console.log('error', e);\n  }\n}\n\nif (require.main === module) {\n  main()\n    .catch(console.error)\n    .finally(() => process.exit(0));\n}\n",
      language: 'ts',
      source: 'examples/typescript/sticky/run.ts',
      highlights: {},
    },
  L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3R5cGVzY3JpcHQvc3RpY2t5L3dvcmtlci50cw__:
    {
      content:
        "import { hatchet } from '../hatchet-client';\nimport { retries } from '../retries/workflow';\n\nasync function main() {\n  const worker = await hatchet.worker('always-fail-worker', {\n    workflows: [retries],\n  });\n\n  await worker.start();\n}\n\nif (require.main === module) {\n  main();\n}\n",
      language: 'ts',
      source: 'examples/typescript/sticky/worker.ts',
      highlights: {},
    },
  L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3R5cGVzY3JpcHQvc3RpY2t5L3dvcmtmbG93LnRz:
    {
      content:
        "\nimport { StickyStrategy } from '@hatchet-dev/typescript-sdk/protoc/workflows';\nimport { hatchet } from '../hatchet-client';\nimport { child } from '../child_workflows/workflow';\n\n// ❓ Sticky Task\nexport const sticky = hatchet.task({\n  name: 'sticky',\n  retries: 3,\n  sticky: StickyStrategy.SOFT,\n  fn: async (_, ctx) => {\n    // specify a child workflow to run on the same worker\n    const result = await ctx.runChild(\n      child,\n      {\n        N: 1,\n      },\n      { sticky: true }\n    );\n\n    return {\n      result,\n    };\n  },\n});\n",
      language: 'ts',
      source: 'examples/typescript/sticky/workflow.ts',
      highlights: {},
    },
  L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3R5cGVzY3JpcHQvdGltZW91dHMvcnVuLnRz:
    {
      content:
        "\n// ❓ Running a Task with Results\nimport { cancellation } from './workflow';\n// ...\nasync function main() {\n  // 👀 Run the workflow with results\n  const res = await cancellation.run({});\n\n  // 👀 Access the results of the workflow\n  console.log(res.Completed);\n\n}\n\nif (require.main === module) {\n  main()\n    .catch(console.error)\n    .finally(() => process.exit(0));\n}\n",
      language: 'ts',
      source: 'examples/typescript/timeouts/run.ts',
      highlights: {},
    },
  L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3R5cGVzY3JpcHQvdGltZW91dHMvd29ya2VyLnRz:
    {
      content:
        "// ❓ Declaring a Worker\nimport { hatchet } from '../hatchet-client';\nimport { cancellation } from './workflow';\n\nasync function main() {\n  const worker = await hatchet.worker('cancellation-worker', {\n    // 👀 Declare the workflows that the worker can execute\n    workflows: [cancellation],\n    // 👀 Declare the number of concurrent task runs the worker can accept\n    slots: 100,\n  });\n\n  await worker.start();\n}\n\nif (require.main === module) {\n  main();\n}\n",
      language: 'ts',
      source: 'examples/typescript/timeouts/worker.ts',
      highlights: {},
    },
  L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3R5cGVzY3JpcHQvdGltZW91dHMvd29ya2Zsb3cudHM_:
    {
      content:
        "// ❓ Declaring a Task\nimport sleep from '@hatchet-dev/typescript-sdk/util/sleep';\nimport { hatchet } from '../hatchet-client';\n\n// (optional) Define the input type for the workflow\nexport const cancellation = hatchet.task({\n  name: 'cancellation',\n  executionTimeout: '3s',\n  fn: async (_, { cancelled }) => {\n    await sleep(10 * 1000);\n\n    if (cancelled) {\n      throw new Error('Task was cancelled');\n    }\n\n    return {\n      Completed: true,\n    };\n  },\n});\n\n// see ./worker.ts and ./run.ts for how to run the workflow\n",
      language: 'ts',
      source: 'examples/typescript/timeouts/workflow.ts',
      highlights: {},
    },
  L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3R5cGVzY3JpcHQvd2l0aF90aW1lb3V0cy93b3JrZmxvdy50cw__:
    {
      content:
        "// ❓ Declaring a Task\nimport sleep from '@hatchet-dev/typescript-sdk/util/sleep';\nimport { hatchet } from '../hatchet-client';\n\n// (optional) Define the input type for the workflow\nexport type SimpleInput = {\n  Message: string;\n};\n\n// ❓ Execution Timeout\nexport const withTimeouts = hatchet.task({\n  name: 'with-timeouts',\n  // time the task can wait in the queue before it is cancelled\n  scheduleTimeout: '10s',\n  // time the task can run before it is cancelled\n  executionTimeout: '10s',\n  fn: async (input: SimpleInput, ctx) => {\n    // wait 15 seconds\n    await sleep(15000);\n\n    // get the abort controller\n    const { controller } = ctx;\n\n    // if the abort controller is aborted, throw an error\n    if (controller.signal.aborted) {\n      throw new Error('cancelled');\n    }\n\n    return {\n      TransformedMessage: input.Message.toLowerCase(),\n    };\n  },\n});\n\n// ❓ Refresh Timeout\nexport const refreshTimeout = hatchet.task({\n  name: 'refresh-timeout',\n  executionTimeout: '10s',\n  scheduleTimeout: '10s',\n  fn: async (input: SimpleInput, ctx) => {\n    // adds 15 seconds to the execution timeout\n    ctx.refreshTimeout('15s');\n    await sleep(15000);\n\n    // get the abort controller\n    const { controller } = ctx;\n\n    // now this condition will not be met\n    // if the abort controller is aborted, throw an error\n    if (controller.signal.aborted) {\n      throw new Error('cancelled');\n    }\n\n    return {\n      TransformedMessage: input.Message.toLowerCase(),\n    };\n  },\n});\n",
      language: 'ts',
      source: 'examples/typescript/with_timeouts/workflow.ts',
      highlights: {},
    },
  L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3B5dGhvbi9fX2luaXRfXy5weQ__:
    {
      content: '',
      language: 'py',
      source: 'examples/python/__init__.py',
      highlights: {},
    },
  L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3B5dGhvbi9hZmZpbml0eV93b3JrZXJzL3RyaWdnZXIucHk_:
    {
      content:
        'from examples.affinity_workers.worker import affinity_worker_workflow\nfrom hatchet_sdk import TriggerWorkflowOptions\n\naffinity_worker_workflow.run(\n    options=TriggerWorkflowOptions(additional_metadata={"hello": "moon"}),\n)\n',
      language: 'py',
      source: 'examples/python/affinity_workers/trigger.py',
      highlights: {},
    },
  L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3B5dGhvbi9hZmZpbml0eV93b3JrZXJzL3dvcmtlci5weQ__:
    {
      content:
        'from hatchet_sdk import Context, EmptyModel, Hatchet, WorkerLabelComparator\nfrom hatchet_sdk.labels import DesiredWorkerLabel\n\nhatchet = Hatchet(debug=True)\n\n# ❓ AffinityWorkflow\n\naffinity_worker_workflow = hatchet.workflow(name="AffinityWorkflow")\n\n@affinity_worker_workflow.task(\n    desired_worker_labels={\n        "model": DesiredWorkerLabel(value="fancy-ai-model-v2", weight=10),\n        "memory": DesiredWorkerLabel(\n            value=256,\n            required=True,\n            comparator=WorkerLabelComparator.LESS_THAN,\n        ),\n    },\n)\n\n# ‼️\n\n# ❓ AffinityTask\nasync def step(input: EmptyModel, ctx: Context) -> dict[str, str | None]:\n    if ctx.worker.labels().get("model") != "fancy-ai-model-v2":\n        ctx.worker.upsert_labels({"model": "unset"})\n        # DO WORK TO EVICT OLD MODEL / LOAD NEW MODEL\n        ctx.worker.upsert_labels({"model": "fancy-ai-model-v2"})\n\n    return {"worker": ctx.worker.id()}\n\n# ‼️\n\ndef main() -> None:\n\n    # ❓ AffinityWorker\n    worker = hatchet.worker(\n        "affinity-worker",\n        slots=10,\n        labels={\n            "model": "fancy-ai-model-v2",\n            "memory": 512,\n        },\n        workflows=[affinity_worker_workflow],\n    )\n    worker.start()\n\n# ‼️\n\nif __name__ == "__main__":\n    main()\n',
      language: 'py',
      source: 'examples/python/affinity_workers/worker.py',
      highlights: {},
    },
  L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3B5dGhvbi9hcGkvYXBpLnB5:
    {
      content:
        'from hatchet_sdk import Hatchet\n\nhatchet = Hatchet(debug=True)\n\ndef main() -> None:\n    workflow_list = hatchet.workflows.list()\n    rows = workflow_list.rows or []\n\n    for workflow in rows:\n        print(workflow.name)\n        print(workflow.metadata.id)\n        print(workflow.metadata.created_at)\n        print(workflow.metadata.updated_at)\n\nif __name__ == "__main__":\n    main()\n',
      language: 'py',
      source: 'examples/python/api/api.py',
      highlights: {},
    },
  L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3B5dGhvbi9hcGkvYXN5bmNfYXBpLnB5:
    {
      content:
        'import asyncio\n\nfrom hatchet_sdk import Hatchet\n\nhatchet = Hatchet(debug=True)\n\nasync def main() -> None:\n    workflow_list = await hatchet.workflows.aio_list()\n    rows = workflow_list.rows or []\n\n    for workflow in rows:\n        print(workflow.name)\n        print(workflow.metadata.id)\n        print(workflow.metadata.created_at)\n        print(workflow.metadata.updated_at)\n\nif __name__ == "__main__":\n    asyncio.run(main())\n',
      language: 'py',
      source: 'examples/python/api/async_api.py',
      highlights: {},
    },
  L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3B5dGhvbi9ibG9ja2VkX2FzeW5jL3RyaWdnZXIucHk_:
    {
      content:
        'from examples.blocked_async.worker import blocked_worker_workflow\nfrom hatchet_sdk import TriggerWorkflowOptions\n\nblocked_worker_workflow.run(\n    options=TriggerWorkflowOptions(additional_metadata={"hello": "moon"}),\n)\n',
      language: 'py',
      source: 'examples/python/blocked_async/trigger.py',
      highlights: {},
    },
  L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3B5dGhvbi9ibG9ja2VkX2FzeW5jL3dvcmtlci5weQ__:
    {
      content:
        'import hashlib\nimport time\nfrom datetime import timedelta\n\nfrom hatchet_sdk import Context, EmptyModel, Hatchet\n\nhatchet = Hatchet(debug=True)\n\n# WARNING: this is an example of what NOT to do\n# This workflow is intentionally blocking the main thread\n# and will block the worker from processing other workflows\n#\n# You do not want to run long sync functions in an async def function\n\nblocked_worker_workflow = hatchet.workflow(name="Blocked")\n\n@blocked_worker_workflow.task(execution_timeout=timedelta(seconds=11), retries=3)\nasync def step1(input: EmptyModel, ctx: Context) -> dict[str, str | int | float]:\n    print("Executing step1")\n\n    # CPU-bound task: Calculate a large number of SHA-256 hashes\n    start_time = time.time()\n    iterations = 10_000_000\n    for i in range(iterations):\n        hashlib.sha256(f"data{i}".encode()).hexdigest()\n\n    end_time = time.time()\n    execution_time = end_time - start_time\n\n    print(f"Completed {iterations} hash calculations in {execution_time:.2f} seconds")\n\n    return {\n        "step1": "step1",\n        "iterations": iterations,\n        "execution_time": execution_time,\n    }\n\ndef main() -> None:\n    worker = hatchet.worker(\n        "blocked-worker", slots=3, workflows=[blocked_worker_workflow]\n    )\n    worker.start()\n\nif __name__ == "__main__":\n    main()\n',
      language: 'py',
      source: 'examples/python/blocked_async/worker.py',
      highlights: {},
    },
  L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3B5dGhvbi9idWxrX2Zhbm91dC9idWxrX3RyaWdnZXIucHk_:
    {
      content:
        'import asyncio\n\nfrom examples.bulk_fanout.worker import ParentInput, bulk_parent_wf\nfrom hatchet_sdk import Hatchet\nfrom hatchet_sdk.clients.admin import TriggerWorkflowOptions\n\nhatchet = Hatchet()\n\nasync def main() -> None:\n    results = bulk_parent_wf.run_many(\n        workflows=[\n            bulk_parent_wf.create_bulk_run_item(\n                input=ParentInput(n=i),\n                options=TriggerWorkflowOptions(\n                    additional_metadata={\n                        "bulk-trigger": i,\n                        "hello-{i}": "earth-{i}",\n                    }\n                ),\n            )\n            for i in range(20)\n        ],\n    )\n\n    for result in results:\n        print(result)\n\nif __name__ == "__main__":\n    asyncio.run(main())\n',
      language: 'py',
      source: 'examples/python/bulk_fanout/bulk_trigger.py',
      highlights: {},
    },
  L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3B5dGhvbi9idWxrX2Zhbm91dC9zdHJlYW0ucHk_:
    {
      content:
        'import asyncio\nimport random\n\nfrom examples.bulk_fanout.worker import ParentInput, bulk_parent_wf\nfrom hatchet_sdk import Hatchet\nfrom hatchet_sdk.clients.admin import TriggerWorkflowOptions\n\nasync def main() -> None:\n    hatchet = Hatchet()\n\n    # Generate a random stream key to use to track all\n    # stream events for this workflow run.\n\n    streamKey = "streamKey"\n    streamVal = f"sk-{random.randint(1, 100)}"\n\n    # Specify the stream key as additional metadata\n    # when running the workflow.\n\n    # This key gets propagated to all child workflows\n    # and can have an arbitrary property name.\n    bulk_parent_wf.run(\n        input=ParentInput(n=2),\n        options=TriggerWorkflowOptions(additional_metadata={streamKey: streamVal}),\n    )\n\n    # Stream all events for the additional meta key value\n    listener = hatchet.listener.stream_by_additional_metadata(streamKey, streamVal)\n\n    async for event in listener:\n        print(event.type, event.payload)\n\nif __name__ == "__main__":\n    asyncio.run(main())\n',
      language: 'py',
      source: 'examples/python/bulk_fanout/stream.py',
      highlights: {},
    },
  L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3B5dGhvbi9idWxrX2Zhbm91dC90ZXN0X2J1bGtfZmFub3V0LnB5:
    {
      content:
        'import pytest\n\nfrom examples.bulk_fanout.worker import ParentInput, bulk_parent_wf\n\n@pytest.mark.asyncio(loop_scope="session")\nasync def test_run() -> None:\n    result = await bulk_parent_wf.aio_run(input=ParentInput(n=12))\n\n    assert len(result["spawn"]["results"]) == 12\n',
      language: 'py',
      source: 'examples/python/bulk_fanout/test_bulk_fanout.py',
      highlights: {},
    },
  L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3B5dGhvbi9idWxrX2Zhbm91dC90cmlnZ2VyLnB5:
    {
      content:
        'from examples.bulk_fanout.worker import ParentInput, bulk_parent_wf\nfrom hatchet_sdk import TriggerWorkflowOptions\n\nbulk_parent_wf.run(\n    ParentInput(n=999),\n    TriggerWorkflowOptions(additional_metadata={"no-dedupe": "world"}),\n)\n',
      language: 'py',
      source: 'examples/python/bulk_fanout/trigger.py',
      highlights: {},
    },
  L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3B5dGhvbi9idWxrX2Zhbm91dC93b3JrZXIucHk_:
    {
      content:
        'from datetime import timedelta\nfrom typing import Any\n\nfrom pydantic import BaseModel\n\nfrom hatchet_sdk import Context, Hatchet\nfrom hatchet_sdk.clients.admin import TriggerWorkflowOptions\n\nhatchet = Hatchet(debug=True)\n\nclass ParentInput(BaseModel):\n    n: int = 100\n\nclass ChildInput(BaseModel):\n    a: str\n\nbulk_parent_wf = hatchet.workflow(name="BulkFanoutParent", input_validator=ParentInput)\nbulk_child_wf = hatchet.workflow(name="BulkFanoutChild", input_validator=ChildInput)\n\n# ❓ BulkFanoutParent\n@bulk_parent_wf.task(execution_timeout=timedelta(minutes=5))\nasync def spawn(input: ParentInput, ctx: Context) -> dict[str, list[dict[str, Any]]]:\n    # 👀 Create each workflow run to spawn\n    child_workflow_runs = [\n        bulk_child_wf.create_bulk_run_item(\n            input=ChildInput(a=str(i)),\n            key=f"child{i}",\n            options=TriggerWorkflowOptions(additional_metadata={"hello": "earth"}),\n        )\n        for i in range(input.n)\n    ]\n\n    # 👀 Run workflows in bulk to improve performance\n    spawn_results = await bulk_child_wf.aio_run_many(child_workflow_runs)\n\n    return {"results": spawn_results}\n\n# ‼️\n\n@bulk_child_wf.task()\ndef process(input: ChildInput, ctx: Context) -> dict[str, str]:\n    print(f"child process {input.a}")\n    return {"status": "success " + input.a}\n\n@bulk_child_wf.task()\ndef process2(input: ChildInput, ctx: Context) -> dict[str, str]:\n    print("child process2")\n    return {"status2": "success"}\n\ndef main() -> None:\n    worker = hatchet.worker(\n        "fanout-worker", slots=40, workflows=[bulk_parent_wf, bulk_child_wf]\n    )\n    worker.start()\n\nif __name__ == "__main__":\n    main()\n',
      language: 'py',
      source: 'examples/python/bulk_fanout/worker.py',
      highlights: {},
    },
  L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3B5dGhvbi9idWxrX29wZXJhdGlvbnMvY2FuY2VsLnB5:
    {
      content:
        '# ❓ Setup\n\nfrom datetime import datetime, timedelta\n\nfrom hatchet_sdk import BulkCancelReplayOpts, Hatchet, RunFilter, V1TaskStatus\n\nhatchet = Hatchet()\n\nworkflows = hatchet.workflows.list()\n\nassert workflows.rows\n\nworkflow = workflows.rows[0]\n\n# ❓ List runs\nworkflow_runs = hatchet.runs.list(workflow_ids=[workflow.metadata.id])\n\n# ❓ Cancel by run ids\nworkflow_run_ids = [workflow_run.metadata.id for workflow_run in workflow_runs.rows]\n\nbulk_cancel_by_ids = BulkCancelReplayOpts(ids=workflow_run_ids)\n\nhatchet.runs.bulk_cancel(bulk_cancel_by_ids)\n\n# ❓ Cancel by filters\n\nbulk_cancel_by_filters = BulkCancelReplayOpts(\n    filters=RunFilter(\n        since=datetime.today() - timedelta(days=1),\n        until=datetime.now(),\n        statuses=[V1TaskStatus.RUNNING],\n        workflow_ids=[workflow.metadata.id],\n        additional_metadata={"key": "value"},\n    )\n)\n\nhatchet.runs.bulk_cancel(bulk_cancel_by_filters)\n',
      language: 'py',
      source: 'examples/python/bulk_operations/cancel.py',
      highlights: {},
    },
  L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3B5dGhvbi9idWxrX29wZXJhdGlvbnMvcmVwbGF5LnB5:
    {
      content:
        '# ❓ Setup\n\nfrom datetime import datetime, timedelta\n\nfrom hatchet_sdk import BulkCancelReplayOpts, Hatchet, RunFilter, V1TaskStatus\n\nhatchet = Hatchet()\n\nworkflows = hatchet.workflows.list()\n\nassert workflows.rows\n\nworkflow = workflows.rows[0]\n\n# ❓ List runs\nworkflow_runs = hatchet.runs.list(workflow_ids=[workflow.metadata.id])\n\n# ❓ Replay by run ids\nworkflow_run_ids = [workflow_run.metadata.id for workflow_run in workflow_runs.rows]\n\nbulk_replay_by_ids = BulkCancelReplayOpts(ids=workflow_run_ids)\n\nhatchet.runs.bulk_replay(bulk_replay_by_ids)\n\n# ❓ Replay by filters\nbulk_replay_by_filters = BulkCancelReplayOpts(\n    filters=RunFilter(\n        since=datetime.today() - timedelta(days=1),\n        until=datetime.now(),\n        statuses=[V1TaskStatus.RUNNING],\n        workflow_ids=[workflow.metadata.id],\n        additional_metadata={"key": "value"},\n    )\n)\n\nhatchet.runs.bulk_replay(bulk_replay_by_filters)\n',
      language: 'py',
      source: 'examples/python/bulk_operations/replay.py',
      highlights: {},
    },
  L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3B5dGhvbi9jYW5jZWxsYXRpb24vdGVzdF9jYW5jZWxsYXRpb24ucHk_:
    {
      content:
        'import asyncio\n\nimport pytest\n\nfrom examples.cancellation.worker import cancellation_workflow\nfrom hatchet_sdk import Hatchet\nfrom hatchet_sdk.clients.rest.models.v1_task_status import V1TaskStatus\n\n@pytest.mark.asyncio(loop_scope="session")\nasync def test_cancellation(hatchet: Hatchet) -> None:\n    ref = await cancellation_workflow.aio_run_no_wait()\n\n    """Sleep for a long time since we only need cancellation to happen _eventually_"""\n    await asyncio.sleep(10)\n\n    for i in range(30):\n        run = await hatchet.runs.aio_get(ref.workflow_run_id)\n\n        if run.run.status == V1TaskStatus.RUNNING:\n            await asyncio.sleep(1)\n            continue\n\n        assert run.run.status == V1TaskStatus.CANCELLED\n        assert not run.run.output\n\n        break\n    else:\n        assert False, "Workflow run did not cancel in time"\n',
      language: 'py',
      source: 'examples/python/cancellation/test_cancellation.py',
      highlights: {},
    },
  L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3B5dGhvbi9jYW5jZWxsYXRpb24vdHJpZ2dlci5weQ__:
    {
      content:
        'import time\n\nfrom examples.cancellation.worker import cancellation_workflow, hatchet\n\nid = cancellation_workflow.run_no_wait()\n\ntime.sleep(5)\n\nhatchet.runs.cancel(id.workflow_run_id)\n',
      language: 'py',
      source: 'examples/python/cancellation/trigger.py',
      highlights: {},
    },
  L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3B5dGhvbi9jYW5jZWxsYXRpb24vd29ya2VyLnB5:
    {
      content:
        'import asyncio\nimport time\n\nfrom hatchet_sdk import Context, EmptyModel, Hatchet\n\nhatchet = Hatchet(debug=True)\n\ncancellation_workflow = hatchet.workflow(name="CancelWorkflow")\n\n# ❓ Self-cancelling task\n@cancellation_workflow.task()\nasync def self_cancel(input: EmptyModel, ctx: Context) -> dict[str, str]:\n    await asyncio.sleep(2)\n\n    ## Cancel the task\n    await ctx.aio_cancel()\n\n    await asyncio.sleep(10)\n\n    return {"error": "Task should have been cancelled"}\n\n# ❓ Checking exit flag\n@cancellation_workflow.task()\ndef check_flag(input: EmptyModel, ctx: Context) -> dict[str, str]:\n    for i in range(3):\n        time.sleep(1)\n\n        # Note: Checking the status of the exit flag is mostly useful for cancelling\n        # sync tasks without needing to forcibly kill the thread they\'re running on.\n        if ctx.exit_flag:\n            print("Task has been cancelled")\n            raise ValueError("Task has been cancelled")\n\n    return {"error": "Task should have been cancelled"}\n\ndef main() -> None:\n    worker = hatchet.worker("cancellation-worker", workflows=[cancellation_workflow])\n    worker.start()\n\nif __name__ == "__main__":\n    main()\n',
      language: 'py',
      source: 'examples/python/cancellation/worker.py',
      highlights: {},
    },
  L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3B5dGhvbi9jaGlsZC9idWxrLnB5:
    {
      content:
        'import asyncio\n\n# ❓ Running a Task\nfrom examples.child.worker import SimpleInput, child_task\n\nchild_task.run(SimpleInput(message="Hello, World!"))\n\nasync def main() -> None:\n    # ❓ Bulk Run a Task\n    greetings = ["Hello, World!", "Hello, Moon!", "Hello, Mars!"]\n\n    results = await child_task.aio_run_many(\n        [\n            # run each greeting as a task in parallel\n            child_task.create_bulk_run_item(\n                input=SimpleInput(message=greeting),\n            )\n            for greeting in greetings\n        ]\n    )\n\n    # this will await all results and return a list of results\n    print(results)\n\n    # ❓ Running Multiple Tasks\n    result1 = child_task.aio_run(SimpleInput(message="Hello, World!"))\n    result2 = child_task.aio_run(SimpleInput(message="Hello, Moon!"))\n\n    #  gather the results of the two tasks\n    gather_results = await asyncio.gather(result1, result2)\n\n    #  print the results of the two tasks\n    print(gather_results[0]["transformed_message"])\n    print(gather_results[1]["transformed_message"])\n',
      language: 'py',
      source: 'examples/python/child/bulk.py',
      highlights: {},
    },
  L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3B5dGhvbi9jaGlsZC9zaW1wbGUtZmFub3V0LnB5:
    {
      content:
        'from typing import Any\n\nfrom examples.child.worker import SimpleInput, child_task\nfrom hatchet_sdk.context.context import Context\nfrom hatchet_sdk.hatchet import Hatchet\nfrom hatchet_sdk.runnables.types import EmptyModel\n\nhatchet = Hatchet(debug=True)\n\n# ❓ Running a Task from within a Task\n@hatchet-dev/typescript-sdk.task(name="SpawnTask")\nasync def spawn(input: EmptyModel, ctx: Context) -> dict[str, Any]:\n    # Simply run the task with the input we received\n    result = await child_task.aio_run(\n        input=SimpleInput(message="Hello, World!"),\n    )\n\n    return {"results": result}\n\n# ‼️\n',
      language: 'py',
      source: 'examples/python/child/simple-fanout.py',
      highlights: {},
    },
  L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3B5dGhvbi9jaGlsZC90cmlnZ2VyLnB5:
    {
      content:
        '# ruff: noqa: E402\n\nimport asyncio\n\n# ❓ Running a Task\nfrom examples.child.worker import SimpleInput, child_task\n\nchild_task.run(SimpleInput(message="Hello, World!"))\n\n# ❓ Schedule a Task\nfrom datetime import datetime, timedelta\n\nchild_task.schedule(\n    datetime.now() + timedelta(minutes=5), SimpleInput(message="Hello, World!")\n)\n\nasync def main() -> None:\n    # ❓ Running a Task AIO\n    result = await child_task.aio_run(SimpleInput(message="Hello, World!"))\n\n    print(result)\n\n    # ❓ Running Multiple Tasks\n    result1 = child_task.aio_run(SimpleInput(message="Hello, World!"))\n    result2 = child_task.aio_run(SimpleInput(message="Hello, Moon!"))\n\n    #  gather the results of the two tasks\n    results = await asyncio.gather(result1, result2)\n\n    #  print the results of the two tasks\n    print(results[0]["transformed_message"])\n    print(results[1]["transformed_message"])\n',
      language: 'py',
      source: 'examples/python/child/trigger.py',
      highlights: {},
    },
  L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3B5dGhvbi9jaGlsZC93b3JrZXIucHk_:
    {
      content:
        '# ❓ Simple\n\nfrom pydantic import BaseModel\n\nfrom hatchet_sdk import Context, Hatchet\n\nhatchet = Hatchet(debug=True)\n\nclass SimpleInput(BaseModel):\n    message: str\n\nclass SimpleOutput(BaseModel):\n    transformed_message: str\n\nchild_task = hatchet.workflow(name="SimpleWorkflow", input_validator=SimpleInput)\n\n@child_task.task(name="step1")\ndef step1(input: SimpleInput, ctx: Context) -> SimpleOutput:\n    print("executed step1: ", input.message)\n    return SimpleOutput(transformed_message=input.message.upper())\n\n# ‼️\n\ndef main() -> None:\n    worker = hatchet.worker("test-worker", slots=1, workflows=[child_task])\n    worker.start()\n\nif __name__ == "__main__":\n    main()\n',
      language: 'py',
      source: 'examples/python/child/worker.py',
      highlights: {},
    },
  L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3B5dGhvbi9jb25jdXJyZW5jeV9saW1pdC90ZXN0X2NvbmN1cnJlbmN5X2xpbWl0LnB5:
    {
      content:
        'import pytest\n\nfrom examples.concurrency_limit.worker import WorkflowInput, concurrency_limit_workflow\nfrom hatchet_sdk.workflow_run import WorkflowRunRef\n\n@pytest.mark.asyncio(loop_scope="session")\n@pytest.mark.skip(reason="The timing for this test is not reliable")\nasync def test_run() -> None:\n    num_runs = 6\n    runs: list[WorkflowRunRef] = []\n\n    # Start all runs\n    for i in range(1, num_runs + 1):\n        run = concurrency_limit_workflow.run_no_wait(\n            WorkflowInput(run=i, group_key=str(i))\n        )\n        runs.append(run)\n\n    # Wait for all results\n    successful_runs = []\n    cancelled_runs = []\n\n    # Process each run individually\n    for i, run in enumerate(runs, start=1):\n        try:\n            result = await run.aio_result()\n            successful_runs.append((i, result))\n        except Exception as e:\n            if "CANCELLED_BY_CONCURRENCY_LIMIT" in str(e):\n                cancelled_runs.append((i, str(e)))\n            else:\n                raise  # Re-raise if it\'s an unexpected error\n\n    # Check that we have the correct number of successful and cancelled runs\n    assert (\n        len(successful_runs) == 5\n    ), f"Expected 5 successful runs, got {len(successful_runs)}"\n    assert (\n        len(cancelled_runs) == 1\n    ), f"Expected 1 cancelled run, got {len(cancelled_runs)}"\n',
      language: 'py',
      source: 'examples/python/concurrency_limit/test_concurrency_limit.py',
      highlights: {},
    },
  L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3B5dGhvbi9jb25jdXJyZW5jeV9saW1pdC90cmlnZ2VyLnB5:
    {
      content:
        'from examples.concurrency_limit.worker import WorkflowInput, concurrency_limit_workflow\n\nconcurrency_limit_workflow.run(WorkflowInput(group_key="test", run=1))\n',
      language: 'py',
      source: 'examples/python/concurrency_limit/trigger.py',
      highlights: {},
    },
  L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3B5dGhvbi9jb25jdXJyZW5jeV9saW1pdC93b3JrZXIucHk_:
    {
      content:
        'import time\nfrom typing import Any\n\nfrom pydantic import BaseModel\n\nfrom hatchet_sdk import (\n    ConcurrencyExpression,\n    ConcurrencyLimitStrategy,\n    Context,\n    Hatchet,\n)\n\nhatchet = Hatchet(debug=True)\n\n# ❓ Workflow\nclass WorkflowInput(BaseModel):\n    run: int\n    group_key: str\n\nconcurrency_limit_workflow = hatchet.workflow(\n    name="ConcurrencyDemoWorkflow",\n    concurrency=ConcurrencyExpression(\n        expression="input.group_key",\n        max_runs=5,\n        limit_strategy=ConcurrencyLimitStrategy.CANCEL_IN_PROGRESS,\n    ),\n    input_validator=WorkflowInput,\n)\n\n# ‼️\n\n@concurrency_limit_workflow.task()\ndef step1(input: WorkflowInput, ctx: Context) -> dict[str, Any]:\n    time.sleep(3)\n    print("executed step1")\n    return {"run": input.run}\n\ndef main() -> None:\n    worker = hatchet.worker(\n        "concurrency-demo-worker", slots=10, workflows=[concurrency_limit_workflow]\n    )\n\n    worker.start()\n\nif __name__ == "__main__":\n    main()\n',
      language: 'py',
      source: 'examples/python/concurrency_limit/worker.py',
      highlights: {},
    },
  L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3B5dGhvbi9jb25jdXJyZW5jeV9saW1pdF9yci90ZXN0X2NvbmN1cnJlbmN5X2xpbWl0X3JyLnB5:
    {
      content:
        'import time\n\nimport pytest\n\nfrom examples.concurrency_limit_rr.worker import concurrency_limit_rr_workflow\nfrom hatchet_sdk.workflow_run import WorkflowRunRef\n\n@pytest.mark.skip(reason="The timing for this test is not reliable")\n@pytest.mark.asyncio(loop_scope="session")\nasync def test_run() -> None:\n    num_groups = 2\n    runs: list[WorkflowRunRef] = []\n\n    # Start all runs\n    for i in range(1, num_groups + 1):\n        run = concurrency_limit_rr_workflow.run_no_wait()\n        runs.append(run)\n        run = concurrency_limit_rr_workflow.run_no_wait()\n        runs.append(run)\n\n    # Wait for all results\n    successful_runs = []\n    cancelled_runs = []\n\n    start_time = time.time()\n\n    # Process each run individually\n    for i, run in enumerate(runs, start=1):\n        try:\n            result = await run.aio_result()\n            successful_runs.append((i, result))\n        except Exception as e:\n            if "CANCELLED_BY_CONCURRENCY_LIMIT" in str(e):\n                cancelled_runs.append((i, str(e)))\n            else:\n                raise  # Re-raise if it\'s an unexpected error\n\n    end_time = time.time()\n    total_time = end_time - start_time\n\n    # Check that we have the correct number of successful and cancelled runs\n    assert (\n        len(successful_runs) == 4\n    ), f"Expected 4 successful runs, got {len(successful_runs)}"\n    assert (\n        len(cancelled_runs) == 0\n    ), f"Expected 0 cancelled run, got {len(cancelled_runs)}"\n\n    # Check that the total time is close to 2 seconds\n    assert (\n        3.8 <= total_time <= 7\n    ), f"Expected runtime to be about 4 seconds, but it took {total_time:.2f} seconds"\n\n    print(f"Total execution time: {total_time:.2f} seconds")\n',
      language: 'py',
      source:
        'examples/python/concurrency_limit_rr/test_concurrency_limit_rr.py',
      highlights: {},
    },
  L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3B5dGhvbi9jb25jdXJyZW5jeV9saW1pdF9yci90cmlnZ2VyLnB5:
    {
      content:
        'from examples.concurrency_limit_rr.worker import (\n    WorkflowInput,\n    concurrency_limit_rr_workflow,\n)\nfrom hatchet_sdk import Hatchet\n\nhatchet = Hatchet()\n\nfor i in range(200):\n    group = "0"\n\n    if i % 2 == 0:\n        group = "1"\n\n    concurrency_limit_rr_workflow.run(WorkflowInput(group=group))\n',
      language: 'py',
      source: 'examples/python/concurrency_limit_rr/trigger.py',
      highlights: {},
    },
  L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3B5dGhvbi9jb25jdXJyZW5jeV9saW1pdF9yci93b3JrZXIucHk_:
    {
      content:
        'import time\n\nfrom pydantic import BaseModel\n\nfrom hatchet_sdk import (\n    ConcurrencyExpression,\n    ConcurrencyLimitStrategy,\n    Context,\n    Hatchet,\n)\n\nhatchet = Hatchet(debug=True)\n\n# ❓ Concurrency Strategy With Key\nclass WorkflowInput(BaseModel):\n    group: str\n\nconcurrency_limit_rr_workflow = hatchet.workflow(\n    name="ConcurrencyDemoWorkflowRR",\n    concurrency=ConcurrencyExpression(\n        expression="input.group",\n        max_runs=1,\n        limit_strategy=ConcurrencyLimitStrategy.GROUP_ROUND_ROBIN,\n    ),\n    input_validator=WorkflowInput,\n)\n# ‼️\n\n@concurrency_limit_rr_workflow.task()\ndef step1(input: WorkflowInput, ctx: Context) -> None:\n    print("starting step1")\n    time.sleep(2)\n    print("finished step1")\n    pass\n\ndef main() -> None:\n    worker = hatchet.worker(\n        "concurrency-demo-worker-rr",\n        slots=10,\n        workflows=[concurrency_limit_rr_workflow],\n    )\n\n    worker.start()\n\nif __name__ == "__main__":\n    main()\n',
      language: 'py',
      source: 'examples/python/concurrency_limit_rr/worker.py',
      highlights: {},
    },
  L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3B5dGhvbi9jb25jdXJyZW5jeV9saW1pdF9ycl9sb2FkL2V2ZW50LnB5:
    {
      content:
        'import random\n\nfrom hatchet_sdk import Hatchet\n\nhatchet = Hatchet(debug=True)\n\n# Create a list of events with desired distribution\nevents = ["1"] * 10000 + ["0"] * 100\nrandom.shuffle(events)\n\n# Send the shuffled events\nfor group in events:\n    hatchet.event.push("concurrency-test", {"group": group})\n',
      language: 'py',
      source: 'examples/python/concurrency_limit_rr_load/event.py',
      highlights: {},
    },
  L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3B5dGhvbi9jb25jdXJyZW5jeV9saW1pdF9ycl9sb2FkL3dvcmtlci5weQ__:
    {
      content:
        'import random\nimport time\n\nfrom pydantic import BaseModel\n\nfrom hatchet_sdk import (\n    ConcurrencyExpression,\n    ConcurrencyLimitStrategy,\n    Context,\n    Hatchet,\n)\n\nhatchet = Hatchet(debug=True)\n\nclass LoadRRInput(BaseModel):\n    group: str\n\nload_rr_workflow = hatchet.workflow(\n    name="LoadRoundRobin",\n    on_events=["concurrency-test"],\n    concurrency=ConcurrencyExpression(\n        expression="input.group",\n        max_runs=1,\n        limit_strategy=ConcurrencyLimitStrategy.GROUP_ROUND_ROBIN,\n    ),\n    input_validator=LoadRRInput,\n)\n\n@load_rr_workflow.on_failure_task()\ndef on_failure(input: LoadRRInput, context: Context) -> dict[str, str]:\n    print("on_failure")\n    return {"on_failure": "on_failure"}\n\n@load_rr_workflow.task()\ndef step1(input: LoadRRInput, context: Context) -> dict[str, str]:\n    print("starting step1")\n    time.sleep(random.randint(2, 20))\n    print("finished step1")\n    return {"step1": "step1"}\n\n@load_rr_workflow.task(\n    retries=3,\n    backoff_factor=5,\n    backoff_max_seconds=60,\n)\ndef step2(sinput: LoadRRInput, context: Context) -> dict[str, str]:\n    print("starting step2")\n    if random.random() < 0.5:  # 1% chance of failure\n        raise Exception("Random failure in step2")\n    time.sleep(2)\n    print("finished step2")\n    return {"step2": "step2"}\n\n@load_rr_workflow.task()\ndef step3(input: LoadRRInput, context: Context) -> dict[str, str]:\n    print("starting step3")\n    time.sleep(0.2)\n    print("finished step3")\n    return {"step3": "step3"}\n\ndef main() -> None:\n    worker = hatchet.worker(\n        "concurrency-demo-worker-rr", slots=50, workflows=[load_rr_workflow]\n    )\n\n    worker.start()\n\nif __name__ == "__main__":\n    main()\n',
      language: 'py',
      source: 'examples/python/concurrency_limit_rr_load/worker.py',
      highlights: {},
    },
  L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3B5dGhvbi9jb25jdXJyZW5jeV9tdWx0aXBsZV9rZXlzL3Rlc3RfbXVsdGlwbGVfY29uY3VycmVuY3lfa2V5cy5weQ__:
    {
      content:
        'import asyncio\nfrom collections import Counter\nfrom datetime import datetime\nfrom random import choice\nfrom typing import Literal\nfrom uuid import uuid4\n\nimport pytest\nfrom pydantic import BaseModel\n\nfrom examples.concurrency_multiple_keys.worker import (\n    DIGIT_MAX_RUNS,\n    NAME_MAX_RUNS,\n    WorkflowInput,\n    concurrency_multiple_keys_workflow,\n)\nfrom hatchet_sdk import Hatchet, TriggerWorkflowOptions\nfrom hatchet_sdk.clients.rest.models.v1_task_summary import V1TaskSummary\n\nCharacter = Literal["Anna", "Vronsky", "Stiva", "Dolly", "Levin", "Karenin"]\ncharacters: list[Character] = [\n    "Anna",\n    "Vronsky",\n    "Stiva",\n    "Dolly",\n    "Levin",\n    "Karenin",\n]\n\nclass RunMetadata(BaseModel):\n    test_run_id: str\n    key: str\n    name: Character\n    digit: str\n    started_at: datetime\n    finished_at: datetime\n\n    @staticmethod\n    def parse(task: V1TaskSummary) -> "RunMetadata":\n        return RunMetadata(\n            test_run_id=task.additional_metadata["test_run_id"],  # type: ignore\n            key=task.additional_metadata["key"],  # type: ignore\n            name=task.additional_metadata["name"],  # type: ignore\n            digit=task.additional_metadata["digit"],  # type: ignore\n            started_at=task.started_at or datetime.max,\n            finished_at=task.finished_at or datetime.min,\n        )\n\n    def __str__(self) -> str:\n        return self.key\n\n@pytest.mark.asyncio(loop_scope="session")\nasync def test_multi_concurrency_key(hatchet: Hatchet) -> None:\n    test_run_id = str(uuid4())\n\n    run_refs = await concurrency_multiple_keys_workflow.aio_run_many_no_wait(\n        [\n            concurrency_multiple_keys_workflow.create_bulk_run_item(\n                WorkflowInput(\n                    name=(name := choice(characters)),\n                    digit=(digit := choice([str(i) for i in range(6)])),\n                ),\n                options=TriggerWorkflowOptions(\n                    additional_metadata={\n                        "test_run_id": test_run_id,\n                        "key": f"{name}-{digit}",\n                        "name": name,\n                        "digit": digit,\n                    },\n                ),\n            )\n            for _ in range(100)\n        ]\n    )\n\n    await asyncio.gather(*[r.aio_result() for r in run_refs])\n\n    workflows = (\n        await hatchet.workflows.aio_list(\n            workflow_name=concurrency_multiple_keys_workflow.name,\n            limit=1_000,\n        )\n    ).rows\n\n    assert workflows\n\n    workflow = next(\n        (w for w in workflows if w.name == concurrency_multiple_keys_workflow.name),\n        None,\n    )\n\n    assert workflow\n\n    assert workflow.name == concurrency_multiple_keys_workflow.name\n\n    runs = await hatchet.runs.aio_list(\n        workflow_ids=[workflow.metadata.id],\n        additional_metadata={\n            "test_run_id": test_run_id,\n        },\n        limit=1_000,\n    )\n\n    sorted_runs = sorted(\n        [RunMetadata.parse(r) for r in runs.rows], key=lambda r: r.started_at\n    )\n\n    overlapping_groups: dict[int, list[RunMetadata]] = {}\n\n    for run in sorted_runs:\n        has_group_membership = False\n\n        if not overlapping_groups:\n            overlapping_groups[1] = [run]\n            continue\n\n        if has_group_membership:\n            continue\n\n        for id, group in overlapping_groups.items():\n            if all(are_overlapping(run, task) for task in group):\n                overlapping_groups[id].append(run)\n                has_group_membership = True\n                break\n\n        if not has_group_membership:\n            overlapping_groups[len(overlapping_groups) + 1] = [run]\n\n    assert {s.key for s in sorted_runs} == {\n        k.key for v in overlapping_groups.values() for k in v\n    }\n\n    for id, group in overlapping_groups.items():\n        assert is_valid_group(group), f"Group {id} is not valid"\n\ndef are_overlapping(x: RunMetadata, y: RunMetadata) -> bool:\n    return (x.started_at < y.finished_at and x.finished_at > y.started_at) or (\n        x.finished_at > y.started_at and x.started_at < y.finished_at\n    )\n\ndef is_valid_group(group: list[RunMetadata]) -> bool:\n    digits = Counter[str]()\n    names = Counter[str]()\n\n    for task in group:\n        digits[task.digit] += 1\n        names[task.name] += 1\n\n    if any(v > DIGIT_MAX_RUNS for v in digits.values()):\n        return False\n\n    if any(v > NAME_MAX_RUNS for v in names.values()):\n        return False\n\n    return True\n',
      language: 'py',
      source:
        'examples/python/concurrency_multiple_keys/test_multiple_concurrency_keys.py',
      highlights: {},
    },
  L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3B5dGhvbi9jb25jdXJyZW5jeV9tdWx0aXBsZV9rZXlzL3dvcmtlci5weQ__:
    {
      content:
        'import asyncio\n\nfrom pydantic import BaseModel\n\nfrom hatchet_sdk import (\n    ConcurrencyExpression,\n    ConcurrencyLimitStrategy,\n    Context,\n    Hatchet,\n)\n\nhatchet = Hatchet(debug=True)\n\nSLEEP_TIME = 2\nDIGIT_MAX_RUNS = 8\nNAME_MAX_RUNS = 3\n\n# ❓ Concurrency Strategy With Key\nclass WorkflowInput(BaseModel):\n    name: str\n    digit: str\n\nconcurrency_multiple_keys_workflow = hatchet.workflow(\n    name="ConcurrencyWorkflowManyKeys",\n    input_validator=WorkflowInput,\n)\n# ‼️\n\n@concurrency_multiple_keys_workflow.task(\n    concurrency=[\n        ConcurrencyExpression(\n            expression="input.digit",\n            max_runs=DIGIT_MAX_RUNS,\n            limit_strategy=ConcurrencyLimitStrategy.GROUP_ROUND_ROBIN,\n        ),\n        ConcurrencyExpression(\n            expression="input.name",\n            max_runs=NAME_MAX_RUNS,\n            limit_strategy=ConcurrencyLimitStrategy.GROUP_ROUND_ROBIN,\n        ),\n    ]\n)\nasync def concurrency_task(input: WorkflowInput, ctx: Context) -> None:\n    await asyncio.sleep(SLEEP_TIME)\n\ndef main() -> None:\n    worker = hatchet.worker(\n        "concurrency-worker-multiple-keys",\n        slots=10,\n        workflows=[concurrency_multiple_keys_workflow],\n    )\n\n    worker.start()\n\nif __name__ == "__main__":\n    main()\n',
      language: 'py',
      source: 'examples/python/concurrency_multiple_keys/worker.py',
      highlights: {},
    },
  L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3B5dGhvbi9jb25jdXJyZW5jeV93b3JrZmxvd19sZXZlbC90ZXN0X3dvcmtmbG93X2xldmVsX2NvbmN1cnJlbmN5LnB5:
    {
      content:
        'import asyncio\nfrom collections import Counter\nfrom datetime import datetime\nfrom random import choice\nfrom typing import Literal\nfrom uuid import uuid4\n\nimport pytest\nfrom pydantic import BaseModel\n\nfrom examples.concurrency_workflow_level.worker import (\n    DIGIT_MAX_RUNS,\n    NAME_MAX_RUNS,\n    WorkflowInput,\n    concurrency_workflow_level_workflow,\n)\nfrom hatchet_sdk import Hatchet, TriggerWorkflowOptions\nfrom hatchet_sdk.clients.rest.models.v1_task_summary import V1TaskSummary\n\nCharacter = Literal["Anna", "Vronsky", "Stiva", "Dolly", "Levin", "Karenin"]\ncharacters: list[Character] = [\n    "Anna",\n    "Vronsky",\n    "Stiva",\n    "Dolly",\n    "Levin",\n    "Karenin",\n]\n\nclass RunMetadata(BaseModel):\n    test_run_id: str\n    key: str\n    name: Character\n    digit: str\n    started_at: datetime\n    finished_at: datetime\n\n    @staticmethod\n    def parse(task: V1TaskSummary) -> "RunMetadata":\n        return RunMetadata(\n            test_run_id=task.additional_metadata["test_run_id"],  # type: ignore\n            key=task.additional_metadata["key"],  # type: ignore\n            name=task.additional_metadata["name"],  # type: ignore\n            digit=task.additional_metadata["digit"],  # type: ignore\n            started_at=task.started_at or datetime.max,\n            finished_at=task.finished_at or datetime.min,\n        )\n\n    def __str__(self) -> str:\n        return self.key\n\n@pytest.mark.asyncio()\nasync def test_workflow_level_concurrency(hatchet: Hatchet) -> None:\n    test_run_id = str(uuid4())\n\n    run_refs = await concurrency_workflow_level_workflow.aio_run_many_no_wait(\n        [\n            concurrency_workflow_level_workflow.create_bulk_run_item(\n                WorkflowInput(\n                    name=(name := choice(characters)),\n                    digit=(digit := choice([str(i) for i in range(6)])),\n                ),\n                options=TriggerWorkflowOptions(\n                    additional_metadata={\n                        "test_run_id": test_run_id,\n                        "key": f"{name}-{digit}",\n                        "name": name,\n                        "digit": digit,\n                    },\n                ),\n            )\n            for _ in range(100)\n        ]\n    )\n\n    await asyncio.gather(*[r.aio_result() for r in run_refs])\n\n    workflows = (\n        await hatchet.workflows.aio_list(\n            workflow_name=concurrency_workflow_level_workflow.name,\n            limit=1_000,\n        )\n    ).rows\n\n    assert workflows\n\n    workflow = next(\n        (w for w in workflows if w.name == concurrency_workflow_level_workflow.name),\n        None,\n    )\n\n    assert workflow\n\n    assert workflow.name == concurrency_workflow_level_workflow.name\n\n    runs = await hatchet.runs.aio_list(\n        workflow_ids=[workflow.metadata.id],\n        additional_metadata={\n            "test_run_id": test_run_id,\n        },\n        limit=1_000,\n    )\n\n    sorted_runs = sorted(\n        [RunMetadata.parse(r) for r in runs.rows], key=lambda r: r.started_at\n    )\n\n    overlapping_groups: dict[int, list[RunMetadata]] = {}\n\n    for run in sorted_runs:\n        has_group_membership = False\n\n        if not overlapping_groups:\n            overlapping_groups[1] = [run]\n            continue\n\n        if has_group_membership:\n            continue\n\n        for id, group in overlapping_groups.items():\n            if all(are_overlapping(run, task) for task in group):\n                overlapping_groups[id].append(run)\n                has_group_membership = True\n                break\n\n        if not has_group_membership:\n            overlapping_groups[len(overlapping_groups) + 1] = [run]\n\n    for id, group in overlapping_groups.items():\n        assert is_valid_group(group), f"Group {id} is not valid"\n\ndef are_overlapping(x: RunMetadata, y: RunMetadata) -> bool:\n    return (x.started_at < y.finished_at and x.finished_at > y.started_at) or (\n        x.finished_at > y.started_at and x.started_at < y.finished_at\n    )\n\ndef is_valid_group(group: list[RunMetadata]) -> bool:\n    digits = Counter[str]()\n    names = Counter[str]()\n\n    for task in group:\n        digits[task.digit] += 1\n        names[task.name] += 1\n\n    if any(v > DIGIT_MAX_RUNS for v in digits.values()):\n        return False\n\n    if any(v > NAME_MAX_RUNS for v in names.values()):\n        return False\n\n    return True\n',
      language: 'py',
      source:
        'examples/python/concurrency_workflow_level/test_workflow_level_concurrency.py',
      highlights: {},
    },
  L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3B5dGhvbi9jb25jdXJyZW5jeV93b3JrZmxvd19sZXZlbC93b3JrZXIucHk_:
    {
      content:
        'import asyncio\n\nfrom pydantic import BaseModel\n\nfrom hatchet_sdk import (\n    ConcurrencyExpression,\n    ConcurrencyLimitStrategy,\n    Context,\n    Hatchet,\n)\n\nhatchet = Hatchet(debug=True)\n\nSLEEP_TIME = 2\nDIGIT_MAX_RUNS = 8\nNAME_MAX_RUNS = 3\n\n# ❓ Multiple Concurrency Keys\nclass WorkflowInput(BaseModel):\n    name: str\n    digit: str\n\nconcurrency_workflow_level_workflow = hatchet.workflow(\n    name="ConcurrencyWorkflowManyKeys",\n    input_validator=WorkflowInput,\n    concurrency=[\n        ConcurrencyExpression(\n            expression="input.digit",\n            max_runs=DIGIT_MAX_RUNS,\n            limit_strategy=ConcurrencyLimitStrategy.GROUP_ROUND_ROBIN,\n        ),\n        ConcurrencyExpression(\n            expression="input.name",\n            max_runs=NAME_MAX_RUNS,\n            limit_strategy=ConcurrencyLimitStrategy.GROUP_ROUND_ROBIN,\n        ),\n    ],\n)\n# ‼️\n\n@concurrency_workflow_level_workflow.task()\nasync def task_1(input: WorkflowInput, ctx: Context) -> None:\n    await asyncio.sleep(SLEEP_TIME)\n\n@concurrency_workflow_level_workflow.task()\nasync def task_2(input: WorkflowInput, ctx: Context) -> None:\n    await asyncio.sleep(SLEEP_TIME)\n\ndef main() -> None:\n    worker = hatchet.worker(\n        "concurrency-worker-workflow-level",\n        slots=10,\n        workflows=[concurrency_workflow_level_workflow],\n    )\n\n    worker.start()\n\nif __name__ == "__main__":\n    main()\n',
      language: 'py',
      source: 'examples/python/concurrency_workflow_level/worker.py',
      highlights: {},
    },
  L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3B5dGhvbi9jcm9uL3Byb2dyYW1hdGljLWFzeW5jLnB5:
    {
      content:
        'from pydantic import BaseModel\n\nfrom hatchet_sdk import Hatchet\n\nhatchet = Hatchet()\n\nclass DynamicCronInput(BaseModel):\n    name: str\n\nasync def create_cron() -> None:\n    dynamic_cron_workflow = hatchet.workflow(\n        name="CronWorkflow", input_validator=DynamicCronInput\n    )\n\n    # ❓ Create\n    cron_trigger = await dynamic_cron_workflow.aio_create_cron(\n        cron_name="customer-a-daily-report",\n        expression="0 12 * * *",\n        input=DynamicCronInput(name="John Doe"),\n        additional_metadata={\n            "customer_id": "customer-a",\n        },\n    )\n\n    cron_trigger.metadata.id  # the id of the cron trigger\n\n    # ❓ List\n    await hatchet.cron.aio_list()\n\n    # ❓ Get\n    cron_trigger = await hatchet.cron.aio_get(cron_id=cron_trigger.metadata.id)\n\n    # ❓ Delete\n    await hatchet.cron.aio_delete(cron_id=cron_trigger.metadata.id)\n',
      language: 'py',
      source: 'examples/python/cron/programatic-async.py',
      highlights: {},
    },
  L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3B5dGhvbi9jcm9uL3Byb2dyYW1hdGljLXN5bmMucHk_:
    {
      content:
        'from pydantic import BaseModel\n\nfrom hatchet_sdk import Hatchet\n\nhatchet = Hatchet()\n\nclass DynamicCronInput(BaseModel):\n    name: str\n\ndynamic_cron_workflow = hatchet.workflow(\n    name="CronWorkflow", input_validator=DynamicCronInput\n)\n\n# ❓ Create\ncron_trigger = dynamic_cron_workflow.create_cron(\n    cron_name="customer-a-daily-report",\n    expression="0 12 * * *",\n    input=DynamicCronInput(name="John Doe"),\n    additional_metadata={\n        "customer_id": "customer-a",\n    },\n)\n\nid = cron_trigger.metadata.id  # the id of the cron trigger\n\n# ❓ List\ncron_triggers = hatchet.cron.list()\n\n# ❓ Get\ncron_trigger = hatchet.cron.get(cron_id=cron_trigger.metadata.id)\n\n# ❓ Delete\nhatchet.cron.delete(cron_id=cron_trigger.metadata.id)\n',
      language: 'py',
      source: 'examples/python/cron/programatic-sync.py',
      highlights: {},
    },
  L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3B5dGhvbi9jcm9uL3dvcmtmbG93LWRlZmluaXRpb24ucHk_:
    {
      content:
        'from hatchet_sdk import Context, EmptyModel, Hatchet\n\nhatchet = Hatchet(debug=True)\n\n# ❓ Workflow Definition Cron Trigger\n# Adding a cron trigger to a workflow is as simple\n# as adding a `cron expression` to the `on_cron`\n# prop of the workflow definition\n\ncron_workflow = hatchet.workflow(name="CronWorkflow", on_crons=["* * * * *"])\n\n@cron_workflow.task()\ndef step1(input: EmptyModel, ctx: Context) -> dict[str, str]:\n    return {\n        "time": "step1",\n    }\n\ndef main() -> None:\n    worker = hatchet.worker("test-worker", slots=1, workflows=[cron_workflow])\n    worker.start()\n\nif __name__ == "__main__":\n    main()\n',
      language: 'py',
      source: 'examples/python/cron/workflow-definition.py',
      highlights: {},
    },
  L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3B5dGhvbi9kYWcvdGVzdF9kYWcucHk_:
    {
      content:
        'import pytest\n\nfrom examples.dag.worker import dag_workflow\nfrom hatchet_sdk import Hatchet\n\n@pytest.mark.asyncio(loop_scope="session")\nasync def test_run(hatchet: Hatchet) -> None:\n    result = await dag_workflow.aio_run()\n\n    one = result["step1"]["random_number"]\n    two = result["step2"]["random_number"]\n    assert result["step3"]["sum"] == one + two\n    assert result["step4"]["step4"] == "step4"\n',
      language: 'py',
      source: 'examples/python/dag/test_dag.py',
      highlights: {},
    },
  L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3B5dGhvbi9kYWcvdHJpZ2dlci5weQ__:
    {
      content:
        'from examples.dag.worker import dag_workflow\n\ndag_workflow.run()\n',
      language: 'py',
      source: 'examples/python/dag/trigger.py',
      highlights: {},
    },
  L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3B5dGhvbi9kYWcvd29ya2VyLnB5:
    {
      content:
        'import random\nimport time\nfrom datetime import timedelta\n\nfrom pydantic import BaseModel\n\nfrom hatchet_sdk import Context, EmptyModel, Hatchet\n\nclass StepOutput(BaseModel):\n    random_number: int\n\nclass RandomSum(BaseModel):\n    sum: int\n\nhatchet = Hatchet(debug=True)\n\ndag_workflow = hatchet.workflow(name="DAGWorkflow")\n\n@dag_workflow.task(execution_timeout=timedelta(seconds=5))\ndef step1(input: EmptyModel, ctx: Context) -> StepOutput:\n    return StepOutput(random_number=random.randint(1, 100))\n\n@dag_workflow.task(execution_timeout=timedelta(seconds=5))\nasync def step2(input: EmptyModel, ctx: Context) -> StepOutput:\n    return StepOutput(random_number=random.randint(1, 100))\n\n@dag_workflow.task(parents=[step1, step2])\nasync def step3(input: EmptyModel, ctx: Context) -> RandomSum:\n    one = ctx.task_output(step1).random_number\n    two = (await ctx.task_output(step2)).random_number\n\n    return RandomSum(sum=one + two)\n\n@dag_workflow.task(parents=[step1, step3])\nasync def step4(input: EmptyModel, ctx: Context) -> dict[str, str]:\n    print(\n        "executed step4",\n        time.strftime("%H:%M:%S", time.localtime()),\n        input,\n        ctx.task_output(step1),\n        await ctx.task_output(step3),\n    )\n    return {\n        "step4": "step4",\n    }\n\ndef main() -> None:\n    worker = hatchet.worker("dag-worker", workflows=[dag_workflow])\n\n    worker.start()\n\nif __name__ == "__main__":\n    main()\n',
      language: 'py',
      source: 'examples/python/dag/worker.py',
      highlights: {},
    },
  L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3B5dGhvbi9kZWR1cGUvd29ya2VyLnB5:
    {
      content:
        'import asyncio\nfrom datetime import timedelta\nfrom typing import Any\n\nfrom hatchet_sdk import Context, EmptyModel, Hatchet, TriggerWorkflowOptions\nfrom hatchet_sdk.clients.admin import DedupeViolationErr\n\nhatchet = Hatchet(debug=True)\n\ndedupe_parent_wf = hatchet.workflow(name="DedupeParent")\ndedupe_child_wf = hatchet.workflow(name="DedupeChild")\n\n@dedupe_parent_wf.task(execution_timeout=timedelta(minutes=1))\nasync def spawn(input: EmptyModel, ctx: Context) -> dict[str, list[Any]]:\n    print("spawning child")\n\n    results = []\n\n    for i in range(2):\n        try:\n            results.append(\n                (\n                    dedupe_child_wf.aio_run(\n                        options=TriggerWorkflowOptions(\n                            additional_metadata={"dedupe": "test"}, key=f"child{i}"\n                        ),\n                    )\n                )\n            )\n        except DedupeViolationErr as e:\n            print(f"dedupe violation {e}")\n            continue\n\n    result = await asyncio.gather(*results)\n    print(f"results {result}")\n\n    return {"results": result}\n\n@dedupe_child_wf.task()\nasync def process(input: EmptyModel, ctx: Context) -> dict[str, str]:\n    await asyncio.sleep(3)\n\n    print("child process")\n    return {"status": "success"}\n\n@dedupe_child_wf.task()\nasync def process2(input: EmptyModel, ctx: Context) -> dict[str, str]:\n    print("child process2")\n    return {"status2": "success"}\n\ndef main() -> None:\n    worker = hatchet.worker(\n        "fanout-worker", slots=100, workflows=[dedupe_parent_wf, dedupe_child_wf]\n    )\n    worker.start()\n\nif __name__ == "__main__":\n    main()\n',
      language: 'py',
      source: 'examples/python/dedupe/worker.py',
      highlights: {},
    },
  L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3B5dGhvbi9kZWxheWVkL3Rlc3RfZGVsYXllZC5weQ__:
    {
      content:
        '# from hatchet_sdk import Hatchet\n# import pytest\n\n# from tests.utils import fixture_bg_worker\n\n# worker = fixture_bg_worker(["poetry", "run", "manual_trigger"])\n\n# # @pytest.mark.asyncio(loop_scope="session")\n# async def test_run(hatchet: Hatchet):\n#     # TODO\n',
      language: 'py',
      source: 'examples/python/delayed/test_delayed.py',
      highlights: {},
    },
  L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3B5dGhvbi9kZWxheWVkL3RyaWdnZXIucHk_:
    {
      content:
        'from examples.delayed.worker import PrinterInput, print_schedule_wf\n\nprint_schedule_wf.run(PrinterInput(message="test"))\n',
      language: 'py',
      source: 'examples/python/delayed/trigger.py',
      highlights: {},
    },
  L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3B5dGhvbi9kZWxheWVkL3dvcmtlci5weQ__:
    {
      content:
        'from datetime import datetime, timedelta\n\nfrom pydantic import BaseModel\n\nfrom hatchet_sdk import Context, Hatchet\n\nhatchet = Hatchet(debug=True)\n\nclass PrinterInput(BaseModel):\n    message: str\n\nprint_schedule_wf = hatchet.workflow(\n    name="PrintScheduleWorkflow",\n    input_validator=PrinterInput,\n)\nprint_printer_wf = hatchet.workflow(\n    name="PrintPrinterWorkflow", input_validator=PrinterInput\n)\n\n@print_schedule_wf.task()\ndef schedule(input: PrinterInput, ctx: Context) -> None:\n    now = datetime.now()\n    print(f"the time is \\t {now.strftime(\'%H:%M:%S\')}")\n    future_time = now + timedelta(seconds=15)\n    print(f"scheduling for \\t {future_time.strftime(\'%H:%M:%S\')}")\n\n    print_printer_wf.schedule(future_time, input=input)\n\n@print_schedule_wf.task()\ndef step1(input: PrinterInput, ctx: Context) -> None:\n    now = datetime.now()\n    print(f"printed at \\t {now.strftime(\'%H:%M:%S\')}")\n    print(f"message \\t {input.message}")\n\ndef main() -> None:\n    worker = hatchet.worker(\n        "delayed-worker", slots=4, workflows=[print_schedule_wf, print_printer_wf]\n    )\n\n    worker.start()\n\nif __name__ == "__main__":\n    main()\n',
      language: 'py',
      source: 'examples/python/delayed/worker.py',
      highlights: {},
    },
  L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3B5dGhvbi9kdXJhYmxlL3Rlc3RfZHVyYWJsZS5weQ__:
    {
      content:
        'import asyncio\nimport os\n\nimport pytest\n\nfrom examples.durable.worker import EVENT_KEY, SLEEP_TIME, durable_workflow\nfrom hatchet_sdk import Hatchet\n\n@pytest.mark.skipif(\n    os.getenv("CI", "false").lower() == "true",\n    reason="Skipped in CI because of unreliability",\n)\n@pytest.mark.asyncio(loop_scope="session")\nasync def test_durable(hatchet: Hatchet) -> None:\n    ref = durable_workflow.run_no_wait()\n\n    await asyncio.sleep(SLEEP_TIME + 10)\n\n    hatchet.event.push(EVENT_KEY, {})\n\n    result = await ref.aio_result()\n\n    workers = await hatchet.workers.aio_list()\n\n    assert workers.rows\n\n    active_workers = [w for w in workers.rows if w.status == "ACTIVE"]\n\n    assert len(active_workers) == 2\n    assert any(w.name == "e2e-test-worker" for w in active_workers)\n    assert any(w.name.endswith("e2e-test-worker_durable") for w in active_workers)\n    assert result["durable_task"]["status"] == "success"\n',
      language: 'py',
      source: 'examples/python/durable/test_durable.py',
      highlights: {},
    },
  L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3B5dGhvbi9kdXJhYmxlL3RyaWdnZXIucHk_:
    {
      content:
        'import time\n\nfrom examples.durable.worker import (\n    EVENT_KEY,\n    SLEEP_TIME,\n    durable_workflow,\n    ephemeral_workflow,\n    hatchet,\n)\n\ndurable_workflow.run_no_wait()\nephemeral_workflow.run_no_wait()\n\nprint("Sleeping")\ntime.sleep(SLEEP_TIME + 2)\n\nprint("Pushing event")\nhatchet.event.push(EVENT_KEY, {})\n',
      language: 'py',
      source: 'examples/python/durable/trigger.py',
      highlights: {},
    },
  L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3B5dGhvbi9kdXJhYmxlL3dvcmtlci5weQ__:
    {
      content:
        'from datetime import timedelta\n\nfrom hatchet_sdk import Context, DurableContext, EmptyModel, Hatchet, UserEventCondition\n\nhatchet = Hatchet(debug=True)\n\n# ❓ Create a durable workflow\ndurable_workflow = hatchet.workflow(name="DurableWorkflow")\n\nephemeral_workflow = hatchet.workflow(name="EphemeralWorkflow")\n\n# ❓ Add durable task\nEVENT_KEY = "durable-example:event"\nSLEEP_TIME = 5\n\n@durable_workflow.task()\nasync def ephemeral_task(input: EmptyModel, ctx: Context) -> None:\n    print("Running non-durable task")\n\n@durable_workflow.durable_task()\nasync def durable_task(input: EmptyModel, ctx: DurableContext) -> dict[str, str]:\n    print("Waiting for sleep")\n    await ctx.aio_sleep_for(duration=timedelta(seconds=SLEEP_TIME))\n    print("Sleep finished")\n\n    print("Waiting for event")\n    await ctx.aio_wait_for(\n        "event",\n        UserEventCondition(event_key=EVENT_KEY, expression="true"),\n    )\n    print("Event received")\n\n    return {\n        "status": "success",\n    }\n\n@ephemeral_workflow.task()\ndef ephemeral_task_2(input: EmptyModel, ctx: Context) -> None:\n    print("Running non-durable task")\n\ndef main() -> None:\n    worker = hatchet.worker(\n        "durable-worker", workflows=[durable_workflow, ephemeral_workflow]\n    )\n    worker.start()\n\nif __name__ == "__main__":\n    main()\n',
      language: 'py',
      source: 'examples/python/durable/worker.py',
      highlights: {},
    },
  L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3B5dGhvbi9kdXJhYmxlX2V2ZW50L3RyaWdnZXIucHk_:
    {
      content:
        'import time\n\nfrom examples.durable_event.worker import (\n    EVENT_KEY,\n    durable_event_task,\n    durable_event_task_with_filter,\n    hatchet,\n)\n\ndurable_event_task.run_no_wait()\ndurable_event_task_with_filter.run_no_wait()\n\nprint("Sleeping")\ntime.sleep(2)\n\nprint("Pushing event")\nhatchet.event.push(\n    EVENT_KEY,\n    {\n        "user_id": "1234",\n    },\n)\n',
      language: 'py',
      source: 'examples/python/durable_event/trigger.py',
      highlights: {},
    },
  L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3B5dGhvbi9kdXJhYmxlX2V2ZW50L3dvcmtlci5weQ__:
    {
      content:
        'from hatchet_sdk import DurableContext, EmptyModel, Hatchet, UserEventCondition\n\nhatchet = Hatchet(debug=True)\n\nEVENT_KEY = "user:update"\n\n# ❓ Durable Event\n@hatchet-dev/typescript-sdk.durable_task(name="DurableEventTask")\nasync def durable_event_task(input: EmptyModel, ctx: DurableContext) -> None:\n    res = await ctx.aio_wait_for(\n        "event",\n        UserEventCondition(event_key="user:update"),\n    )\n\n    print("got event", res)\n\n@hatchet-dev/typescript-sdk.durable_task(name="DurableEventWithFilterTask")\nasync def durable_event_task_with_filter(\n    input: EmptyModel, ctx: DurableContext\n) -> None:\n    # ❓ Durable Event With Filter\n    res = await ctx.aio_wait_for(\n        "event",\n        UserEventCondition(\n            event_key="user:update", expression="input.user_id == \'1234\'"\n        ),\n    )\n\n    print("got event", res)\n\ndef main() -> None:\n    worker = hatchet.worker(\n        "durable-event-worker",\n        workflows=[durable_event_task, durable_event_task_with_filter],\n    )\n    worker.start()\n\nif __name__ == "__main__":\n    main()\n',
      language: 'py',
      source: 'examples/python/durable_event/worker.py',
      highlights: {},
    },
  L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3B5dGhvbi9kdXJhYmxlX3NsZWVwL3RyaWdnZXIucHk_:
    {
      content:
        'from examples.durable_sleep.worker import durable_sleep_task\n\ndurable_sleep_task.run_no_wait()\n',
      language: 'py',
      source: 'examples/python/durable_sleep/trigger.py',
      highlights: {},
    },
  L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3B5dGhvbi9kdXJhYmxlX3NsZWVwL3dvcmtlci5weQ__:
    {
      content:
        'from datetime import timedelta\n\nfrom hatchet_sdk import DurableContext, EmptyModel, Hatchet\n\nhatchet = Hatchet(debug=True)\n\n# ❓ Durable Sleep\n@hatchet-dev/typescript-sdk.durable_task(name="DurableSleepTask")\nasync def durable_sleep_task(input: EmptyModel, ctx: DurableContext) -> None:\n    res = await ctx.aio_sleep_for(timedelta(seconds=5))\n\n    print("got result", res)\n\ndef main() -> None:\n    worker = hatchet.worker("durable-sleep-worker", workflows=[durable_sleep_task])\n    worker.start()\n\nif __name__ == "__main__":\n    main()\n',
      language: 'py',
      source: 'examples/python/durable_sleep/worker.py',
      highlights: {},
    },
  L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3B5dGhvbi9ldmVudHMvZXZlbnQucHk_:
    {
      content:
        'from hatchet_sdk import Hatchet\n\nhatchet = Hatchet()\n\n# ❓ Event trigger\nhatchet.event.push("user:create", {})\n# ‼️\n',
      language: 'py',
      source: 'examples/python/events/event.py',
      highlights: {},
    },
  L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3B5dGhvbi9ldmVudHMvdGVzdF9ldmVudC5weQ__:
    {
      content:
        'import pytest\n\nfrom hatchet_sdk.clients.events import BulkPushEventOptions, BulkPushEventWithMetadata\nfrom hatchet_sdk.hatchet import Hatchet\n\n@pytest.mark.asyncio(loop_scope="session")\nasync def test_event_push(hatchet: Hatchet) -> None:\n    e = hatchet.event.push("user:create", {"test": "test"})\n\n    assert e.eventId is not None\n\n@pytest.mark.asyncio(loop_scope="session")\nasync def test_async_event_push(hatchet: Hatchet) -> None:\n    e = await hatchet.event.aio_push("user:create", {"test": "test"})\n\n    assert e.eventId is not None\n\n@pytest.mark.asyncio(loop_scope="session")\nasync def test_async_event_bulk_push(hatchet: Hatchet) -> None:\n\n    events = [\n        BulkPushEventWithMetadata(\n            key="event1",\n            payload={"message": "This is event 1"},\n            additional_metadata={"source": "test", "user_id": "user123"},\n        ),\n        BulkPushEventWithMetadata(\n            key="event2",\n            payload={"message": "This is event 2"},\n            additional_metadata={"source": "test", "user_id": "user456"},\n        ),\n        BulkPushEventWithMetadata(\n            key="event3",\n            payload={"message": "This is event 3"},\n            additional_metadata={"source": "test", "user_id": "user789"},\n        ),\n    ]\n    opts = BulkPushEventOptions(namespace="bulk-test")\n\n    e = await hatchet.event.aio_bulk_push(events, opts)\n\n    assert len(e) == 3\n\n    # Sort both lists of events by their key to ensure comparison order\n    sorted_events = sorted(events, key=lambda x: x.key)\n    sorted_returned_events = sorted(e, key=lambda x: x.key)\n    namespace = "bulk-test"\n\n    # Check that the returned events match the original events\n    for original_event, returned_event in zip(sorted_events, sorted_returned_events):\n        assert returned_event.key == namespace + original_event.key\n',
      language: 'py',
      source: 'examples/python/events/test_event.py',
      highlights: {},
    },
  L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3B5dGhvbi9ldmVudHMvd29ya2VyLnB5:
    {
      content:
        'from hatchet_sdk import Context, EmptyModel, Hatchet\n\nhatchet = Hatchet()\n\n# ❓ Event trigger\nevent_workflow = hatchet.workflow(name="EventWorkflow", on_events=["user:create"])\n# ‼️\n\n@event_workflow.task()\ndef task(input: EmptyModel, ctx: Context) -> None:\n    print("event received")\n',
      language: 'py',
      source: 'examples/python/events/worker.py',
      highlights: {},
    },
  L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3B5dGhvbi9mYW5vdXQvc3RyZWFtLnB5:
    {
      content:
        'import asyncio\nimport random\n\nfrom examples.fanout.worker import ParentInput, parent_wf\nfrom hatchet_sdk import Hatchet\nfrom hatchet_sdk.clients.admin import TriggerWorkflowOptions\n\nasync def main() -> None:\n\n    hatchet = Hatchet()\n\n    # Generate a random stream key to use to track all\n    # stream events for this workflow run.\n\n    streamKey = "streamKey"\n    streamVal = f"sk-{random.randint(1, 100)}"\n\n    # Specify the stream key as additional metadata\n    # when running the workflow.\n\n    # This key gets propagated to all child workflows\n    # and can have an arbitrary property name.\n\n    parent_wf.run(\n        ParentInput(n=2),\n        options=TriggerWorkflowOptions(additional_metadata={streamKey: streamVal}),\n    )\n\n    # Stream all events for the additional meta key value\n    listener = hatchet.listener.stream_by_additional_metadata(streamKey, streamVal)\n\n    async for event in listener:\n        print(event.type, event.payload)\n\nif __name__ == "__main__":\n    asyncio.run(main())\n',
      language: 'py',
      source: 'examples/python/fanout/stream.py',
      highlights: {},
    },
  L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3B5dGhvbi9mYW5vdXQvc3luY19zdHJlYW0ucHk_:
    {
      content:
        'import random\n\nfrom examples.fanout.worker import ParentInput, parent_wf\nfrom hatchet_sdk import Hatchet\nfrom hatchet_sdk.clients.admin import TriggerWorkflowOptions\n\ndef main() -> None:\n\n    hatchet = Hatchet()\n\n    # Generate a random stream key to use to track all\n    # stream events for this workflow run.\n\n    streamKey = "streamKey"\n    streamVal = f"sk-{random.randint(1, 100)}"\n\n    # Specify the stream key as additional metadata\n    # when running the workflow.\n\n    # This key gets propagated to all child workflows\n    # and can have an arbitrary property name.\n\n    parent_wf.run(\n        ParentInput(n=2),\n        options=TriggerWorkflowOptions(additional_metadata={streamKey: streamVal}),\n    )\n\n    # Stream all events for the additional meta key value\n    listener = hatchet.listener.stream_by_additional_metadata(streamKey, streamVal)\n\n    for event in listener:\n        print(event.type, event.payload)\n\n    print("DONE.")\n\nif __name__ == "__main__":\n    main()\n',
      language: 'py',
      source: 'examples/python/fanout/sync_stream.py',
      highlights: {},
    },
  L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3B5dGhvbi9mYW5vdXQvdGVzdF9mYW5vdXQucHk_:
    {
      content:
        'import pytest\n\nfrom examples.fanout.worker import ParentInput, parent_wf\n\n@pytest.mark.asyncio(loop_scope="session")\nasync def test_run() -> None:\n    result = await parent_wf.aio_run(ParentInput(n=2))\n\n    assert len(result["spawn"]["results"]) == 2\n',
      language: 'py',
      source: 'examples/python/fanout/test_fanout.py',
      highlights: {},
    },
  L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3B5dGhvbi9mYW5vdXQvdHJpZ2dlci5weQ__:
    {
      content:
        'import asyncio\n\nfrom examples.fanout.worker import ParentInput, parent_wf\nfrom hatchet_sdk import Hatchet\nfrom hatchet_sdk.clients.admin import TriggerWorkflowOptions\n\nhatchet = Hatchet()\n\nasync def main() -> None:\n    await parent_wf.aio_run(\n        ParentInput(n=2),\n        options=TriggerWorkflowOptions(additional_metadata={"hello": "moon"}),\n    )\n\nif __name__ == "__main__":\n    asyncio.run(main())\n',
      language: 'py',
      source: 'examples/python/fanout/trigger.py',
      highlights: {},
    },
  L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3B5dGhvbi9mYW5vdXQvd29ya2VyLnB5:
    {
      content:
        'from datetime import timedelta\nfrom typing import Any\n\nfrom pydantic import BaseModel\n\nfrom hatchet_sdk import Context, Hatchet, TriggerWorkflowOptions\n\nhatchet = Hatchet(debug=True)\n\n# ❓ FanoutParent\nclass ParentInput(BaseModel):\n    n: int = 100\n\nclass ChildInput(BaseModel):\n    a: str\n\nparent_wf = hatchet.workflow(name="FanoutParent", input_validator=ParentInput)\nchild_wf = hatchet.workflow(name="FanoutChild", input_validator=ChildInput)\n\n@parent_wf.task(execution_timeout=timedelta(minutes=5))\nasync def spawn(input: ParentInput, ctx: Context) -> dict[str, Any]:\n    print("spawning child")\n\n    result = await child_wf.aio_run_many(\n        [\n            child_wf.create_bulk_run_item(\n                input=ChildInput(a=str(i)),\n                options=TriggerWorkflowOptions(\n                    additional_metadata={"hello": "earth"}, key=f"child{i}"\n                ),\n            )\n            for i in range(input.n)\n        ]\n    )\n\n    print(f"results {result}")\n\n    return {"results": result}\n\n# ‼️\n\n# ❓ FanoutChild\n@child_wf.task()\ndef process(input: ChildInput, ctx: Context) -> dict[str, str]:\n    print(f"child process {input.a}")\n    return {"status": input.a}\n\n@child_wf.task(parents=[process])\ndef process2(input: ChildInput, ctx: Context) -> dict[str, str]:\n    process_output = ctx.task_output(process)\n    a = process_output["status"]\n\n    return {"status2": a + "2"}\n\n# ‼️\n\nchild_wf.create_bulk_run_item()\n\ndef main() -> None:\n    worker = hatchet.worker("fanout-worker", slots=40, workflows=[parent_wf, child_wf])\n    worker.start()\n\nif __name__ == "__main__":\n    main()\n',
      language: 'py',
      source: 'examples/python/fanout/worker.py',
      highlights: {},
    },
  L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3B5dGhvbi9mYW5vdXRfc3luYy90ZXN0X2Zhbm91dF9zeW5jLnB5:
    {
      content:
        'from examples.fanout_sync.worker import ParentInput, sync_fanout_parent\n\ndef test_run() -> None:\n    N = 2\n\n    result = sync_fanout_parent.run(ParentInput(n=N))\n\n    assert len(result["spawn"]["results"]) == N\n',
      language: 'py',
      source: 'examples/python/fanout_sync/test_fanout_sync.py',
      highlights: {},
    },
  L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3B5dGhvbi9mYW5vdXRfc3luYy90cmlnZ2VyLnB5:
    {
      content:
        'import asyncio\n\nfrom examples.fanout_sync.worker import ParentInput, sync_fanout_parent\nfrom hatchet_sdk import Hatchet, TriggerWorkflowOptions\n\nhatchet = Hatchet()\n\nasync def main() -> None:\n    sync_fanout_parent.run(\n        ParentInput(n=2),\n        options=TriggerWorkflowOptions(additional_metadata={"hello": "moon"}),\n    )\n\nif __name__ == "__main__":\n    asyncio.run(main())\n',
      language: 'py',
      source: 'examples/python/fanout_sync/trigger.py',
      highlights: {},
    },
  L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3B5dGhvbi9mYW5vdXRfc3luYy93b3JrZXIucHk_:
    {
      content:
        'from datetime import timedelta\nfrom typing import Any\n\nfrom pydantic import BaseModel\n\nfrom hatchet_sdk import Context, Hatchet, TriggerWorkflowOptions\n\nhatchet = Hatchet(debug=True)\n\nclass ParentInput(BaseModel):\n    n: int = 5\n\nclass ChildInput(BaseModel):\n    a: str\n\nsync_fanout_parent = hatchet.workflow(\n    name="SyncFanoutParent", input_validator=ParentInput\n)\nsync_fanout_child = hatchet.workflow(name="SyncFanoutChild", input_validator=ChildInput)\n\n@sync_fanout_parent.task(execution_timeout=timedelta(minutes=5))\ndef spawn(input: ParentInput, ctx: Context) -> dict[str, list[dict[str, Any]]]:\n    print("spawning child")\n\n    results = sync_fanout_child.run_many(\n        [\n            sync_fanout_child.create_bulk_run_item(\n                input=ChildInput(a=str(i)),\n                key=f"child{i}",\n                options=TriggerWorkflowOptions(additional_metadata={"hello": "earth"}),\n            )\n            for i in range(input.n)\n        ],\n    )\n\n    print(f"results {results}")\n\n    return {"results": results}\n\n@sync_fanout_child.task()\ndef process(input: ChildInput, ctx: Context) -> dict[str, str]:\n    return {"status": "success " + input.a}\n\ndef main() -> None:\n    worker = hatchet.worker(\n        "sync-fanout-worker",\n        slots=40,\n        workflows=[sync_fanout_parent, sync_fanout_child],\n    )\n    worker.start()\n\nif __name__ == "__main__":\n    main()\n',
      language: 'py',
      source: 'examples/python/fanout_sync/worker.py',
      highlights: {},
    },
  L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3B5dGhvbi9saWZlc3BhbnMvc2ltcGxlLnB5:
    {
      content:
        '# ❓ Lifespan\n\nfrom typing import AsyncGenerator, cast\n\nfrom pydantic import BaseModel\n\nfrom hatchet_sdk import Context, EmptyModel, Hatchet\n\nhatchet = Hatchet(debug=True)\n\nclass Lifespan(BaseModel):\n    foo: str\n    pi: float\n\nasync def lifespan() -> AsyncGenerator[Lifespan, None]:\n    yield Lifespan(foo="bar", pi=3.14)\n\n@hatchet-dev/typescript-sdk.task(name="LifespanWorkflow")\ndef lifespan_task(input: EmptyModel, ctx: Context) -> Lifespan:\n    return cast(Lifespan, ctx.lifespan)\n\ndef main() -> None:\n    worker = hatchet.worker(\n        "test-worker", slots=1, workflows=[lifespan_task], lifespan=lifespan\n    )\n    worker.start()\n\n# ‼️\n\nif __name__ == "__main__":\n    main()\n',
      language: 'py',
      source: 'examples/python/lifespans/simple.py',
      highlights: {},
    },
  L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3B5dGhvbi9saWZlc3BhbnMvdGVzdF9saWZlc3BhbnMucHk_:
    {
      content:
        'import pytest\n\nfrom examples.lifespans.simple import Lifespan, lifespan_task\n\n@pytest.mark.asyncio(loop_scope="session")\nasync def test_lifespans() -> None:\n    result = await lifespan_task.aio_run()\n\n    assert isinstance(result, Lifespan)\n    assert result.pi == 3.14\n    assert result.foo == "bar"\n',
      language: 'py',
      source: 'examples/python/lifespans/test_lifespans.py',
      highlights: {},
    },
  L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3B5dGhvbi9saWZlc3BhbnMvdHJpZ2dlci5weQ__:
    {
      content:
        'from examples.lifespans.worker import lifespan_workflow\n\nresult = lifespan_workflow.run()\n\nprint(result)\n',
      language: 'py',
      source: 'examples/python/lifespans/trigger.py',
      highlights: {},
    },
  L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3B5dGhvbi9saWZlc3BhbnMvd29ya2VyLnB5:
    {
      content:
        'from typing import AsyncGenerator, cast\nfrom uuid import UUID\n\nfrom psycopg_pool import ConnectionPool\nfrom pydantic import BaseModel, ConfigDict\n\nfrom hatchet_sdk import Context, EmptyModel, Hatchet\n\nhatchet = Hatchet(debug=True)\n\n# ❓ Use the lifespan in a task\nclass TaskOutput(BaseModel):\n    num_rows: int\n    external_ids: list[UUID]\n\nlifespan_workflow = hatchet.workflow(name="LifespanWorkflow")\n\n@lifespan_workflow.task()\ndef sync_lifespan_task(input: EmptyModel, ctx: Context) -> TaskOutput:\n    pool = cast(Lifespan, ctx.lifespan).pool\n\n    with pool.connection() as conn:\n        query = conn.execute("SELECT * FROM v1_lookup_table_olap LIMIT 5;")\n        rows = query.fetchall()\n\n        for row in rows:\n            print(row)\n\n        print("executed sync task with lifespan", ctx.lifespan)\n\n        return TaskOutput(\n            num_rows=len(rows),\n            external_ids=[cast(UUID, row[0]) for row in rows],\n        )\n\n@lifespan_workflow.task()\nasync def async_lifespan_task(input: EmptyModel, ctx: Context) -> TaskOutput:\n    pool = cast(Lifespan, ctx.lifespan).pool\n\n    with pool.connection() as conn:\n        query = conn.execute("SELECT * FROM v1_lookup_table_olap LIMIT 5;")\n        rows = query.fetchall()\n\n        for row in rows:\n            print(row)\n\n        print("executed async task with lifespan", ctx.lifespan)\n\n        return TaskOutput(\n            num_rows=len(rows),\n            external_ids=[cast(UUID, row[0]) for row in rows],\n        )\n\n# ❓ Define a lifespan\nclass Lifespan(BaseModel):\n    model_config = ConfigDict(arbitrary_types_allowed=True)\n\n    foo: str\n    pool: ConnectionPool\n\nasync def lifespan() -> AsyncGenerator[Lifespan, None]:\n    print("Running lifespan!")\n    with ConnectionPool("postgres://hatchet:hatchet@localhost:5431/hatchet") as pool:\n        yield Lifespan(\n            foo="bar",\n            pool=pool,\n        )\n\n    print("Cleaning up lifespan!")\n\nworker = hatchet.worker(\n    "test-worker", slots=1, workflows=[lifespan_workflow], lifespan=lifespan\n)\n\ndef main() -> None:\n    worker.start()\n\nif __name__ == "__main__":\n    main()\n',
      language: 'py',
      source: 'examples/python/lifespans/worker.py',
      highlights: {},
    },
  L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3B5dGhvbi9sb2dnZXIvY2xpZW50LnB5:
    {
      content:
        '# ❓ RootLogger\n\nimport logging\n\nfrom hatchet_sdk import ClientConfig, Hatchet\n\nlogging.basicConfig(level=logging.INFO)\n\nroot_logger = logging.getLogger()\n\nhatchet = Hatchet(\n    debug=True,\n    config=ClientConfig(\n        logger=root_logger,\n    ),\n)\n\n# ‼️\n',
      language: 'py',
      source: 'examples/python/logger/client.py',
      highlights: {},
    },
  L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3B5dGhvbi9sb2dnZXIvdGVzdF9sb2dnZXIucHk_:
    {
      content:
        'import pytest\n\nfrom examples.logger.workflow import logging_workflow\n\n@pytest.mark.asyncio(loop_scope="session")\nasync def test_run() -> None:\n    result = await logging_workflow.aio_run()\n\n    assert result["root_logger"]["status"] == "success"\n',
      language: 'py',
      source: 'examples/python/logger/test_logger.py',
      highlights: {},
    },
  L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3B5dGhvbi9sb2dnZXIvdHJpZ2dlci5weQ__:
    {
      content:
        'from examples.logger.workflow import logging_workflow\n\nlogging_workflow.run()\n',
      language: 'py',
      source: 'examples/python/logger/trigger.py',
      highlights: {},
    },
  L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3B5dGhvbi9sb2dnZXIvd29ya2VyLnB5:
    {
      content:
        'from examples.logger.client import hatchet\nfrom examples.logger.workflow import logging_workflow\n\ndef main() -> None:\n    worker = hatchet.worker("logger-worker", slots=5, workflows=[logging_workflow])\n\n    worker.start()\n\nif __name__ == "__main__":\n    main()\n',
      language: 'py',
      source: 'examples/python/logger/worker.py',
      highlights: {},
    },
  L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3B5dGhvbi9sb2dnZXIvd29ya2Zsb3cucHk_:
    {
      content:
        '# ❓ LoggingWorkflow\n\nimport logging\nimport time\n\nfrom examples.logger.client import hatchet\nfrom hatchet_sdk import Context, EmptyModel\n\nlogger = logging.getLogger(__name__)\n\nlogging_workflow = hatchet.workflow(\n    name="LoggingWorkflow",\n)\n\n@logging_workflow.task()\ndef root_logger(input: EmptyModel, ctx: Context) -> dict[str, str]:\n    for i in range(12):\n        logger.info("executed step1 - {}".format(i))\n        logger.info({"step1": "step1"})\n\n        time.sleep(0.1)\n\n    return {"status": "success"}\n\n# ‼️\n\n# ❓ ContextLogger\n\n@logging_workflow.task()\ndef context_logger(input: EmptyModel, ctx: Context) -> dict[str, str]:\n    for i in range(12):\n        ctx.log("executed step1 - {}".format(i))\n        ctx.log({"step1": "step1"})\n\n        time.sleep(0.1)\n\n    return {"status": "success"}\n\n# ‼️\n',
      language: 'py',
      source: 'examples/python/logger/workflow.py',
      highlights: {},
    },
  L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3B5dGhvbi9tYW51YWxfc2xvdF9yZWxlYXNlL3dvcmtlci5weQ__:
    {
      content:
        'import time\n\nfrom hatchet_sdk import Context, EmptyModel, Hatchet\n\nhatchet = Hatchet()\n\n# ❓ SlotRelease\n\nslot_release_workflow = hatchet.workflow(name="SlotReleaseWorkflow")\n\n@slot_release_workflow.task()\ndef step1(input: EmptyModel, ctx: Context) -> dict[str, str]:\n    print("RESOURCE INTENSIVE PROCESS")\n    time.sleep(10)\n\n    # 👀 Release the slot after the resource-intensive process, so that other steps can run\n    ctx.release_slot()\n\n    print("NON RESOURCE INTENSIVE PROCESS")\n    return {"status": "success"}\n\n# ‼️\n',
      language: 'py',
      source: 'examples/python/manual_slot_release/worker.py',
      highlights: {},
    },
  L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3B5dGhvbi9taWdyYXRpb25fZ3VpZGVzL19faW5pdF9fLnB5:
    {
      content: '',
      language: 'py',
      source: 'examples/python/migration_guides/__init__.py',
      highlights: {},
    },
  L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3B5dGhvbi9taWdyYXRpb25fZ3VpZGVzL2hhdGNoZXRfY2xpZW50LnB5:
    {
      content: 'from hatchet_sdk import Hatchet\n\nhatchet = Hatchet()\n',
      language: 'py',
      source: 'examples/python/migration_guides/hatchet_client.py',
      highlights: {},
    },
  L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3B5dGhvbi9taWdyYXRpb25fZ3VpZGVzL21lcmdlbnQucHk_:
    {
      content:
        'from datetime import datetime, timedelta\nfrom typing import Any, Dict, List, Mapping\n\nimport requests\nfrom pydantic import BaseModel\nfrom requests import Response\n\nfrom hatchet_sdk.context.context import Context\n\nfrom .hatchet_client import hatchet\n\nasync def process_image(image_url: str, filters: List[str]) -> Dict[str, Any]:\n    # Do some image processing\n    return {"url": image_url, "size": 100, "format": "png"}\n\n# ❓ Before (Mergent)\nasync def process_image_task(request: Any) -> Dict[str, Any]:\n    image_url = request.json["image_url"]\n    filters = request.json["filters"]\n    try:\n        result = await process_image(image_url, filters)\n        return {"success": True, "processed_url": result["url"]}\n    except Exception as e:\n        print(f"Image processing failed: {e}")\n        raise\n\n# ❓ After (Hatchet)\nclass ImageProcessInput(BaseModel):\n    image_url: str\n    filters: List[str]\n\nclass ImageProcessOutput(BaseModel):\n    processed_url: str\n    metadata: Dict[str, Any]\n\n@hatchet-dev/typescript-sdk.task(\n    name="image-processor",\n    retries=3,\n    execution_timeout="10m",\n    input_validator=ImageProcessInput,\n)\nasync def image_processor(input: ImageProcessInput, ctx: Context) -> ImageProcessOutput:\n    # Do some image processing\n    result = await process_image(input.image_url, input.filters)\n\n    if not result["url"]:\n        raise ValueError("Processing failed to generate URL")\n\n    return ImageProcessOutput(\n        processed_url=result["url"],\n        metadata={\n            "size": result["size"],\n            "format": result["format"],\n            "applied_filters": input.filters,\n        },\n    )\n\nasync def run() -> None:\n    # ❓ Running a task (Mergent)\n    headers: Mapping[str, str] = {\n        "Authorization": "Bearer <token>",\n        "Content-Type": "application/json",\n    }\n\n    task_data = {\n        "name": "4cf95241-fa19-47ef-8a67-71e483747649",\n        "queue": "default",\n        "request": {\n            "url": "https://example.com",\n            "headers": {\n                "Authorization": "fake-secret-token",\n                "Content-Type": "application/json",\n            },\n            "body": "Hello, world!",\n        },\n    }\n\n    try:\n        response: Response = requests.post(\n            "https://api.mergent.co/v2/tasks",\n            headers=headers,\n            json=task_data,\n        )\n        print(response.json())\n    except Exception as e:\n        print(f"Error: {e}")\n\n    # ❓ Running a task (Hatchet)\n    result = await image_processor.aio_run(\n        ImageProcessInput(image_url="https://example.com/image.png", filters=["blur"])\n    )\n\n    # you can await fully typed results\n    print(result)\n\nasync def schedule() -> None:\n    # ❓ Scheduling tasks (Mergent)\n    options = {\n        # same options as before\n        "json": {\n            # same body as before\n            "delay": "5m"\n        }\n    }\n\n    print(options)\n\n    # ❓ Scheduling tasks (Hatchet)\n    # Schedule the task to run at a specific time\n    run_at = datetime.now() + timedelta(days=1)\n    await image_processor.aio_schedule(\n        run_at,\n        ImageProcessInput(image_url="https://example.com/image.png", filters=["blur"]),\n    )\n\n    # Schedule the task to run every hour\n    await image_processor.aio_create_cron(\n        "run-hourly",\n        "0 * * * *",\n        ImageProcessInput(image_url="https://example.com/image.png", filters=["blur"]),\n    )\n',
      language: 'py',
      source: 'examples/python/migration_guides/mergent.py',
      highlights: {},
    },
  L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3B5dGhvbi9ub25fcmV0cnlhYmxlL3Rlc3Rfbm9fcmV0cnkucHk_:
    {
      content:
        'import pytest\n\nfrom examples.non_retryable.worker import (\n    non_retryable_workflow,\n    should_not_retry,\n    should_not_retry_successful_task,\n    should_retry_wrong_exception_type,\n)\nfrom hatchet_sdk import Hatchet\nfrom hatchet_sdk.clients.rest.models.v1_task_event_type import V1TaskEventType\nfrom hatchet_sdk.clients.rest.models.v1_workflow_run_details import V1WorkflowRunDetails\n\ndef find_id(runs: V1WorkflowRunDetails, match: str) -> str:\n    return next(t.metadata.id for t in runs.tasks if match in t.display_name)\n\n@pytest.mark.asyncio(loop_scope="session")\nasync def test_no_retry(hatchet: Hatchet) -> None:\n    ref = await non_retryable_workflow.aio_run_no_wait()\n\n    with pytest.raises(Exception, match="retry"):\n        await ref.aio_result()\n\n    runs = await hatchet.runs.aio_get(ref.workflow_run_id)\n    task_to_id = {\n        task: find_id(runs, task.name)\n        for task in [\n            should_not_retry_successful_task,\n            should_retry_wrong_exception_type,\n            should_not_retry,\n        ]\n    }\n\n    retrying_events = [\n        e for e in runs.task_events if e.event_type == V1TaskEventType.RETRYING\n    ]\n\n    """Only one task should be retried."""\n    assert len(retrying_events) == 1\n\n    """The task id of the retrying events should match the tasks that are retried"""\n    assert {e.task_id for e in retrying_events} == {\n        task_to_id[should_retry_wrong_exception_type],\n    }\n\n    """Three failed events should emit, one each for the two failing initial runs and one for the retry."""\n    assert (\n        len([e for e in runs.task_events if e.event_type == V1TaskEventType.FAILED])\n        == 3\n    )\n',
      language: 'py',
      source: 'examples/python/non_retryable/test_no_retry.py',
      highlights: {},
    },
  L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3B5dGhvbi9ub25fcmV0cnlhYmxlL3RyaWdnZXIucHk_:
    {
      content:
        'from examples.non_retryable.worker import non_retryable_workflow\n\nnon_retryable_workflow.run_no_wait()\n',
      language: 'py',
      source: 'examples/python/non_retryable/trigger.py',
      highlights: {},
    },
  L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3B5dGhvbi9ub25fcmV0cnlhYmxlL3dvcmtlci5weQ__:
    {
      content:
        'from hatchet_sdk import Context, EmptyModel, Hatchet\nfrom hatchet_sdk.exceptions import NonRetryableException\n\nhatchet = Hatchet(debug=True)\n\nnon_retryable_workflow = hatchet.workflow(name="NonRetryableWorkflow")\n\n# ❓ Non-retryable task\n@non_retryable_workflow.task(retries=1)\ndef should_not_retry(input: EmptyModel, ctx: Context) -> None:\n    raise NonRetryableException("This task should not retry")\n\n@non_retryable_workflow.task(retries=1)\ndef should_retry_wrong_exception_type(input: EmptyModel, ctx: Context) -> None:\n    raise TypeError("This task should retry because it\'s not a NonRetryableException")\n\n@non_retryable_workflow.task(retries=1)\ndef should_not_retry_successful_task(input: EmptyModel, ctx: Context) -> None:\n    pass\n\ndef main() -> None:\n    worker = hatchet.worker("non-retry-worker", workflows=[non_retryable_workflow])\n\n    worker.start()\n\nif __name__ == "__main__":\n    main()\n',
      language: 'py',
      source: 'examples/python/non_retryable/worker.py',
      highlights: {},
    },
  L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3B5dGhvbi9vbl9mYWlsdXJlL3Rlc3Rfb25fZmFpbHVyZS5weQ__:
    {
      content:
        'import asyncio\n\nimport pytest\n\nfrom examples.on_failure.worker import on_failure_wf\nfrom hatchet_sdk import Hatchet\nfrom hatchet_sdk.clients.rest.models.v1_task_status import V1TaskStatus\n\n@pytest.mark.asyncio(loop_scope="session")\nasync def test_run_timeout(hatchet: Hatchet) -> None:\n    run = on_failure_wf.run_no_wait()\n    try:\n        await run.aio_result()\n\n        assert False, "Expected workflow to timeout"\n    except Exception as e:\n        assert "step1 failed" in str(e)\n\n    await asyncio.sleep(5)  # Wait for the on_failure job to finish\n\n    details = await hatchet.runs.aio_get(run.workflow_run_id)\n\n    assert len(details.tasks) == 2\n    assert sum(t.status == V1TaskStatus.COMPLETED for t in details.tasks) == 1\n    assert sum(t.status == V1TaskStatus.FAILED for t in details.tasks) == 1\n\n    completed_task = next(\n        t for t in details.tasks if t.status == V1TaskStatus.COMPLETED\n    )\n    failed_task = next(t for t in details.tasks if t.status == V1TaskStatus.FAILED)\n\n    assert "on_failure" in completed_task.display_name\n    assert "step1" in failed_task.display_name\n',
      language: 'py',
      source: 'examples/python/on_failure/test_on_failure.py',
      highlights: {},
    },
  L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3B5dGhvbi9vbl9mYWlsdXJlL3RyaWdnZXIucHk_:
    {
      content:
        'from examples.on_failure.worker import on_failure_wf_with_details\n\non_failure_wf_with_details.run_no_wait()\n',
      language: 'py',
      source: 'examples/python/on_failure/trigger.py',
      highlights: {},
    },
  L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3B5dGhvbi9vbl9mYWlsdXJlL3dvcmtlci5weQ__:
    {
      content:
        'import json\nfrom datetime import timedelta\n\nfrom hatchet_sdk import Context, EmptyModel, Hatchet\n\nhatchet = Hatchet(debug=True)\n\nERROR_TEXT = "step1 failed"\n\n# ❓ OnFailure Step\n# This workflow will fail because the step will throw an error\n# we define an onFailure step to handle this case\n\non_failure_wf = hatchet.workflow(name="OnFailureWorkflow")\n\n@on_failure_wf.task(execution_timeout=timedelta(seconds=1))\ndef step1(input: EmptyModel, ctx: Context) -> None:\n    # 👀 this step will always raise an exception\n    raise Exception(ERROR_TEXT)\n\n# 👀 After the workflow fails, this special step will run\n@on_failure_wf.on_failure_task()\ndef on_failure(input: EmptyModel, ctx: Context) -> dict[str, str]:\n    # 👀 we can do things like perform cleanup logic\n    # or notify a user here\n\n    # 👀 Fetch the errors from upstream step runs from the context\n    print(ctx.task_run_errors)\n\n    return {"status": "success"}\n\n# ‼️\n\n# ❓ OnFailure With Details\n# We can access the failure details in the onFailure step\n# via the context method\n\non_failure_wf_with_details = hatchet.workflow(name="OnFailureWorkflowWithDetails")\n\n# ... defined as above\n@on_failure_wf_with_details.task(execution_timeout=timedelta(seconds=1))\ndef details_step1(input: EmptyModel, ctx: Context) -> None:\n    raise Exception(ERROR_TEXT)\n\n# 👀 After the workflow fails, this special step will run\n@on_failure_wf_with_details.on_failure_task()\ndef details_on_failure(input: EmptyModel, ctx: Context) -> dict[str, str]:\n    error = ctx.fetch_task_run_error(details_step1)\n\n    # 👀 we can access the failure details here\n    print(json.dumps(error, indent=2))\n\n    if error and error.startswith(ERROR_TEXT):\n        return {"status": "success"}\n\n    raise Exception("unexpected failure")\n\n# ‼️\n\ndef main() -> None:\n    worker = hatchet.worker(\n        "on-failure-worker",\n        slots=4,\n        workflows=[on_failure_wf, on_failure_wf_with_details],\n    )\n    worker.start()\n\nif __name__ == "__main__":\n    main()\n',
      language: 'py',
      source: 'examples/python/on_failure/worker.py',
      highlights: {},
    },
  L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3B5dGhvbi9vbl9zdWNjZXNzL3RyaWdnZXIucHk_:
    {
      content:
        'from examples.on_success.worker import on_success_workflow\n\non_success_workflow.run_no_wait()\n',
      language: 'py',
      source: 'examples/python/on_success/trigger.py',
      highlights: {},
    },
  L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3B5dGhvbi9vbl9zdWNjZXNzL3dvcmtlci5weQ__:
    {
      content:
        'from hatchet_sdk import Context, EmptyModel, Hatchet\n\nhatchet = Hatchet(debug=True)\n\non_success_workflow = hatchet.workflow(name="OnSuccessWorkflow")\n\n@on_success_workflow.task()\ndef first_task(input: EmptyModel, ctx: Context) -> None:\n    print("First task completed successfully")\n\n@on_success_workflow.task(parents=[first_task])\ndef second_task(input: EmptyModel, ctx: Context) -> None:\n    print("Second task completed successfully")\n\n@on_success_workflow.task(parents=[first_task, second_task])\ndef third_task(input: EmptyModel, ctx: Context) -> None:\n    print("Third task completed successfully")\n\n@on_success_workflow.task()\ndef fourth_task(input: EmptyModel, ctx: Context) -> None:\n    print("Fourth task completed successfully")\n\n@on_success_workflow.on_success_task()\ndef on_success_task(input: EmptyModel, ctx: Context) -> None:\n    print("On success task completed successfully")\n\ndef main() -> None:\n    worker = hatchet.worker("on-success-worker", workflows=[on_success_workflow])\n    worker.start()\n\nif __name__ == "__main__":\n    main()\n',
      language: 'py',
      source: 'examples/python/on_success/worker.py',
      highlights: {},
    },
  L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3B5dGhvbi9vcGVudGVsZW1ldHJ5X2luc3RydW1lbnRhdGlvbi9jbGllbnQucHk_:
    {
      content:
        'from hatchet_sdk import Hatchet\n\nhatchet = Hatchet(debug=True)\n',
      language: 'py',
      source: 'examples/python/opentelemetry_instrumentation/client.py',
      highlights: {},
    },
  L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3B5dGhvbi9vcGVudGVsZW1ldHJ5X2luc3RydW1lbnRhdGlvbi90cmFjZXIucHk_:
    {
      content:
        'import os\nfrom typing import cast\n\nfrom opentelemetry.exporter.otlp.proto.http.trace_exporter import OTLPSpanExporter\nfrom opentelemetry.sdk.resources import SERVICE_NAME, Resource\nfrom opentelemetry.sdk.trace import TracerProvider\nfrom opentelemetry.sdk.trace.export import BatchSpanProcessor\nfrom opentelemetry.trace import NoOpTracerProvider\n\ntrace_provider: TracerProvider | NoOpTracerProvider\n\nif os.getenv("CI", "false") == "true":\n    trace_provider = NoOpTracerProvider()\nelse:\n    resource = Resource(\n        attributes={\n            SERVICE_NAME: os.getenv("HATCHET_CLIENT_OTEL_SERVICE_NAME", "test-service")\n        }\n    )\n\n    headers = dict(\n        [\n            cast(\n                tuple[str, str],\n                tuple(\n                    os.getenv(\n                        "HATCHET_CLIENT_OTEL_EXPORTER_OTLP_HEADERS", "foo=bar"\n                    ).split("=")\n                ),\n            )\n        ]\n    )\n\n    processor = BatchSpanProcessor(\n        OTLPSpanExporter(\n            endpoint=os.getenv(\n                "HATCHET_CLIENT_OTEL_EXPORTER_OTLP_ENDPOINT", "http://localhost:4317"\n            ),\n            headers=headers,\n        ),\n    )\n\n    trace_provider = TracerProvider(resource=resource)\n\n    trace_provider.add_span_processor(processor)\n',
      language: 'py',
      source: 'examples/python/opentelemetry_instrumentation/tracer.py',
      highlights: {},
    },
  L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3B5dGhvbi9vcGVudGVsZW1ldHJ5X2luc3RydW1lbnRhdGlvbi90cmlnZ2Vycy5weQ__:
    {
      content:
        'import asyncio\n\nfrom examples.opentelemetry_instrumentation.client import hatchet\nfrom examples.opentelemetry_instrumentation.tracer import trace_provider\nfrom examples.opentelemetry_instrumentation.worker import otel_workflow\nfrom hatchet_sdk.clients.admin import TriggerWorkflowOptions\nfrom hatchet_sdk.clients.events import BulkPushEventWithMetadata, PushEventOptions\nfrom hatchet_sdk.opentelemetry.instrumentor import (\n    HatchetInstrumentor,\n    inject_traceparent_into_metadata,\n)\n\ninstrumentor = HatchetInstrumentor(tracer_provider=trace_provider)\ntracer = trace_provider.get_tracer(__name__)\n\ndef create_additional_metadata() -> dict[str, str]:\n    return inject_traceparent_into_metadata({"hello": "world"})\n\ndef create_push_options() -> PushEventOptions:\n    return PushEventOptions(additional_metadata=create_additional_metadata())\n\ndef push_event() -> None:\n    print("\\npush_event")\n    with tracer.start_as_current_span("push_event"):\n        hatchet.event.push(\n            "otel:event",\n            {"test": "test"},\n            options=create_push_options(),\n        )\n\nasync def async_push_event() -> None:\n    print("\\nasync_push_event")\n    with tracer.start_as_current_span("async_push_event"):\n        await hatchet.event.aio_push(\n            "otel:event", {"test": "test"}, options=create_push_options()\n        )\n\ndef bulk_push_event() -> None:\n    print("\\nbulk_push_event")\n    with tracer.start_as_current_span("bulk_push_event"):\n        hatchet.event.bulk_push(\n            [\n                BulkPushEventWithMetadata(\n                    key="otel:event",\n                    payload={"test": "test 1"},\n                    additional_metadata=create_additional_metadata(),\n                ),\n                BulkPushEventWithMetadata(\n                    key="otel:event",\n                    payload={"test": "test 2"},\n                    additional_metadata=create_additional_metadata(),\n                ),\n            ],\n        )\n\nasync def async_bulk_push_event() -> None:\n    print("\\nasync_bulk_push_event")\n    with tracer.start_as_current_span("bulk_push_event"):\n        await hatchet.event.aio_bulk_push(\n            [\n                BulkPushEventWithMetadata(\n                    key="otel:event",\n                    payload={"test": "test 1"},\n                    additional_metadata=create_additional_metadata(),\n                ),\n                BulkPushEventWithMetadata(\n                    key="otel:event",\n                    payload={"test": "test 2"},\n                    additional_metadata=create_additional_metadata(),\n                ),\n            ],\n        )\n\ndef run_workflow() -> None:\n    print("\\nrun_workflow")\n    with tracer.start_as_current_span("run_workflow"):\n        otel_workflow.run(\n            options=TriggerWorkflowOptions(\n                additional_metadata=create_additional_metadata()\n            ),\n        )\n\nasync def async_run_workflow() -> None:\n    print("\\nasync_run_workflow")\n    with tracer.start_as_current_span("async_run_workflow"):\n        await otel_workflow.aio_run(\n            options=TriggerWorkflowOptions(\n                additional_metadata=create_additional_metadata()\n            ),\n        )\n\ndef run_workflows() -> None:\n    print("\\nrun_workflows")\n    with tracer.start_as_current_span("run_workflows"):\n        otel_workflow.run_many(\n            [\n                otel_workflow.create_bulk_run_item(\n                    options=TriggerWorkflowOptions(\n                        additional_metadata=create_additional_metadata()\n                    )\n                ),\n                otel_workflow.create_bulk_run_item(\n                    options=TriggerWorkflowOptions(\n                        additional_metadata=create_additional_metadata()\n                    )\n                ),\n            ],\n        )\n\nasync def async_run_workflows() -> None:\n    print("\\nasync_run_workflows")\n    with tracer.start_as_current_span("async_run_workflows"):\n        await otel_workflow.aio_run_many(\n            [\n                otel_workflow.create_bulk_run_item(\n                    options=TriggerWorkflowOptions(\n                        additional_metadata=create_additional_metadata()\n                    )\n                ),\n                otel_workflow.create_bulk_run_item(\n                    options=TriggerWorkflowOptions(\n                        additional_metadata=create_additional_metadata()\n                    )\n                ),\n            ],\n        )\n\nasync def main() -> None:\n    push_event()\n    await async_push_event()\n    bulk_push_event()\n    await async_bulk_push_event()\n    run_workflow()\n    # await async_run_workflow()\n    run_workflows()\n    # await async_run_workflows()\n\nif __name__ == "__main__":\n    asyncio.run(main())\n',
      language: 'py',
      source: 'examples/python/opentelemetry_instrumentation/triggers.py',
      highlights: {},
    },
  L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3B5dGhvbi9vcGVudGVsZW1ldHJ5X2luc3RydW1lbnRhdGlvbi93b3JrZXIucHk_:
    {
      content:
        'from examples.opentelemetry_instrumentation.client import hatchet\nfrom examples.opentelemetry_instrumentation.tracer import trace_provider\nfrom hatchet_sdk import Context, EmptyModel\nfrom hatchet_sdk.opentelemetry.instrumentor import HatchetInstrumentor\n\nHatchetInstrumentor(\n    tracer_provider=trace_provider,\n).instrument()\n\notel_workflow = hatchet.workflow(\n    name="OTelWorkflow",\n)\n\n@otel_workflow.task()\ndef your_spans_are_children_of_hatchet_span(\n    input: EmptyModel, ctx: Context\n) -> dict[str, str]:\n    with trace_provider.get_tracer(__name__).start_as_current_span("step1"):\n        print("executed step")\n        return {\n            "foo": "bar",\n        }\n\n@otel_workflow.task()\ndef your_spans_are_still_children_of_hatchet_span(\n    input: EmptyModel, ctx: Context\n) -> None:\n    with trace_provider.get_tracer(__name__).start_as_current_span("step2"):\n        raise Exception("Manually instrumented step failed failed")\n\n@otel_workflow.task()\ndef this_step_is_still_instrumented(input: EmptyModel, ctx: Context) -> dict[str, str]:\n    print("executed still-instrumented step")\n    return {\n        "still": "instrumented",\n    }\n\n@otel_workflow.task()\ndef this_step_is_also_still_instrumented(input: EmptyModel, ctx: Context) -> None:\n    raise Exception("Still-instrumented step failed")\n\ndef main() -> None:\n    worker = hatchet.worker("otel-example-worker", slots=1, workflows=[otel_workflow])\n    worker.start()\n\nif __name__ == "__main__":\n    main()\n',
      language: 'py',
      source: 'examples/python/opentelemetry_instrumentation/worker.py',
      highlights: {},
    },
  L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3B5dGhvbi9wcmlvcml0eS90ZXN0X3ByaW9yaXR5LnB5:
    {
      content:
        'import asyncio\nfrom datetime import datetime, timedelta\nfrom random import choice\nfrom typing import AsyncGenerator, Literal\nfrom uuid import uuid4\n\nimport pytest\nimport pytest_asyncio\nfrom pydantic import BaseModel\n\nfrom examples.priority.worker import DEFAULT_PRIORITY, SLEEP_TIME, priority_workflow\nfrom hatchet_sdk import Hatchet, ScheduleTriggerWorkflowOptions, TriggerWorkflowOptions\nfrom hatchet_sdk.clients.rest.models.v1_task_status import V1TaskStatus\n\nPriority = Literal["low", "medium", "high", "default"]\n\nclass RunPriorityStartedAt(BaseModel):\n    priority: Priority\n    started_at: datetime\n    finished_at: datetime\n\ndef priority_to_int(priority: Priority) -> int:\n    match priority:\n        case "high":\n            return 3\n        case "medium":\n            return 2\n        case "low":\n            return 1\n        case "default":\n            return DEFAULT_PRIORITY\n        case _:\n            raise ValueError(f"Invalid priority: {priority}")\n\n@pytest_asyncio.fixture(loop_scope="session", scope="function")\nasync def dummy_runs() -> None:\n    priority: Priority = "high"\n\n    await priority_workflow.aio_run_many_no_wait(\n        [\n            priority_workflow.create_bulk_run_item(\n                options=TriggerWorkflowOptions(\n                    priority=(priority_to_int(priority)),\n                    additional_metadata={\n                        "priority": priority,\n                        "key": ix,\n                        "type": "dummy",\n                    },\n                )\n            )\n            for ix in range(40)\n        ]\n    )\n\n    await asyncio.sleep(3)\n\n    return None\n\n@pytest.mark.asyncio()\nasync def test_priority(hatchet: Hatchet, dummy_runs: None) -> None:\n    test_run_id = str(uuid4())\n    choices: list[Priority] = ["low", "medium", "high", "default"]\n    N = 30\n\n    run_refs = await priority_workflow.aio_run_many_no_wait(\n        [\n            priority_workflow.create_bulk_run_item(\n                options=TriggerWorkflowOptions(\n                    priority=(priority_to_int(priority := choice(choices))),\n                    additional_metadata={\n                        "priority": priority,\n                        "key": ix,\n                        "test_run_id": test_run_id,\n                    },\n                )\n            )\n            for ix in range(N)\n        ]\n    )\n\n    await asyncio.gather(*[r.aio_result() for r in run_refs])\n\n    workflows = (\n        await hatchet.workflows.aio_list(workflow_name=priority_workflow.name)\n    ).rows\n\n    assert workflows\n\n    workflow = next((w for w in workflows if w.name == priority_workflow.name), None)\n\n    assert workflow\n\n    assert workflow.name == priority_workflow.name\n\n    runs = await hatchet.runs.aio_list(\n        workflow_ids=[workflow.metadata.id],\n        additional_metadata={\n            "test_run_id": test_run_id,\n        },\n        limit=1_000,\n    )\n\n    runs_ids_started_ats: list[RunPriorityStartedAt] = sorted(\n        [\n            RunPriorityStartedAt(\n                priority=(r.additional_metadata or {}).get("priority") or "low",\n                started_at=r.started_at or datetime.min,\n                finished_at=r.finished_at or datetime.min,\n            )\n            for r in runs.rows\n        ],\n        key=lambda x: x.started_at,\n    )\n\n    assert len(runs_ids_started_ats) == len(run_refs)\n    assert len(runs_ids_started_ats) == N\n\n    for i in range(len(runs_ids_started_ats) - 1):\n        curr = runs_ids_started_ats[i]\n        nxt = runs_ids_started_ats[i + 1]\n\n        """Run start times should be in order of priority"""\n        assert priority_to_int(curr.priority) >= priority_to_int(nxt.priority)\n\n        """Runs should proceed one at a time"""\n        assert curr.finished_at <= nxt.finished_at\n        assert nxt.finished_at >= nxt.started_at\n\n        """Runs should finish after starting (this is mostly a test for engine datetime handling bugs)"""\n        assert curr.finished_at >= curr.started_at\n\n@pytest.mark.asyncio()\nasync def test_priority_via_scheduling(hatchet: Hatchet, dummy_runs: None) -> None:\n    test_run_id = str(uuid4())\n    sleep_time = 3\n    n = 30\n    choices: list[Priority] = ["low", "medium", "high", "default"]\n    run_at = datetime.now() + timedelta(seconds=sleep_time)\n\n    versions = await asyncio.gather(\n        *[\n            priority_workflow.aio_schedule(\n                run_at=run_at,\n                options=ScheduleTriggerWorkflowOptions(\n                    priority=(priority_to_int(priority := choice(choices))),\n                    additional_metadata={\n                        "priority": priority,\n                        "key": ix,\n                        "test_run_id": test_run_id,\n                    },\n                ),\n            )\n            for ix in range(n)\n        ]\n    )\n\n    await asyncio.sleep(sleep_time * 2)\n\n    workflow_id = versions[0].workflow_id\n\n    attempts = 0\n\n    while True:\n        if attempts >= SLEEP_TIME * n * 2:\n            raise TimeoutError("Timed out waiting for runs to finish")\n\n        attempts += 1\n        await asyncio.sleep(1)\n        runs = await hatchet.runs.aio_list(\n            workflow_ids=[workflow_id],\n            additional_metadata={\n                "test_run_id": test_run_id,\n            },\n            limit=1_000,\n        )\n\n        if not runs.rows:\n            continue\n\n        if any(\n            r.status in [V1TaskStatus.FAILED, V1TaskStatus.CANCELLED] for r in runs.rows\n        ):\n            raise ValueError("One or more runs failed or were cancelled")\n\n        if all(r.status == V1TaskStatus.COMPLETED for r in runs.rows):\n            break\n\n    runs_ids_started_ats: list[RunPriorityStartedAt] = sorted(\n        [\n            RunPriorityStartedAt(\n                priority=(r.additional_metadata or {}).get("priority") or "low",\n                started_at=r.started_at or datetime.min,\n                finished_at=r.finished_at or datetime.min,\n            )\n            for r in runs.rows\n        ],\n        key=lambda x: x.started_at,\n    )\n\n    assert len(runs_ids_started_ats) == len(versions)\n\n    for i in range(len(runs_ids_started_ats) - 1):\n        curr = runs_ids_started_ats[i]\n        nxt = runs_ids_started_ats[i + 1]\n\n        """Run start times should be in order of priority"""\n        assert priority_to_int(curr.priority) >= priority_to_int(nxt.priority)\n\n        """Runs should proceed one at a time"""\n        assert curr.finished_at <= nxt.finished_at\n        assert nxt.finished_at >= nxt.started_at\n\n        """Runs should finish after starting (this is mostly a test for engine datetime handling bugs)"""\n        assert curr.finished_at >= curr.started_at\n\n@pytest_asyncio.fixture(loop_scope="session", scope="function")\nasync def crons(\n    hatchet: Hatchet, dummy_runs: None\n) -> AsyncGenerator[tuple[str, str, int], None]:\n    test_run_id = str(uuid4())\n    choices: list[Priority] = ["low", "medium", "high"]\n    n = 30\n\n    crons = await asyncio.gather(\n        *[\n            hatchet.cron.aio_create(\n                workflow_name=priority_workflow.name,\n                cron_name=f"{test_run_id}-cron-{i}",\n                expression="* * * * *",\n                input={},\n                additional_metadata={\n                    "trigger": "cron",\n                    "test_run_id": test_run_id,\n                    "priority": (priority := choice(choices)),\n                    "key": str(i),\n                },\n                priority=(priority_to_int(priority)),\n            )\n            for i in range(n)\n        ]\n    )\n\n    yield crons[0].workflow_id, test_run_id, n\n\n    await asyncio.gather(*[hatchet.cron.aio_delete(cron.metadata.id) for cron in crons])\n\ndef time_until_next_minute() -> float:\n    now = datetime.now()\n    next_minute = now.replace(second=0, microsecond=0, minute=now.minute + 1)\n\n    return (next_minute - now).total_seconds()\n\n@pytest.mark.asyncio()\nasync def test_priority_via_cron(hatchet: Hatchet, crons: tuple[str, str, int]) -> None:\n    workflow_id, test_run_id, n = crons\n\n    await asyncio.sleep(time_until_next_minute() + 10)\n\n    attempts = 0\n\n    while True:\n        if attempts >= SLEEP_TIME * n * 2:\n            raise TimeoutError("Timed out waiting for runs to finish")\n\n        attempts += 1\n        await asyncio.sleep(1)\n        runs = await hatchet.runs.aio_list(\n            workflow_ids=[workflow_id],\n            additional_metadata={\n                "test_run_id": test_run_id,\n            },\n            limit=1_000,\n        )\n\n        if not runs.rows:\n            continue\n\n        if any(\n            r.status in [V1TaskStatus.FAILED, V1TaskStatus.CANCELLED] for r in runs.rows\n        ):\n            raise ValueError("One or more runs failed or were cancelled")\n\n        if all(r.status == V1TaskStatus.COMPLETED for r in runs.rows):\n            break\n\n    runs_ids_started_ats: list[RunPriorityStartedAt] = sorted(\n        [\n            RunPriorityStartedAt(\n                priority=(r.additional_metadata or {}).get("priority") or "low",\n                started_at=r.started_at or datetime.min,\n                finished_at=r.finished_at or datetime.min,\n            )\n            for r in runs.rows\n        ],\n        key=lambda x: x.started_at,\n    )\n\n    assert len(runs_ids_started_ats) == n\n\n    for i in range(len(runs_ids_started_ats) - 1):\n        curr = runs_ids_started_ats[i]\n        nxt = runs_ids_started_ats[i + 1]\n\n        """Run start times should be in order of priority"""\n        assert priority_to_int(curr.priority) >= priority_to_int(nxt.priority)\n\n        """Runs should proceed one at a time"""\n        assert curr.finished_at <= nxt.finished_at\n        assert nxt.finished_at >= nxt.started_at\n\n        """Runs should finish after starting (this is mostly a test for engine datetime handling bugs)"""\n        assert curr.finished_at >= curr.started_at\n',
      language: 'py',
      source: 'examples/python/priority/test_priority.py',
      highlights: {},
    },
  L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3B5dGhvbi9wcmlvcml0eS90cmlnZ2VyLnB5:
    {
      content:
        'from datetime import datetime, timedelta\n\nfrom examples.priority.worker import priority_workflow\nfrom hatchet_sdk import ScheduleTriggerWorkflowOptions, TriggerWorkflowOptions\n\npriority_workflow.run_no_wait()\n\n# ❓ Runtime priority\nlow_prio = priority_workflow.run_no_wait(\n    options=TriggerWorkflowOptions(\n        ## 👀 Adding priority and key to metadata to show them in the dashboard\n        priority=1,\n        additional_metadata={"priority": "low", "key": 1},\n    )\n)\n\nhigh_prio = priority_workflow.run_no_wait(\n    options=TriggerWorkflowOptions(\n        ## 👀 Adding priority and key to metadata to show them in the dashboard\n        priority=3,\n        additional_metadata={"priority": "high", "key": 1},\n    )\n)\n\n# ❓ Scheduled priority\nschedule = priority_workflow.schedule(\n    run_at=datetime.now() + timedelta(minutes=1),\n    options=ScheduleTriggerWorkflowOptions(priority=3),\n)\n\ncron = priority_workflow.create_cron(\n    cron_name="my-scheduled-cron",\n    expression="0 * * * *",\n    priority=3,\n)\n\n# ❓ Default priority\nlow_prio = priority_workflow.run_no_wait(\n    options=TriggerWorkflowOptions(\n        ## 👀 Adding priority and key to metadata to show them in the dashboard\n        priority=1,\n        additional_metadata={"priority": "low", "key": 2},\n    )\n)\nhigh_prio = priority_workflow.run_no_wait(\n    options=TriggerWorkflowOptions(\n        ## 👀 Adding priority and key to metadata to show them in the dashboard\n        priority=3,\n        additional_metadata={"priority": "high", "key": 2},\n    )\n)\n',
      language: 'py',
      source: 'examples/python/priority/trigger.py',
      highlights: {},
    },
  L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3B5dGhvbi9wcmlvcml0eS93b3JrZXIucHk_:
    {
      content:
        'import time\n\nfrom hatchet_sdk import (\n    ConcurrencyExpression,\n    ConcurrencyLimitStrategy,\n    Context,\n    EmptyModel,\n    Hatchet,\n)\n\nhatchet = Hatchet(debug=True)\n\n# ❓ Default priority\nDEFAULT_PRIORITY = 1\nSLEEP_TIME = 0.25\n\npriority_workflow = hatchet.workflow(\n    name="PriorityWorkflow",\n    default_priority=DEFAULT_PRIORITY,\n    concurrency=ConcurrencyExpression(\n        max_runs=1,\n        expression="\'true\'",\n        limit_strategy=ConcurrencyLimitStrategy.GROUP_ROUND_ROBIN,\n    ),\n)\n\n@priority_workflow.task()\ndef priority_task(input: EmptyModel, ctx: Context) -> None:\n    print("Priority:", ctx.priority)\n    time.sleep(SLEEP_TIME)\n\ndef main() -> None:\n    worker = hatchet.worker(\n        "priority-worker",\n        slots=1,\n        workflows=[priority_workflow],\n    )\n\n    worker.start()\n\nif __name__ == "__main__":\n    main()\n',
      language: 'py',
      source: 'examples/python/priority/worker.py',
      highlights: {},
    },
  L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3B5dGhvbi9yYXRlX2xpbWl0L2R5bmFtaWMucHk_:
    {
      content:
        'from pydantic import BaseModel\n\nfrom hatchet_sdk import Context, Hatchet\nfrom hatchet_sdk.rate_limit import RateLimit\n\nhatchet = Hatchet(debug=True)\n\nclass DynamicRateLimitInput(BaseModel):\n    group: str\n    units: int\n    limit: int\n\ndynamic_rate_limit_workflow = hatchet.workflow(\n    name="DynamicRateLimitWorkflow", input_validator=DynamicRateLimitInput\n)\n\n@dynamic_rate_limit_workflow.task(\n    rate_limits=[\n        RateLimit(\n            dynamic_key=\'"LIMIT:"+input.group\',\n            units="input.units",\n            limit="input.limit",\n        )\n    ]\n)\ndef step1(input: DynamicRateLimitInput, ctx: Context) -> None:\n    print("executed step1")\n\ndef main() -> None:\n    worker = hatchet.worker(\n        "rate-limit-worker", slots=10, workflows=[dynamic_rate_limit_workflow]\n    )\n    worker.start()\n',
      language: 'py',
      source: 'examples/python/rate_limit/dynamic.py',
      highlights: {},
    },
  L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3B5dGhvbi9yYXRlX2xpbWl0L3Rlc3RfcmF0ZV9saW1pdC5weQ__:
    {
      content:
        'import asyncio\nimport time\n\nimport pytest\n\nfrom examples.rate_limit.worker import rate_limit_workflow\n\n@pytest.mark.skip(reason="The timing for this test is not reliable")\n@pytest.mark.asyncio(loop_scope="session")\nasync def test_run() -> None:\n\n    run1 = rate_limit_workflow.run_no_wait()\n    run2 = rate_limit_workflow.run_no_wait()\n    run3 = rate_limit_workflow.run_no_wait()\n\n    start_time = time.time()\n\n    await asyncio.gather(run1.aio_result(), run2.aio_result(), run3.aio_result())\n\n    end_time = time.time()\n\n    total_time = end_time - start_time\n\n    assert (\n        1 <= total_time <= 5\n    ), f"Expected runtime to be a bit more than 1 seconds, but it took {total_time:.2f} seconds"\n',
      language: 'py',
      source: 'examples/python/rate_limit/test_rate_limit.py',
      highlights: {},
    },
  L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3B5dGhvbi9yYXRlX2xpbWl0L3RyaWdnZXIucHk_:
    {
      content:
        'from examples.rate_limit.worker import rate_limit_workflow\nfrom hatchet_sdk.hatchet import Hatchet\n\nhatchet = Hatchet(debug=True)\n\nrate_limit_workflow.run()\nrate_limit_workflow.run()\nrate_limit_workflow.run()\n',
      language: 'py',
      source: 'examples/python/rate_limit/trigger.py',
      highlights: {},
    },
  L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3B5dGhvbi9yYXRlX2xpbWl0L3dvcmtlci5weQ__:
    {
      content:
        'from pydantic import BaseModel\n\nfrom hatchet_sdk import Context, Hatchet\nfrom hatchet_sdk.rate_limit import RateLimit, RateLimitDuration\n\nhatchet = Hatchet(debug=True)\n\n# ❓ Workflow\nclass RateLimitInput(BaseModel):\n    user_id: str\n\nrate_limit_workflow = hatchet.workflow(\n    name="RateLimitWorkflow", input_validator=RateLimitInput\n)\n\n# ❓ Static\nRATE_LIMIT_KEY = "test-limit"\n\n@rate_limit_workflow.task(rate_limits=[RateLimit(static_key=RATE_LIMIT_KEY, units=1)])\ndef step_1(input: RateLimitInput, ctx: Context) -> None:\n    print("executed step_1")\n\n# ❓ Dynamic\n\n@rate_limit_workflow.task(\n    rate_limits=[\n        RateLimit(\n            dynamic_key="input.user_id",\n            units=1,\n            limit=10,\n            duration=RateLimitDuration.MINUTE,\n        )\n    ]\n)\ndef step_2(input: RateLimitInput, ctx: Context) -> None:\n    print("executed step_2")\n\ndef main() -> None:\n    hatchet.rate_limits.put(RATE_LIMIT_KEY, 2, RateLimitDuration.SECOND)\n\n    worker = hatchet.worker(\n        "rate-limit-worker", slots=10, workflows=[rate_limit_workflow]\n    )\n\n    worker.start()\n\nif __name__ == "__main__":\n    main()\n',
      language: 'py',
      source: 'examples/python/rate_limit/worker.py',
      highlights: {},
    },
  L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3B5dGhvbi9yZXRyaWVzL3dvcmtlci5weQ__:
    {
      content:
        'from hatchet_sdk import Context, EmptyModel, Hatchet\n\nhatchet = Hatchet(debug=True)\n\nsimple_workflow = hatchet.workflow(name="SimpleRetryWorkflow")\nbackoff_workflow = hatchet.workflow(name="BackoffWorkflow")\n\n# ❓ Simple Step Retries\n@simple_workflow.task(retries=3)\ndef always_fail(input: EmptyModel, ctx: Context) -> dict[str, str]:\n    raise Exception("simple task failed")\n\n# ‼️\n\n# ❓ Retries with Count\n@simple_workflow.task(retries=3)\ndef fail_twice(input: EmptyModel, ctx: Context) -> dict[str, str]:\n    if ctx.retry_count < 2:\n        raise Exception("simple task failed")\n\n    return {"status": "success"}\n\n# ‼️\n\n# ❓ Retries with Backoff\n@backoff_workflow.task(\n    retries=10,\n    # 👀 Maximum number of seconds to wait between retries\n    backoff_max_seconds=10,\n    # 👀 Factor to increase the wait time between retries.\n    # This sequence will be 2s, 4s, 8s, 10s, 10s, 10s... due to the maxSeconds limit\n    backoff_factor=2.0,\n)\ndef backoff_task(input: EmptyModel, ctx: Context) -> dict[str, str]:\n    if ctx.retry_count < 3:\n        raise Exception("backoff task failed")\n\n    return {"status": "success"}\n\n# ‼️\n\ndef main() -> None:\n    worker = hatchet.worker("backoff-worker", slots=4, workflows=[backoff_workflow])\n    worker.start()\n\nif __name__ == "__main__":\n    main()\n',
      language: 'py',
      source: 'examples/python/retries/worker.py',
      highlights: {},
    },
  L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3B5dGhvbi9zY2hlZHVsZWQvcHJvZ3JhbWF0aWMtYXN5bmMucHk_:
    {
      content:
        'from datetime import datetime, timedelta\n\nfrom hatchet_sdk import Hatchet\n\nhatchet = Hatchet()\n\nasync def create_scheduled() -> None:\n    # ❓ Create\n    scheduled_run = await hatchet.scheduled.aio_create(\n        workflow_name="simple-workflow",\n        trigger_at=datetime.now() + timedelta(seconds=10),\n        input={\n            "data": "simple-workflow-data",\n        },\n        additional_metadata={\n            "customer_id": "customer-a",\n        },\n    )\n\n    scheduled_run.metadata.id  # the id of the scheduled run trigger\n\n    # ❓ Delete\n    await hatchet.scheduled.aio_delete(scheduled_id=scheduled_run.metadata.id)\n\n    # ❓ List\n    await hatchet.scheduled.aio_list()\n\n    # ❓ Get\n    scheduled_run = await hatchet.scheduled.aio_get(\n        scheduled_id=scheduled_run.metadata.id\n    )\n',
      language: 'py',
      source: 'examples/python/scheduled/programatic-async.py',
      highlights: {},
    },
  L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3B5dGhvbi9zY2hlZHVsZWQvcHJvZ3JhbWF0aWMtc3luYy5weQ__:
    {
      content:
        'from datetime import datetime, timedelta\n\nfrom hatchet_sdk import Hatchet\n\nhatchet = Hatchet()\n\n# ❓ Create\nscheduled_run = hatchet.scheduled.create(\n    workflow_name="simple-workflow",\n    trigger_at=datetime.now() + timedelta(seconds=10),\n    input={\n        "data": "simple-workflow-data",\n    },\n    additional_metadata={\n        "customer_id": "customer-a",\n    },\n)\n\nid = scheduled_run.metadata.id  # the id of the scheduled run trigger\n\n# ❓ Delete\nhatchet.scheduled.delete(scheduled_id=scheduled_run.metadata.id)\n\n# ❓ List\nscheduled_runs = hatchet.scheduled.list()\n\n# ❓ Get\nscheduled_run = hatchet.scheduled.get(scheduled_id=scheduled_run.metadata.id)\n',
      language: 'py',
      source: 'examples/python/scheduled/programatic-sync.py',
      highlights: {},
    },
  L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3B5dGhvbi9zaW1wbGUvdHJpZ2dlci5weQ__:
    {
      content: 'from examples.simple.worker import step1\n\nstep1.run()\n',
      language: 'py',
      source: 'examples/python/simple/trigger.py',
      highlights: {},
    },
  L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3B5dGhvbi9zaW1wbGUvd29ya2VyLnB5:
    {
      content:
        '# ❓ Simple\n\nfrom hatchet_sdk import Context, EmptyModel, Hatchet\n\nhatchet = Hatchet(debug=True)\n\n@hatchet-dev/typescript-sdk.task(name="SimpleWorkflow")\ndef step1(input: EmptyModel, ctx: Context) -> None:\n    print("executed step1")\n\ndef main() -> None:\n    worker = hatchet.worker("test-worker", slots=1, workflows=[step1])\n    worker.start()\n\n# ‼️\n\nif __name__ == "__main__":\n    main()\n',
      language: 'py',
      source: 'examples/python/simple/worker.py',
      highlights: {},
    },
  L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3B5dGhvbi9zdGlja3lfd29ya2Vycy9ldmVudC5weQ__:
    {
      content:
        'from examples.sticky_workers.worker import sticky_workflow\nfrom hatchet_sdk import TriggerWorkflowOptions\n\nsticky_workflow.run(\n    options=TriggerWorkflowOptions(additional_metadata={"hello": "moon"}),\n)\n',
      language: 'py',
      source: 'examples/python/sticky_workers/event.py',
      highlights: {},
    },
  L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3B5dGhvbi9zdGlja3lfd29ya2Vycy93b3JrZXIucHk_:
    {
      content:
        'from hatchet_sdk import (\n    Context,\n    EmptyModel,\n    Hatchet,\n    StickyStrategy,\n    TriggerWorkflowOptions,\n)\n\nhatchet = Hatchet(debug=True)\n\n# ❓ StickyWorker\n\nsticky_workflow = hatchet.workflow(\n    name="StickyWorkflow",\n    # 👀 Specify a sticky strategy when declaring the workflow\n    sticky=StickyStrategy.SOFT,\n)\n\n@sticky_workflow.task()\ndef step1a(input: EmptyModel, ctx: Context) -> dict[str, str | None]:\n    return {"worker": ctx.worker.id()}\n\n@sticky_workflow.task()\ndef step1b(input: EmptyModel, ctx: Context) -> dict[str, str | None]:\n    return {"worker": ctx.worker.id()}\n\n# ‼️\n\n# ❓ StickyChild\n\nsticky_child_workflow = hatchet.workflow(\n    name="StickyChildWorkflow", sticky=StickyStrategy.SOFT\n)\n\n@sticky_workflow.task(parents=[step1a, step1b])\nasync def step2(input: EmptyModel, ctx: Context) -> dict[str, str | None]:\n    ref = await sticky_child_workflow.aio_run_no_wait(\n        options=TriggerWorkflowOptions(sticky=True)\n    )\n\n    await ref.aio_result()\n\n    return {"worker": ctx.worker.id()}\n\n@sticky_child_workflow.task()\ndef child(input: EmptyModel, ctx: Context) -> dict[str, str | None]:\n    return {"worker": ctx.worker.id()}\n\n# ‼️\n\ndef main() -> None:\n    worker = hatchet.worker(\n        "sticky-worker", slots=10, workflows=[sticky_workflow, sticky_child_workflow]\n    )\n    worker.start()\n\nif __name__ == "__main__":\n    main()\n',
      language: 'py',
      source: 'examples/python/sticky_workers/worker.py',
      highlights: {},
    },
  L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3B5dGhvbi9zdHJlYW1pbmcvYXN5bmNfc3RyZWFtLnB5:
    {
      content:
        'import asyncio\n\nfrom examples.streaming.worker import streaming_workflow\n\nasync def main() -> None:\n    ref = await streaming_workflow.aio_run_no_wait()\n    await asyncio.sleep(1)\n\n    stream = ref.stream()\n\n    async for chunk in stream:\n        print(chunk)\n\nif __name__ == "__main__":\n    import asyncio\n\n    asyncio.run(main())\n',
      language: 'py',
      source: 'examples/python/streaming/async_stream.py',
      highlights: {},
    },
  L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3B5dGhvbi9zdHJlYW1pbmcvc3luY19zdHJlYW0ucHk_:
    {
      content:
        'import time\n\nfrom examples.streaming.worker import streaming_workflow\n\ndef main() -> None:\n    ref = streaming_workflow.run_no_wait()\n    time.sleep(1)\n\n    stream = ref.stream()\n\n    for chunk in stream:\n        print(chunk)\n\nif __name__ == "__main__":\n    main()\n',
      language: 'py',
      source: 'examples/python/streaming/sync_stream.py',
      highlights: {},
    },
  L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3B5dGhvbi9zdHJlYW1pbmcvd29ya2VyLnB5:
    {
      content:
        'import asyncio\n\nfrom hatchet_sdk import Context, EmptyModel, Hatchet\n\nhatchet = Hatchet(debug=True)\n\n# ❓ Streaming\n\nstreaming_workflow = hatchet.workflow(name="StreamingWorkflow")\n\n@streaming_workflow.task()\nasync def step1(input: EmptyModel, ctx: Context) -> None:\n    for i in range(10):\n        await asyncio.sleep(1)\n        ctx.put_stream(f"Processing {i}")\n\ndef main() -> None:\n    worker = hatchet.worker("test-worker", workflows=[streaming_workflow])\n    worker.start()\n\n# ‼️\n\nif __name__ == "__main__":\n    main()\n',
      language: 'py',
      source: 'examples/python/streaming/worker.py',
      highlights: {},
    },
  L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3B5dGhvbi90aW1lb3V0L3Rlc3RfdGltZW91dC5weQ__:
    {
      content:
        'import pytest\n\nfrom examples.timeout.worker import refresh_timeout_wf, timeout_wf\n\n@pytest.mark.asyncio(loop_scope="session")\nasync def test_execution_timeout() -> None:\n    run = timeout_wf.run_no_wait()\n\n    with pytest.raises(Exception, match="(Task exceeded timeout|TIMED_OUT)"):\n        await run.aio_result()\n\n@pytest.mark.asyncio(loop_scope="session")\nasync def test_run_refresh_timeout() -> None:\n    result = await refresh_timeout_wf.aio_run()\n\n    assert result["refresh_task"]["status"] == "success"\n',
      language: 'py',
      source: 'examples/python/timeout/test_timeout.py',
      highlights: {},
    },
  L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3B5dGhvbi90aW1lb3V0L3RyaWdnZXIucHk_:
    {
      content:
        'from examples.timeout.worker import refresh_timeout_wf, timeout_wf\n\ntimeout_wf.run()\nrefresh_timeout_wf.run()\n',
      language: 'py',
      source: 'examples/python/timeout/trigger.py',
      highlights: {},
    },
  L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3B5dGhvbi90aW1lb3V0L3dvcmtlci5weQ__:
    {
      content:
        'import time\nfrom datetime import timedelta\n\nfrom hatchet_sdk import Context, EmptyModel, Hatchet, TaskDefaults\n\nhatchet = Hatchet(debug=True)\n\n# ❓ ScheduleTimeout\ntimeout_wf = hatchet.workflow(\n    name="TimeoutWorkflow",\n    task_defaults=TaskDefaults(execution_timeout=timedelta(minutes=2)),\n)\n# ‼️\n\n# ❓ ExecutionTimeout\n# 👀 Specify an execution timeout on a task\n@timeout_wf.task(\n    execution_timeout=timedelta(seconds=4), schedule_timeout=timedelta(minutes=10)\n)\ndef timeout_task(input: EmptyModel, ctx: Context) -> dict[str, str]:\n    time.sleep(5)\n    return {"status": "success"}\n\n# ‼️\n\nrefresh_timeout_wf = hatchet.workflow(name="RefreshTimeoutWorkflow")\n\n# ❓ RefreshTimeout\n@refresh_timeout_wf.task(execution_timeout=timedelta(seconds=4))\ndef refresh_task(input: EmptyModel, ctx: Context) -> dict[str, str]:\n\n    ctx.refresh_timeout(timedelta(seconds=10))\n    time.sleep(5)\n\n    return {"status": "success"}\n\n# ‼️\n\ndef main() -> None:\n    worker = hatchet.worker(\n        "timeout-worker", slots=4, workflows=[timeout_wf, refresh_timeout_wf]\n    )\n\n    worker.start()\n\nif __name__ == "__main__":\n    main()\n',
      language: 'py',
      source: 'examples/python/timeout/worker.py',
      highlights: {},
    },
  L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3B5dGhvbi93YWl0cy90ZXN0X3dhaXRzLnB5:
    {
      content:
        'import asyncio\nimport os\n\nimport pytest\n\nfrom examples.waits.worker import task_condition_workflow\nfrom hatchet_sdk import Hatchet\n\n@pytest.mark.skipif(\n    os.getenv("CI", "false").lower() == "true",\n    reason="Skipped in CI because of unreliability",\n)\n@pytest.mark.asyncio(loop_scope="session")\nasync def test_waits(hatchet: Hatchet) -> None:\n\n    ref = task_condition_workflow.run_no_wait()\n\n    await asyncio.sleep(15)\n\n    hatchet.event.push("skip_on_event:skip", {})\n    hatchet.event.push("wait_for_event:start", {})\n\n    result = await ref.aio_result()\n\n    assert result["skip_on_event"] == {"skipped": True}\n\n    first_random_number = result["start"]["random_number"]\n    wait_for_event_random_number = result["wait_for_event"]["random_number"]\n    wait_for_sleep_random_number = result["wait_for_sleep"]["random_number"]\n\n    left_branch = result["left_branch"]\n    right_branch = result["right_branch"]\n\n    assert left_branch.get("skipped") is True or right_branch.get("skipped") is True\n\n    branch_random_number = left_branch.get("random_number") or right_branch.get(\n        "random_number"\n    )\n\n    result_sum = result["sum"]["sum"]\n\n    assert (\n        result_sum\n        == first_random_number\n        + wait_for_event_random_number\n        + wait_for_sleep_random_number\n        + branch_random_number\n    )\n',
      language: 'py',
      source: 'examples/python/waits/test_waits.py',
      highlights: {},
    },
  L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3B5dGhvbi93YWl0cy90cmlnZ2VyLnB5:
    {
      content:
        'import time\n\nfrom examples.waits.worker import hatchet, task_condition_workflow\n\ntask_condition_workflow.run_no_wait()\n\ntime.sleep(5)\n\nhatchet.event.push("skip_on_event:skip", {})\nhatchet.event.push("wait_for_event:start", {})\n',
      language: 'py',
      source: 'examples/python/waits/trigger.py',
      highlights: {},
    },
  L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3B5dGhvbi93YWl0cy93b3JrZXIucHk_:
    {
      content:
        '# ❓ Create a workflow\n\nimport random\nfrom datetime import timedelta\n\nfrom pydantic import BaseModel\n\nfrom hatchet_sdk import (\n    Context,\n    EmptyModel,\n    Hatchet,\n    ParentCondition,\n    SleepCondition,\n    UserEventCondition,\n    or_,\n)\n\nhatchet = Hatchet(debug=True)\n\nclass StepOutput(BaseModel):\n    random_number: int\n\nclass RandomSum(BaseModel):\n    sum: int\n\ntask_condition_workflow = hatchet.workflow(name="TaskConditionWorkflow")\n\n# ❓ Add base task\n@task_condition_workflow.task()\ndef start(input: EmptyModel, ctx: Context) -> StepOutput:\n    return StepOutput(random_number=random.randint(1, 100))\n\n# ❓ Add wait for sleep\n@task_condition_workflow.task(\n    parents=[start], wait_for=[SleepCondition(timedelta(seconds=10))]\n)\ndef wait_for_sleep(input: EmptyModel, ctx: Context) -> StepOutput:\n    return StepOutput(random_number=random.randint(1, 100))\n\n# ❓ Add skip on event\n@task_condition_workflow.task(\n    parents=[start],\n    wait_for=[SleepCondition(timedelta(seconds=30))],\n    skip_if=[UserEventCondition(event_key="skip_on_event:skip")],\n)\ndef skip_on_event(input: EmptyModel, ctx: Context) -> StepOutput:\n    return StepOutput(random_number=random.randint(1, 100))\n\n# ❓ Add branching\n@task_condition_workflow.task(\n    parents=[wait_for_sleep],\n    skip_if=[\n        ParentCondition(\n            parent=wait_for_sleep,\n            expression="output.random_number > 50",\n        )\n    ],\n)\ndef left_branch(input: EmptyModel, ctx: Context) -> StepOutput:\n    return StepOutput(random_number=random.randint(1, 100))\n\n@task_condition_workflow.task(\n    parents=[wait_for_sleep],\n    skip_if=[\n        ParentCondition(\n            parent=wait_for_sleep,\n            expression="output.random_number <= 50",\n        )\n    ],\n)\ndef right_branch(input: EmptyModel, ctx: Context) -> StepOutput:\n    return StepOutput(random_number=random.randint(1, 100))\n\n# ❓ Add wait for event\n@task_condition_workflow.task(\n    parents=[start],\n    wait_for=[\n        or_(\n            SleepCondition(duration=timedelta(minutes=1)),\n            UserEventCondition(event_key="wait_for_event:start"),\n        )\n    ],\n)\ndef wait_for_event(input: EmptyModel, ctx: Context) -> StepOutput:\n    return StepOutput(random_number=random.randint(1, 100))\n\n# ❓ Add sum\n@task_condition_workflow.task(\n    parents=[\n        start,\n        wait_for_sleep,\n        wait_for_event,\n        skip_on_event,\n        left_branch,\n        right_branch,\n    ],\n)\ndef sum(input: EmptyModel, ctx: Context) -> RandomSum:\n    one = ctx.task_output(start).random_number\n    two = ctx.task_output(wait_for_event).random_number\n    three = ctx.task_output(wait_for_sleep).random_number\n    four = (\n        ctx.task_output(skip_on_event).random_number\n        if not ctx.was_skipped(skip_on_event)\n        else 0\n    )\n\n    five = (\n        ctx.task_output(left_branch).random_number\n        if not ctx.was_skipped(left_branch)\n        else 0\n    )\n    six = (\n        ctx.task_output(right_branch).random_number\n        if not ctx.was_skipped(right_branch)\n        else 0\n    )\n\n    return RandomSum(sum=one + two + three + four + five + six)\n\ndef main() -> None:\n    worker = hatchet.worker("dag-worker", workflows=[task_condition_workflow])\n\n    worker.start()\n\nif __name__ == "__main__":\n    main()\n',
      language: 'py',
      source: 'examples/python/waits/worker.py',
      highlights: {},
    },
  L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3B5dGhvbi93b3JrZXIucHk_:
    {
      content:
        'from examples.affinity_workers.worker import affinity_worker_workflow\nfrom examples.bulk_fanout.worker import bulk_child_wf, bulk_parent_wf\nfrom examples.cancellation.worker import cancellation_workflow\nfrom examples.concurrency_limit.worker import concurrency_limit_workflow\nfrom examples.concurrency_limit_rr.worker import concurrency_limit_rr_workflow\nfrom examples.concurrency_multiple_keys.worker import concurrency_multiple_keys_workflow\nfrom examples.concurrency_workflow_level.worker import (\n    concurrency_workflow_level_workflow,\n)\nfrom examples.dag.worker import dag_workflow\nfrom examples.dedupe.worker import dedupe_child_wf, dedupe_parent_wf\nfrom examples.durable.worker import durable_workflow\nfrom examples.fanout.worker import child_wf, parent_wf\nfrom examples.fanout_sync.worker import sync_fanout_child, sync_fanout_parent\nfrom examples.lifespans.simple import lifespan, lifespan_task\nfrom examples.logger.workflow import logging_workflow\nfrom examples.non_retryable.worker import non_retryable_workflow\nfrom examples.on_failure.worker import on_failure_wf, on_failure_wf_with_details\nfrom examples.priority.worker import priority_workflow\nfrom examples.timeout.worker import refresh_timeout_wf, timeout_wf\nfrom examples.waits.worker import task_condition_workflow\nfrom hatchet_sdk import Hatchet\n\nhatchet = Hatchet(debug=True)\n\ndef main() -> None:\n    worker = hatchet.worker(\n        "e2e-test-worker",\n        slots=100,\n        workflows=[\n            affinity_worker_workflow,\n            bulk_child_wf,\n            bulk_parent_wf,\n            concurrency_limit_workflow,\n            concurrency_limit_rr_workflow,\n            concurrency_multiple_keys_workflow,\n            dag_workflow,\n            dedupe_child_wf,\n            dedupe_parent_wf,\n            durable_workflow,\n            child_wf,\n            parent_wf,\n            on_failure_wf,\n            on_failure_wf_with_details,\n            logging_workflow,\n            timeout_wf,\n            refresh_timeout_wf,\n            task_condition_workflow,\n            cancellation_workflow,\n            sync_fanout_parent,\n            sync_fanout_child,\n            non_retryable_workflow,\n            concurrency_workflow_level_workflow,\n            priority_workflow,\n            lifespan_task,\n        ],\n        lifespan=lifespan,\n    )\n\n    worker.start()\n\nif __name__ == "__main__":\n    main()\n',
      language: 'py',
      source: 'examples/python/worker.py',
      highlights: {},
    },
  L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3B5dGhvbi93b3JrZXJfZXhpc3RpbmdfbG9vcC93b3JrZXIucHk_:
    {
      content:
        'import asyncio\nfrom contextlib import suppress\n\nfrom hatchet_sdk import Context, EmptyModel, Hatchet\n\nhatchet = Hatchet(debug=True)\n\nexisting_loop_worker = hatchet.workflow(name="WorkerExistingLoopWorkflow")\n\n@existing_loop_worker.task()\nasync def task(input: EmptyModel, ctx: Context) -> dict[str, str]:\n    print("started")\n    await asyncio.sleep(10)\n    print("finished")\n    return {"result": "returned result"}\n\nasync def async_main() -> None:\n    worker = None\n    try:\n        worker = hatchet.worker(\n            "test-worker", slots=1, workflows=[existing_loop_worker]\n        )\n        worker.start()\n\n        ref = existing_loop_worker.run_no_wait()\n        print(await ref.aio_result())\n        while True:\n            await asyncio.sleep(1)\n    finally:\n        if worker:\n            await worker.exit_gracefully()\n\ndef main() -> None:\n    with suppress(KeyboardInterrupt):\n        asyncio.run(async_main())\n\nif __name__ == "__main__":\n    main()\n',
      language: 'py',
      source: 'examples/python/worker_existing_loop/worker.py',
      highlights: {},
    },
  L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3B5dGhvbi93b3JrZmxvd19yZWdpc3RyYXRpb24vd29ya2VyLnB5:
    {
      content:
        '# ❓ WorkflowRegistration\n\nfrom hatchet_sdk import Hatchet\n\nhatchet = Hatchet(debug=True)\n\nwf_one = hatchet.workflow(name="wf_one")\nwf_two = hatchet.workflow(name="wf_two")\nwf_three = hatchet.workflow(name="wf_three")\nwf_four = hatchet.workflow(name="wf_four")\nwf_five = hatchet.workflow(name="wf_five")\n\n# define tasks here\n\ndef main() -> None:\n    # 👀 Register workflows directly when instantiating the worker\n    worker = hatchet.worker("test-worker", slots=1, workflows=[wf_one, wf_two])\n\n    # 👀 Register a single workflow after instantiating the worker\n    worker.register_workflow(wf_three)\n\n    # 👀 Register multiple workflows in bulk after instantiating the worker\n    worker.register_workflows([wf_four, wf_five])\n\n    worker.start()\n\n# ‼️\n\nif __name__ == "__main__":\n    main()\n',
      language: 'py',
      source: 'examples/python/workflow_registration/worker.py',
      highlights: {},
    },
  L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL2dvL21pZ3JhdGlvbi1ndWlkZXMvaGF0Y2hldC1jbGllbnQuZ28_:
    {
      content:
        'package migration_guides\n\nimport (\n\tv1 "github.com/hatchet-dev/hatchet/pkg/v1"\n)\n\nfunc HatchetClient() (v1.HatchetClient, error) {\n\thatchet, err := v1.NewHatchetClient()\n\n\tif err != nil {\n\t\treturn nil, err\n\t}\n\n\treturn hatchet, nil\n}\n',
      language: 'go',
      source: 'examples/go/migration-guides/hatchet-client.go',
      highlights: {},
    },
  L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL2dvL21pZ3JhdGlvbi1ndWlkZXMvbWVyZ2VudC5nbw__:
    {
      content:
        'package migration_guides\n\nimport (\n\t"bytes"\n\t"context"\n\t"encoding/json"\n\t"fmt"\n\t"net/http"\n\t"time"\n\n\t"github.com/hatchet-dev/hatchet/pkg/client/create"\n\tv1 "github.com/hatchet-dev/hatchet/pkg/v1"\n\t"github.com/hatchet-dev/hatchet/pkg/v1/factory"\n\tv1worker "github.com/hatchet-dev/hatchet/pkg/v1/worker"\n\t"github.com/hatchet-dev/hatchet/pkg/v1/workflow"\n\t"github.com/hatchet-dev/hatchet/pkg/worker"\n)\n\n// ProcessImage simulates image processing\nfunc ProcessImage(imageURL string, filters []string) (map[string]interface{}, error) {\n\t// Do some image processing\n\treturn map[string]interface{}{\n\t\t"url":    imageURL,\n\t\t"size":   100,\n\t\t"format": "png",\n\t}, nil\n}\n\n// ❓ Before (Mergent)\ntype MergentRequest struct {\n\tImageURL string   `json:"image_url"`\n\tFilters  []string `json:"filters"`\n}\n\ntype MergentResponse struct {\n\tSuccess      bool   `json:"success"`\n\tProcessedURL string `json:"processed_url"`\n}\n\nfunc ProcessImageMergent(req MergentRequest) (*MergentResponse, error) {\n\tresult, err := ProcessImage(req.ImageURL, req.Filters)\n\tif err != nil {\n\t\treturn nil, err\n\t}\n\n\treturn &MergentResponse{\n\t\tSuccess:      true,\n\t\tProcessedURL: result["url"].(string),\n\t}, nil\n}\n\n// !!\n\n// ❓ After (Hatchet)\ntype ImageProcessInput struct {\n\tImageURL string   `json:"image_url"`\n\tFilters  []string `json:"filters"`\n}\n\ntype ImageProcessOutput struct {\n\tProcessedURL string `json:"processed_url"`\n\tMetadata     struct {\n\t\tSize           int      `json:"size"`\n\t\tFormat         string   `json:"format"`\n\t\tAppliedFilters []string `json:"applied_filters"`\n\t} `json:"metadata"`\n}\n\nfunc ImageProcessor(hatchet v1.HatchetClient) workflow.WorkflowDeclaration[ImageProcessInput, ImageProcessOutput] {\n\tprocessor := factory.NewTask(\n\t\tcreate.StandaloneTask{\n\t\t\tName: "image-processor",\n\t\t},\n\t\tfunc(ctx worker.HatchetContext, input ImageProcessInput) (*ImageProcessOutput, error) {\n\t\t\tresult, err := ProcessImage(input.ImageURL, input.Filters)\n\t\t\tif err != nil {\n\t\t\t\treturn nil, fmt.Errorf("processing image: %w", err)\n\t\t\t}\n\n\t\t\tif result["url"] == "" {\n\t\t\t\treturn nil, fmt.Errorf("processing failed to generate URL")\n\t\t\t}\n\n\t\t\toutput := &ImageProcessOutput{\n\t\t\t\tProcessedURL: result["url"].(string),\n\t\t\t\tMetadata: struct {\n\t\t\t\t\tSize           int      `json:"size"`\n\t\t\t\t\tFormat         string   `json:"format"`\n\t\t\t\t\tAppliedFilters []string `json:"applied_filters"`\n\t\t\t\t}{\n\t\t\t\t\tSize:           result["size"].(int),\n\t\t\t\t\tFormat:         result["format"].(string),\n\t\t\t\t\tAppliedFilters: input.Filters,\n\t\t\t\t},\n\t\t\t}\n\n\t\t\treturn output, nil\n\t\t},\n\t\thatchet,\n\t)\n\t// !!\n\n\t// Example of running a task\n\t_ = func() error {\n\t\t// ❓ Running a task\n\t\tresult, err := processor.Run(context.Background(), ImageProcessInput{\n\t\t\tImageURL: "https://example.com/image.png",\n\t\t\tFilters:  []string{"blur"},\n\t\t})\n\t\tif err != nil {\n\t\t\treturn err\n\t\t}\n\t\tfmt.Printf("Result: %+v\\n", result)\n\t\treturn nil\n\t\t// !!\n\t}\n\n\t// Example of registering a task on a worker\n\t_ = func() error {\n\t\t// ❓ Declaring a Worker\n\t\tw, err := hatchet.Worker(v1worker.WorkerOpts{\n\t\t\tName: "image-processor-worker",\n\t\t\tWorkflows: []workflow.WorkflowBase{\n\t\t\t\tprocessor,\n\t\t\t},\n\t\t})\n\t\tif err != nil {\n\t\t\treturn err\n\t\t}\n\t\terr = w.StartBlocking(context.Background())\n\t\tif err != nil {\n\t\t\treturn err\n\t\t}\n\t\treturn nil\n\t\t// !!\n\t}\n\n\treturn processor\n}\n\nfunc RunMergentTask() error {\n\n\treturn nil\n}\n\nfunc RunningTasks(hatchet v1.HatchetClient) error {\n\t// ❓ Running a task (Mergent)\n\ttask := struct {\n\t\tRequest struct {\n\t\t\tURL     string            `json:"url"`\n\t\t\tBody    string            `json:"body"`\n\t\t\tHeaders map[string]string `json:"headers"`\n\t\t} `json:"request"`\n\t\tName  string `json:"name"`\n\t\tQueue string `json:"queue"`\n\t}{\n\t\tRequest: struct {\n\t\t\tURL     string            `json:"url"`\n\t\t\tBody    string            `json:"body"`\n\t\t\tHeaders map[string]string `json:"headers"`\n\t\t}{\n\t\t\tURL: "https://example.com",\n\t\t\tHeaders: map[string]string{\n\t\t\t\t"Authorization": "fake-secret-token",\n\t\t\t\t"Content-Type":  "application/json",\n\t\t\t},\n\t\t\tBody: "Hello, world!",\n\t\t},\n\t\tName:  "4cf95241-fa19-47ef-8a67-71e483747649",\n\t\tQueue: "default",\n\t}\n\n\ttaskJSON, err := json.Marshal(task)\n\tif err != nil {\n\t\treturn fmt.Errorf("marshaling task: %w", err)\n\t}\n\n\treq, err := http.NewRequest(http.MethodPost, "https://api.mergent.co/v2/tasks", bytes.NewBuffer(taskJSON))\n\tif err != nil {\n\t\treturn fmt.Errorf("creating request: %w", err)\n\t}\n\n\treq.Header.Add("Authorization", "Bearer <API_KEY>")\n\treq.Header.Add("Content-Type", "application/json")\n\n\tclient := &http.Client{}\n\tres, err := client.Do(req)\n\tif err != nil {\n\t\treturn fmt.Errorf("sending request: %w", err)\n\t}\n\tdefer res.Body.Close()\n\n\tfmt.Printf("Mergent task created with status: %d\\n", res.StatusCode)\n\t// !!\n\n\t// ❓ Running a task (Hatchet)\n\tprocessor := ImageProcessor(hatchet)\n\n\tresult, err := processor.Run(context.Background(), ImageProcessInput{\n\t\tImageURL: "https://example.com/image.png",\n\t\tFilters:  []string{"blur"},\n\t})\n\tif err != nil {\n\t\treturn err\n\t}\n\tfmt.Printf("Result: %+v\\n", result)\n\t// !!\n\n\t// ❓ Scheduling tasks (Hatchet)\n\t// Schedule the task to run at a specific time\n\tscheduleRef, err := processor.Schedule(\n\t\tcontext.Background(),\n\t\ttime.Now().Add(time.Second*10),\n\t\tImageProcessInput{\n\t\t\tImageURL: "https://example.com/image.png",\n\t\t\tFilters:  []string{"blur"},\n\t\t},\n\t)\n\tif err != nil {\n\t\treturn err\n\t}\n\n\t// or schedule to run every hour\n\tcronRef, err := processor.Cron(\n\t\tcontext.Background(),\n\t\t"run-hourly",\n\t\t"0 * * * *",\n\t\tImageProcessInput{\n\t\t\tImageURL: "https://example.com/image.png",\n\t\t\tFilters:  []string{"blur"},\n\t\t},\n\t)\n\t// !!\n\tif err != nil {\n\t\treturn err\n\t}\n\n\tfmt.Printf("Scheduled tasks with refs: %+v, %+v\\n", scheduleRef, cronRef)\n\treturn nil\n}\n',
      language: 'go',
      source: 'examples/go/migration-guides/mergent.go',
      highlights: {},
    },
  L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL2dvL3J1bi9hbGwuZ28_:
    {
      content:
        'package main\n\nimport (\n\t"context"\n\t"fmt"\n\t"math/rand"\n\t"os"\n\t"time"\n\n\t"github.com/google/uuid"\n\tv1_workflows "github.com/hatchet-dev/hatchet/examples/go/workflows"\n\t"github.com/hatchet-dev/hatchet/pkg/client"\n\t"github.com/hatchet-dev/hatchet/pkg/client/rest"\n\tv1 "github.com/hatchet-dev/hatchet/pkg/v1"\n\t"github.com/joho/godotenv"\n\t"github.com/oapi-codegen/runtime/types"\n)\n\nfunc main() {\n\terr := godotenv.Load()\n\tif err != nil {\n\t\tpanic(err)\n\t}\n\n\thatchet, err := v1.NewHatchetClient()\n\n\tif err != nil {\n\t\tpanic(err)\n\t}\n\n\t// Get workflow name from command line arguments\n\tvar workflowName string\n\tif len(os.Args) > 1 {\n\t\tworkflowName = os.Args[1]\n\t\tfmt.Println("workflow name provided:", workflowName)\n\t} else {\n\t\tfmt.Println("No workflow name provided. Defaulting to \'simple\'")\n\t\tworkflowName = "simple"\n\t}\n\n\tctx := context.Background()\n\n\t// Define workflow runners map\n\trunnerMap := map[string]func() error{\n\t\t"simple": func() error {\n\t\t\tsimple := v1_workflows.Simple(hatchet)\n\t\t\tresult, err := simple.Run(ctx, v1_workflows.SimpleInput{\n\t\t\t\tMessage: "Hello, World!",\n\t\t\t})\n\t\t\tif err != nil {\n\t\t\t\treturn err\n\t\t\t}\n\t\t\tfmt.Println(result.TransformedMessage)\n\t\t\treturn nil\n\t\t},\n\t\t"child": func() error {\n\t\t\tparent := v1_workflows.Parent(hatchet)\n\n\t\t\tresult, err := parent.Run(ctx, v1_workflows.ParentInput{\n\t\t\t\tN: 50,\n\t\t\t})\n\n\t\t\tif err != nil {\n\t\t\t\treturn err\n\t\t\t}\n\t\t\tfmt.Println("Parent result:", result.Result)\n\t\t\treturn nil\n\t\t},\n\t\t"dag": func() error {\n\t\t\tdag := v1_workflows.DagWorkflow(hatchet)\n\t\t\tresult, err := dag.Run(ctx, v1_workflows.DagInput{\n\t\t\t\tMessage: "Hello, DAG!",\n\t\t\t})\n\t\t\tif err != nil {\n\t\t\t\treturn err\n\t\t\t}\n\t\t\tfmt.Println(result.Step1.Step)\n\t\t\tfmt.Println(result.Step2.Step)\n\t\t\treturn nil\n\t\t},\n\t\t"sleep": func() error {\n\t\t\tsleep := v1_workflows.DurableSleep(hatchet)\n\t\t\t_, err := sleep.Run(ctx, v1_workflows.DurableSleepInput{\n\t\t\t\tMessage: "Hello, Sleep!",\n\t\t\t})\n\t\t\tif err != nil {\n\t\t\t\treturn err\n\t\t\t}\n\t\t\tfmt.Println("Sleep workflow completed")\n\t\t\treturn nil\n\t\t},\n\t\t"durable-event": func() error {\n\t\t\tdurableEventWorkflow := v1_workflows.DurableEvent(hatchet)\n\t\t\trun, err := durableEventWorkflow.RunNoWait(ctx, v1_workflows.DurableEventInput{\n\t\t\t\tMessage: "Hello, World!",\n\t\t\t})\n\n\t\t\tif err != nil {\n\t\t\t\treturn err\n\t\t\t}\n\n\t\t\t_, err = hatchet.Runs().Cancel(ctx, rest.V1CancelTaskRequest{\n\t\t\t\tExternalIds: &[]types.UUID{uuid.MustParse(run.WorkflowRunId())},\n\t\t\t})\n\n\t\t\tif err != nil {\n\t\t\t\treturn nil // We expect an error here\n\t\t\t}\n\n\t\t\t_, err = run.Result()\n\n\t\t\tif err != nil {\n\t\t\t\tfmt.Println("Received expected error:", err)\n\t\t\t\treturn nil // We expect an error here\n\t\t\t}\n\t\t\tfmt.Println("Cancellation workflow completed unexpectedly")\n\t\t\treturn nil\n\t\t},\n\t\t"timeout": func() error {\n\t\t\ttimeout := v1_workflows.Timeout(hatchet)\n\t\t\t_, err := timeout.Run(ctx, v1_workflows.TimeoutInput{})\n\t\t\tif err != nil {\n\t\t\t\tfmt.Println("Received expected error:", err)\n\t\t\t\treturn nil // We expect an error here\n\t\t\t}\n\t\t\tfmt.Println("Timeout workflow completed unexpectedly")\n\t\t\treturn nil\n\t\t},\n\t\t"sticky": func() error {\n\t\t\tsticky := v1_workflows.Sticky(hatchet)\n\t\t\tresult, err := sticky.Run(ctx, v1_workflows.StickyInput{})\n\t\t\tif err != nil {\n\t\t\t\treturn err\n\t\t\t}\n\t\t\tfmt.Println("Value from child workflow:", result.Result)\n\t\t\treturn nil\n\t\t},\n\t\t"sticky-dag": func() error {\n\t\t\tstickyDag := v1_workflows.StickyDag(hatchet)\n\t\t\tresult, err := stickyDag.Run(ctx, v1_workflows.StickyInput{})\n\t\t\tif err != nil {\n\t\t\t\treturn err\n\t\t\t}\n\t\t\tfmt.Println("Value from task 1:", result.StickyTask1.Result)\n\t\t\tfmt.Println("Value from task 2:", result.StickyTask2.Result)\n\t\t\treturn nil\n\t\t},\n\t\t"retries": func() error {\n\t\t\tretries := v1_workflows.Retries(hatchet)\n\t\t\t_, err := retries.Run(ctx, v1_workflows.RetriesInput{})\n\t\t\tif err != nil {\n\t\t\t\tfmt.Println("Received expected error:", err)\n\t\t\t\treturn nil // We expect an error here\n\t\t\t}\n\t\t\tfmt.Println("Retries workflow completed unexpectedly")\n\t\t\treturn nil\n\t\t},\n\t\t"retries-count": func() error {\n\t\t\tretriesCount := v1_workflows.RetriesWithCount(hatchet)\n\t\t\tresult, err := retriesCount.Run(ctx, v1_workflows.RetriesWithCountInput{})\n\t\t\tif err != nil {\n\t\t\t\treturn err\n\t\t\t}\n\t\t\tfmt.Println("Result message:", result.Message)\n\t\t\treturn nil\n\t\t},\n\t\t"with-backoff": func() error {\n\t\t\twithBackoff := v1_workflows.WithBackoff(hatchet)\n\t\t\t_, err := withBackoff.Run(ctx, v1_workflows.BackoffInput{})\n\t\t\tif err != nil {\n\t\t\t\tfmt.Println("Received expected error:", err)\n\t\t\t\treturn nil // We expect an error here\n\t\t\t}\n\t\t\tfmt.Println("WithBackoff workflow completed unexpectedly")\n\t\t\treturn nil\n\t\t},\n\t\t"non-retryable": func() error {\n\t\t\tnonRetryable := v1_workflows.NonRetryableError(hatchet)\n\t\t\t_, err := nonRetryable.Run(ctx, v1_workflows.NonRetryableInput{})\n\t\t\tif err != nil {\n\t\t\t\tfmt.Println("Received expected error:", err)\n\t\t\t\treturn nil // We expect an error here\n\t\t\t}\n\t\t\tfmt.Println("NonRetryable workflow completed unexpectedly")\n\t\t\treturn nil\n\t\t},\n\t\t"on-cron": func() error {\n\t\t\tcronTask := v1_workflows.OnCron(hatchet)\n\t\t\tresult, err := cronTask.Run(ctx, v1_workflows.OnCronInput{\n\t\t\t\tMessage: "Hello, Cron!",\n\t\t\t})\n\t\t\tif err != nil {\n\t\t\t\treturn err\n\t\t\t}\n\t\t\tfmt.Println("Cron task result:", result.Job.TransformedMessage)\n\t\t\treturn nil\n\t\t},\n\t\t"priority": func() error {\n\n\t\t\tnRuns := 10\n\t\t\tpriorityWorkflow := v1_workflows.Priority(hatchet)\n\n\t\t\tfor i := 0; i < nRuns; i++ {\n\t\t\t\trandomPrio := int32(rand.Intn(3) + 1)\n\n\t\t\t\tfmt.Println("Random priority:", randomPrio)\n\n\t\t\t\tpriorityWorkflow.RunNoWait(ctx, v1_workflows.PriorityInput{\n\t\t\t\t\tUserId: "1234",\n\t\t\t\t}, client.WithRunMetadata(map[string]int32{"priority": randomPrio}), client.WithPriority(randomPrio))\n\t\t\t}\n\n\t\t\ttriggerAt := time.Now().Add(time.Second + 5)\n\n\t\t\tfor i := 0; i < nRuns; i++ {\n\t\t\t\trandomPrio := int32(rand.Intn(3) + 1)\n\n\t\t\t\tfmt.Println("Random priority:", randomPrio)\n\n\t\t\t\tpriorityWorkflow.Schedule(ctx, triggerAt, v1_workflows.PriorityInput{\n\t\t\t\t\tUserId: "1234",\n\t\t\t\t}, client.WithRunMetadata(map[string]int32{"priority": randomPrio}), client.WithPriority(randomPrio))\n\t\t\t}\n\n\t\t\treturn nil\n\t\t},\n\t}\n\n\t// Lookup workflow runner from map\n\trunner, ok := runnerMap[workflowName]\n\tif !ok {\n\t\tfmt.Println("Invalid workflow name provided. Usage: go run examples/v1/run/simple.go [workflow-name]")\n\t\tfmt.Println("Available workflows:", getAvailableWorkflows(runnerMap))\n\t\tos.Exit(1)\n\t}\n\n\t// Run the selected workflow\n\terr = runner()\n\tif err != nil {\n\t\tpanic(err)\n\t}\n}\n\n// Helper function to get available workflows as a formatted string\nfunc getAvailableWorkflows(runnerMap map[string]func() error) string {\n\tvar workflows string\n\tcount := 0\n\tfor name := range runnerMap {\n\t\tif count > 0 {\n\t\t\tworkflows += ", "\n\t\t}\n\t\tworkflows += fmt.Sprintf("\'%s\'", name)\n\t\tcount++\n\t}\n\treturn workflows\n}\n',
      language: 'go',
      source: 'examples/go/run/all.go',
      highlights: {},
    },
  L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL2dvL3J1bi9idWxrLmdv:
    {
      content:
        'package main\n\nimport (\n\t"context"\n\t"fmt"\n\n\tv1_workflows "github.com/hatchet-dev/hatchet/examples/go/workflows"\n\tv1 "github.com/hatchet-dev/hatchet/pkg/v1"\n\t"github.com/joho/godotenv"\n)\n\nfunc bulk() {\n\terr := godotenv.Load()\n\tif err != nil {\n\t\tpanic(err)\n\t}\n\n\thatchet, err := v1.NewHatchetClient()\n\n\tif err != nil {\n\t\tpanic(err)\n\t}\n\n\tctx := context.Background()\n\t// ❓ Bulk Run Tasks\n\tsimple := v1_workflows.Simple(hatchet)\n\tbulkRunIds, err := simple.RunBulkNoWait(ctx, []v1_workflows.SimpleInput{\n\t\t{\n\t\t\tMessage: "Hello, World!",\n\t\t},\n\t\t{\n\t\t\tMessage: "Hello, Moon!",\n\t\t},\n\t})\n\n\tif err != nil {\n\t\tpanic(err)\n\t}\n\n\tfmt.Println(bulkRunIds)\n\t// !!\n}\n',
      language: 'go',
      source: 'examples/go/run/bulk.go',
      highlights: {},
    },
  L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL2dvL3J1bi9jcm9uLmdv:
    {
      content:
        'package main\n\nimport (\n\t"context"\n\t"fmt"\n\n\tv1_workflows "github.com/hatchet-dev/hatchet/examples/go/workflows"\n\t"github.com/hatchet-dev/hatchet/pkg/client/rest"\n\tv1 "github.com/hatchet-dev/hatchet/pkg/v1"\n\t"github.com/joho/godotenv"\n)\n\nfunc cron() {\n\terr := godotenv.Load()\n\tif err != nil {\n\t\tpanic(err)\n\t}\n\n\thatchet, err := v1.NewHatchetClient()\n\n\tif err != nil {\n\t\tpanic(err)\n\t}\n\t// ❓ Create\n\tsimple := v1_workflows.Simple(hatchet)\n\n\tctx := context.Background()\n\n\tresult, err := simple.Cron(\n\t\tctx,\n\t\t"daily-run",\n\t\t"0 0 * * *",\n\t\tv1_workflows.SimpleInput{\n\t\t\tMessage: "Hello, World!",\n\t\t},\n\t)\n\n\tif err != nil {\n\t\tpanic(err)\n\t}\n\n\t// it may be useful to save the cron id for later\n\tfmt.Println(result.Metadata.Id)\n\t// !!\n\n\t// ❓ Delete\n\thatchet.Crons().Delete(ctx, result.Metadata.Id)\n\t// !!\n\n\t// ❓ List\n\tcrons, err := hatchet.Crons().List(ctx, rest.CronWorkflowListParams{\n\t\tAdditionalMetadata: &[]string{"user:daily-run"},\n\t})\n\t// !!\n\tif err != nil {\n\t\tpanic(err)\n\t}\n\tfmt.Println(crons)\n}\n',
      language: 'go',
      source: 'examples/go/run/cron.go',
      highlights: {},
    },
  L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL2dvL3J1bi9ldmVudC5nbw__:
    {
      content:
        'package main\n\nimport (\n\t"context"\n\n\tv1_workflows "github.com/hatchet-dev/hatchet/examples/go/workflows"\n\tv1 "github.com/hatchet-dev/hatchet/pkg/v1"\n\t"github.com/joho/godotenv"\n)\n\nfunc event() {\n\terr := godotenv.Load()\n\tif err != nil {\n\t\tpanic(err)\n\t}\n\n\thatchet, err := v1.NewHatchetClient()\n\n\tif err != nil {\n\t\tpanic(err)\n\t}\n\t// ❓ Pushing an Event\n\terr = hatchet.Events().Push(\n\t\tcontext.Background(),\n\t\t"simple-event:create",\n\t\tv1_workflows.SimpleInput{\n\t\t\tMessage: "Hello, World!",\n\t\t},\n\t)\n\t// !!\n\n\tif err != nil {\n\t\tpanic(err)\n\t}\n}\n',
      language: 'go',
      source: 'examples/go/run/event.go',
      highlights: {},
    },
  L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL2dvL3J1bi9wcmlvcml0eS5nbw__:
    {
      content:
        'package main\n\nimport (\n\t"context"\n\t"fmt"\n\t"time"\n\n\tv1_workflows "github.com/hatchet-dev/hatchet/examples/go/workflows"\n\t"github.com/hatchet-dev/hatchet/pkg/client"\n\tv1 "github.com/hatchet-dev/hatchet/pkg/v1"\n\t"github.com/joho/godotenv"\n)\n\nfunc priority() {\n\terr := godotenv.Load()\n\tif err != nil {\n\t\tpanic(err)\n\t}\n\n\thatchet, err := v1.NewHatchetClient()\n\n\tif err != nil {\n\t\tpanic(err)\n\t}\n\n\tctx := context.Background()\n\n\tpriorityWorkflow := v1_workflows.Priority(hatchet)\n\n\t// ❓ Running a Task with Priority\n\tpriority := int32(3)\n\n\trunId, err := priorityWorkflow.RunNoWait(ctx, v1_workflows.PriorityInput{\n\t\tUserId: "1234",\n\t}, client.WithPriority(priority))\n\t// !!\n\n\tif err != nil {\n\t\tpanic(err)\n\t}\n\n\tfmt.Println(runId)\n\n\t// ❓ Schedule and cron\n\tschedulePriority := int32(3)\n\trunAt := time.Now().Add(time.Minute)\n\n\tscheduledRunId, _ := priorityWorkflow.Schedule(ctx, runAt, v1_workflows.PriorityInput{\n\t\tUserId: "1234",\n\t}, client.WithPriority(schedulePriority))\n\n\tcronId, _ := priorityWorkflow.Cron(ctx, "my-cron", "* * * * *", v1_workflows.PriorityInput{\n\t\tUserId: "1234",\n\t}, client.WithPriority(schedulePriority))\n\t// !!\n\n\tfmt.Println(scheduledRunId)\n\tfmt.Println(cronId)\n\n\t// !!\n}\n',
      language: 'go',
      source: 'examples/go/run/priority.go',
      highlights: {},
    },
  L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL2dvL3J1bi9zaW1wbGUuZ28_:
    {
      content:
        'package main\n\nimport (\n\t"context"\n\t"fmt"\n\t"sync"\n\n\tv1_workflows "github.com/hatchet-dev/hatchet/examples/go/workflows"\n\tv1 "github.com/hatchet-dev/hatchet/pkg/v1"\n\t"github.com/joho/godotenv"\n)\n\nfunc simple() {\n\terr := godotenv.Load()\n\tif err != nil {\n\t\tpanic(err)\n\t}\n\n\thatchet, err := v1.NewHatchetClient()\n\n\tif err != nil {\n\t\tpanic(err)\n\t}\n\n\tctx := context.Background()\n\t// ❓ Running a Task\n\tsimple := v1_workflows.Simple(hatchet)\n\tresult, err := simple.Run(ctx, v1_workflows.SimpleInput{\n\t\tMessage: "Hello, World!",\n\t})\n\n\tif err != nil {\n\t\tpanic(err)\n\t}\n\n\tfmt.Println(result.TransformedMessage)\n\t// !!\n\n\t// ❓ Running Multiple Tasks\n\tvar results []string\n\tvar resultsMutex sync.Mutex\n\tvar errs []error\n\tvar errsMutex sync.Mutex\n\n\twg := sync.WaitGroup{}\n\twg.Add(2)\n\n\tgo func() {\n\t\tdefer wg.Done()\n\t\tresult, err := simple.Run(ctx, v1_workflows.SimpleInput{\n\t\t\tMessage: "Hello, World!",\n\t\t})\n\n\t\tif err != nil {\n\t\t\terrsMutex.Lock()\n\t\t\terrs = append(errs, err)\n\t\t\terrsMutex.Unlock()\n\t\t\treturn\n\t\t}\n\n\t\tresultsMutex.Lock()\n\t\tresults = append(results, result.TransformedMessage)\n\t\tresultsMutex.Unlock()\n\t}()\n\n\tgo func() {\n\t\tdefer wg.Done()\n\t\tresult, err := simple.Run(ctx, v1_workflows.SimpleInput{\n\t\t\tMessage: "Hello, Moon!",\n\t\t})\n\n\t\tif err != nil {\n\t\t\terrsMutex.Lock()\n\t\t\terrs = append(errs, err)\n\t\t\terrsMutex.Unlock()\n\t\t\treturn\n\t\t}\n\n\t\tresultsMutex.Lock()\n\t\tresults = append(results, result.TransformedMessage)\n\t\tresultsMutex.Unlock()\n\t}()\n\n\twg.Wait()\n\t// !!\n\n\t// ❓ Running a Task Without Waiting\n\tsimple = v1_workflows.Simple(hatchet)\n\trunRef, err := simple.RunNoWait(ctx, v1_workflows.SimpleInput{\n\t\tMessage: "Hello, World!",\n\t})\n\n\tif err != nil {\n\t\tpanic(err)\n\t}\n\n\t// The Run Ref Exposes an ID that can be used to wait for the task to complete\n\t// or check on the status of the task\n\trunId := runRef.RunId()\n\tfmt.Println(runId)\n\t// !!\n\n\t// ❓ Subscribing to results\n\t// finally, we can wait for the task to complete and get the result\n\tfinalResult, err := runRef.Result()\n\n\tif err != nil {\n\t\tpanic(err)\n\t}\n\n\tfmt.Println(finalResult)\n\t// !!\n}\n',
      language: 'go',
      source: 'examples/go/run/simple.go',
      highlights: {},
    },
  L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL2dvL3dvcmtlci9zdGFydC5nbw__:
    {
      content:
        'package main\n\nimport (\n\t"fmt"\n\t"os"\n\t"time"\n\n\tv1_workflows "github.com/hatchet-dev/hatchet/examples/go/workflows"\n\t"github.com/hatchet-dev/hatchet/pkg/cmdutils"\n\tv1 "github.com/hatchet-dev/hatchet/pkg/v1"\n\t"github.com/hatchet-dev/hatchet/pkg/v1/worker"\n\t"github.com/hatchet-dev/hatchet/pkg/v1/workflow"\n\t"github.com/joho/godotenv"\n)\n\nfunc main() {\n\terr := godotenv.Load()\n\tif err != nil {\n\t\tpanic(err)\n\t}\n\n\thatchet, err := v1.NewHatchetClient()\n\n\tif err != nil {\n\t\tpanic(err)\n\t}\n\n\t// Get workflow name from command line arguments\n\tvar workflowName string\n\tif len(os.Args) > 1 {\n\t\tworkflowName = os.Args[1]\n\t\tfmt.Println("workflow name provided:", workflowName)\n\t}\n\n\t// Define workflows map\n\tworkflowMap := map[string][]workflow.WorkflowBase{\n\t\t"dag":           {v1_workflows.DagWorkflow(hatchet)},\n\t\t"on-failure":    {v1_workflows.OnFailure(hatchet)},\n\t\t"simple":        {v1_workflows.Simple(hatchet)},\n\t\t"sleep":         {v1_workflows.DurableSleep(hatchet)},\n\t\t"child":         {v1_workflows.Parent(hatchet), v1_workflows.Child(hatchet)},\n\t\t"cancellation":  {v1_workflows.Cancellation(hatchet)},\n\t\t"timeout":       {v1_workflows.Timeout(hatchet)},\n\t\t"sticky":        {v1_workflows.Sticky(hatchet), v1_workflows.StickyDag(hatchet), v1_workflows.Child(hatchet)},\n\t\t"retries":       {v1_workflows.Retries(hatchet), v1_workflows.RetriesWithCount(hatchet), v1_workflows.WithBackoff(hatchet)},\n\t\t"on-cron":       {v1_workflows.OnCron(hatchet)},\n\t\t"non-retryable": {v1_workflows.NonRetryableError(hatchet)},\n\t\t"priority":      {v1_workflows.Priority(hatchet)},\n\t}\n\n\t// Add an "all" option that registers all workflows\n\tallWorkflows := []workflow.WorkflowBase{}\n\tfor _, wfs := range workflowMap {\n\t\tallWorkflows = append(allWorkflows, wfs...)\n\t}\n\tworkflowMap["all"] = allWorkflows\n\n\t// Lookup workflow from map\n\tworkflow, ok := workflowMap[workflowName]\n\tif !ok {\n\t\tfmt.Println("Invalid workflow name provided. Usage: go run examples/v1/worker/start.go [workflow-name]")\n\t\tfmt.Println("Available workflows:", getAvailableWorkflows(workflowMap))\n\t\tos.Exit(1)\n\t}\n\n\tvar slots int\n\tif workflowName == "priority" {\n\t\tslots = 1\n\t} else {\n\t\tslots = 100\n\t}\n\n\tworker, err := hatchet.Worker(\n\t\tworker.WorkerOpts{\n\t\t\tName:      fmt.Sprintf("%s-worker", workflowName),\n\t\t\tWorkflows: workflow,\n\t\t\tSlots:     slots,\n\t\t},\n\t)\n\n\tif err != nil {\n\t\tpanic(err)\n\t}\n\n\tinterruptCtx, cancel := cmdutils.NewInterruptContext()\n\n\terr = worker.StartBlocking(interruptCtx)\n\n\tif err != nil {\n\t\tpanic(err)\n\t}\n\n\tgo func() {\n\t\ttime.Sleep(10 * time.Second)\n\t\tcancel()\n\t}()\n}\n\n// Helper function to get available workflows as a formatted string\nfunc getAvailableWorkflows(workflowMap map[string][]workflow.WorkflowBase) string {\n\tvar workflows string\n\tcount := 0\n\tfor name := range workflowMap {\n\t\tif count > 0 {\n\t\t\tworkflows += ", "\n\t\t}\n\t\tworkflows += fmt.Sprintf("\'%s\'", name)\n\t\tcount++\n\t}\n\treturn workflows\n}\n',
      language: 'go',
      source: 'examples/go/worker/start.go',
      highlights: {},
    },
  L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL2dvL3dvcmtmbG93cy9jYW5jZWxsYXRpb25zLmdv:
    {
      content:
        'package v1_workflows\n\nimport (\n\t"errors"\n\t"time"\n\n\t"github.com/hatchet-dev/hatchet/pkg/client/create"\n\tv1 "github.com/hatchet-dev/hatchet/pkg/v1"\n\t"github.com/hatchet-dev/hatchet/pkg/v1/factory"\n\t"github.com/hatchet-dev/hatchet/pkg/v1/workflow"\n\t"github.com/hatchet-dev/hatchet/pkg/worker"\n)\n\ntype CancellationInput struct{}\ntype CancellationResult struct {\n\tCompleted bool\n}\n\nfunc Cancellation(hatchet v1.HatchetClient) workflow.WorkflowDeclaration[CancellationInput, CancellationResult] {\n\n\t// ❓ Cancelled task\n\t// Create a task that sleeps for 10 seconds and checks if it was cancelled\n\tcancellation := factory.NewTask(\n\t\tcreate.StandaloneTask{\n\t\t\tName: "cancellation-task",\n\t\t}, func(ctx worker.HatchetContext, input CancellationInput) (*CancellationResult, error) {\n\t\t\t// Sleep for 10 seconds\n\t\t\ttime.Sleep(10 * time.Second)\n\n\t\t\t// Check if the context was cancelled\n\t\t\tselect {\n\t\t\tcase <-ctx.Done():\n\t\t\t\treturn nil, errors.New("Task was cancelled")\n\t\t\tdefault:\n\t\t\t\t// Continue execution\n\t\t\t}\n\n\t\t\treturn &CancellationResult{\n\t\t\t\tCompleted: true,\n\t\t\t}, nil\n\t\t},\n\t\thatchet,\n\t)\n\t// !!\n\n\treturn cancellation\n}\n',
      language: 'go',
      source: 'examples/go/workflows/cancellations.go',
      highlights: {},
    },
  L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL2dvL3dvcmtmbG93cy9jaGlsZC13b3JrZmxvd3MuZ28_:
    {
      content:
        'package v1_workflows\n\nimport (\n\t"github.com/hatchet-dev/hatchet/pkg/client/create"\n\tv1 "github.com/hatchet-dev/hatchet/pkg/v1"\n\t"github.com/hatchet-dev/hatchet/pkg/v1/factory"\n\t"github.com/hatchet-dev/hatchet/pkg/v1/workflow"\n\t"github.com/hatchet-dev/hatchet/pkg/worker"\n)\n\ntype ChildInput struct {\n\tN int `json:"n"`\n}\n\ntype ValueOutput struct {\n\tValue int `json:"value"`\n}\n\ntype ParentInput struct {\n\tN int `json:"n"`\n}\n\ntype SumOutput struct {\n\tResult int `json:"result"`\n}\n\nfunc Child(hatchet v1.HatchetClient) workflow.WorkflowDeclaration[ChildInput, ValueOutput] {\n\tchild := factory.NewTask(\n\t\tcreate.StandaloneTask{\n\t\t\tName: "child",\n\t\t}, func(ctx worker.HatchetContext, input ChildInput) (*ValueOutput, error) {\n\t\t\treturn &ValueOutput{\n\t\t\t\tValue: input.N,\n\t\t\t}, nil\n\t\t},\n\t\thatchet,\n\t)\n\n\treturn child\n}\n\nfunc Parent(hatchet v1.HatchetClient) workflow.WorkflowDeclaration[ParentInput, SumOutput] {\n\n\tchild := Child(hatchet)\n\tparent := factory.NewTask(\n\t\tcreate.StandaloneTask{\n\t\t\tName: "parent",\n\t\t}, func(ctx worker.HatchetContext, input ParentInput) (*SumOutput, error) {\n\n\t\t\tsum := 0\n\n\t\t\t// Launch child workflows in parallel\n\t\t\tresults := make([]*ValueOutput, 0, input.N)\n\t\t\tfor j := 0; j < input.N; j++ {\n\t\t\t\tresult, err := child.RunAsChild(ctx, ChildInput{N: j}, workflow.RunAsChildOpts{})\n\n\t\t\t\tif err != nil {\n\t\t\t\t\t// firstErr = err\n\t\t\t\t\treturn nil, err\n\t\t\t\t}\n\n\t\t\t\tresults = append(results, result)\n\n\t\t\t}\n\n\t\t\t// Sum results from all children\n\t\t\tfor _, result := range results {\n\t\t\t\tsum += result.Value\n\t\t\t}\n\n\t\t\treturn &SumOutput{\n\t\t\t\tResult: sum,\n\t\t\t}, nil\n\t\t},\n\t\thatchet,\n\t)\n\n\treturn parent\n}\n',
      language: 'go',
      source: 'examples/go/workflows/child-workflows.go',
      highlights: {},
    },
  L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL2dvL3dvcmtmbG93cy9jb21wbGV4LWNvbmRpdGlvbnMuZ28_:
    {
      content:
        'package v1_workflows\n\nimport (\n\t"math/rand"\n\t"time"\n\n\t"github.com/hatchet-dev/hatchet/pkg/client/create"\n\tv1 "github.com/hatchet-dev/hatchet/pkg/v1"\n\t"github.com/hatchet-dev/hatchet/pkg/v1/factory"\n\t"github.com/hatchet-dev/hatchet/pkg/v1/workflow"\n\t"github.com/hatchet-dev/hatchet/pkg/worker"\n\t"github.com/hatchet-dev/hatchet/pkg/worker/condition"\n)\n\n// StepOutput represents the output of most tasks in this workflow\ntype StepOutput struct {\n\tRandomNumber int `json:"randomNumber"`\n}\n\n// RandomSum represents the output of the sum task\ntype RandomSum struct {\n\tSum int `json:"sum"`\n}\n\n// TaskConditionWorkflowResult represents the aggregate output of all tasks\ntype TaskConditionWorkflowResult struct {\n\tStart        StepOutput `json:"start"`\n\tWaitForSleep StepOutput `json:"waitForSleep"`\n\tWaitForEvent StepOutput `json:"waitForEvent"`\n\tSkipOnEvent  StepOutput `json:"skipOnEvent"`\n\tLeftBranch   StepOutput `json:"leftBranch"`\n\tRightBranch  StepOutput `json:"rightBranch"`\n\tSum          RandomSum  `json:"sum"`\n}\n\n// taskOpts is a type alias for workflow task options\ntype taskOpts = create.WorkflowTask[struct{}, TaskConditionWorkflowResult]\n\nfunc TaskConditionWorkflow(hatchet v1.HatchetClient) workflow.WorkflowDeclaration[struct{}, TaskConditionWorkflowResult] {\n\t// ❓ Create a workflow\n\twf := factory.NewWorkflow[struct{}, TaskConditionWorkflowResult](\n\t\tcreate.WorkflowCreateOpts[struct{}]{\n\t\t\tName: "TaskConditionWorkflow",\n\t\t},\n\t\thatchet,\n\t)\n\t// !!\n\n\t// ❓ Add base task\n\tstart := wf.Task(\n\t\ttaskOpts{\n\t\t\tName: "start",\n\t\t},\n\t\tfunc(ctx worker.HatchetContext, _ struct{}) (interface{}, error) {\n\t\t\treturn &StepOutput{\n\t\t\t\tRandomNumber: rand.Intn(100) + 1,\n\t\t\t}, nil\n\t\t},\n\t)\n\t// !!\n\n\t// ❓ Add wait for sleep\n\twaitForSleep := wf.Task(\n\t\ttaskOpts{\n\t\t\tName:    "waitForSleep",\n\t\t\tParents: []create.NamedTask{start},\n\t\t\tWaitFor: condition.SleepCondition(time.Second * 10),\n\t\t},\n\t\tfunc(ctx worker.HatchetContext, _ struct{}) (interface{}, error) {\n\t\t\treturn &StepOutput{\n\t\t\t\tRandomNumber: rand.Intn(100) + 1,\n\t\t\t}, nil\n\t\t},\n\t)\n\t// !!\n\n\t// ❓ Add skip on event\n\tskipOnEvent := wf.Task(\n\t\ttaskOpts{\n\t\t\tName:    "skipOnEvent",\n\t\t\tParents: []create.NamedTask{start},\n\t\t\tWaitFor: condition.SleepCondition(time.Second * 30),\n\t\t\tSkipIf:  condition.UserEventCondition("skip_on_event:skip", "true"),\n\t\t},\n\t\tfunc(ctx worker.HatchetContext, _ struct{}) (interface{}, error) {\n\t\t\treturn &StepOutput{\n\t\t\t\tRandomNumber: rand.Intn(100) + 1,\n\t\t\t}, nil\n\t\t},\n\t)\n\t// !!\n\n\t// ❓ Add branching\n\tleftBranch := wf.Task(\n\t\ttaskOpts{\n\t\t\tName:    "leftBranch",\n\t\t\tParents: []create.NamedTask{waitForSleep},\n\t\t\tSkipIf:  condition.ParentCondition(waitForSleep, "output.randomNumber > 50"),\n\t\t},\n\t\tfunc(ctx worker.HatchetContext, _ struct{}) (interface{}, error) {\n\t\t\treturn &StepOutput{\n\t\t\t\tRandomNumber: rand.Intn(100) + 1,\n\t\t\t}, nil\n\t\t},\n\t)\n\n\trightBranch := wf.Task(\n\t\ttaskOpts{\n\t\t\tName:    "rightBranch",\n\t\t\tParents: []create.NamedTask{waitForSleep},\n\t\t\tSkipIf:  condition.ParentCondition(waitForSleep, "output.randomNumber <= 50"),\n\t\t},\n\t\tfunc(ctx worker.HatchetContext, _ struct{}) (interface{}, error) {\n\t\t\treturn &StepOutput{\n\t\t\t\tRandomNumber: rand.Intn(100) + 1,\n\t\t\t}, nil\n\t\t},\n\t)\n\t// !!\n\n\t// ❓ Add wait for event\n\twaitForEvent := wf.Task(\n\t\ttaskOpts{\n\t\t\tName:    "waitForEvent",\n\t\t\tParents: []create.NamedTask{start},\n\t\t\tWaitFor: condition.Or(\n\t\t\t\tcondition.SleepCondition(time.Minute),\n\t\t\t\tcondition.UserEventCondition("wait_for_event:start", "true"),\n\t\t\t),\n\t\t},\n\t\tfunc(ctx worker.HatchetContext, _ struct{}) (interface{}, error) {\n\t\t\treturn &StepOutput{\n\t\t\t\tRandomNumber: rand.Intn(100) + 1,\n\t\t\t}, nil\n\t\t},\n\t)\n\t// !!\n\n\t// ❓ Add sum\n\twf.Task(\n\t\ttaskOpts{\n\t\t\tName: "sum",\n\t\t\tParents: []create.NamedTask{\n\t\t\t\tstart,\n\t\t\t\twaitForSleep,\n\t\t\t\twaitForEvent,\n\t\t\t\tskipOnEvent,\n\t\t\t\tleftBranch,\n\t\t\t\trightBranch,\n\t\t\t},\n\t\t},\n\t\tfunc(ctx worker.HatchetContext, _ struct{}) (interface{}, error) {\n\t\t\tvar startOutput StepOutput\n\t\t\tif err := ctx.ParentOutput(start, &startOutput); err != nil {\n\t\t\t\treturn nil, err\n\t\t\t}\n\n\t\t\tvar waitForSleepOutput StepOutput\n\t\t\tif err := ctx.ParentOutput(waitForSleep, &waitForSleepOutput); err != nil {\n\t\t\t\treturn nil, err\n\t\t\t}\n\n\t\t\tvar waitForEventOutput StepOutput\n\t\t\tctx.ParentOutput(waitForEvent, &waitForEventOutput)\n\n\t\t\t// Handle potentially skipped tasks\n\t\t\tvar skipOnEventOutput StepOutput\n\t\t\tvar four int\n\n\t\t\terr := ctx.ParentOutput(skipOnEvent, &skipOnEventOutput)\n\n\t\t\tif err != nil {\n\t\t\t\tfour = 0\n\t\t\t} else {\n\t\t\t\tfour = skipOnEventOutput.RandomNumber\n\t\t\t}\n\n\t\t\tvar leftBranchOutput StepOutput\n\t\t\tvar five int\n\n\t\t\terr = ctx.ParentOutput(leftBranch, leftBranchOutput)\n\t\t\tif err != nil {\n\t\t\t\tfive = 0\n\t\t\t} else {\n\t\t\t\tfive = leftBranchOutput.RandomNumber\n\t\t\t}\n\n\t\t\tvar rightBranchOutput StepOutput\n\t\t\tvar six int\n\n\t\t\terr = ctx.ParentOutput(rightBranch, rightBranchOutput)\n\t\t\tif err != nil {\n\t\t\t\tsix = 0\n\t\t\t} else {\n\t\t\t\tsix = rightBranchOutput.RandomNumber\n\t\t\t}\n\n\t\t\treturn &RandomSum{\n\t\t\t\tSum: startOutput.RandomNumber + waitForEventOutput.RandomNumber +\n\t\t\t\t\twaitForSleepOutput.RandomNumber + four + five + six,\n\t\t\t}, nil\n\t\t},\n\t)\n\t// !!\n\n\treturn wf\n}\n',
      language: 'go',
      source: 'examples/go/workflows/complex-conditions.go',
      highlights: {},
    },
  L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL2dvL3dvcmtmbG93cy9jb25jdXJyZW5jeS1yci5nbw__:
    {
      content:
        'package v1_workflows\n\nimport (\n\t"math/rand"\n\t"time"\n\n\t"github.com/hatchet-dev/hatchet/pkg/client/create"\n\t"github.com/hatchet-dev/hatchet/pkg/client/types"\n\tv1 "github.com/hatchet-dev/hatchet/pkg/v1"\n\t"github.com/hatchet-dev/hatchet/pkg/v1/factory"\n\t"github.com/hatchet-dev/hatchet/pkg/v1/workflow"\n\t"github.com/hatchet-dev/hatchet/pkg/worker"\n)\n\ntype ConcurrencyInput struct {\n\tMessage string\n\tTier    string\n\tAccount string\n}\n\ntype TransformedOutput struct {\n\tTransformedMessage string\n}\n\nfunc ConcurrencyRoundRobin(hatchet v1.HatchetClient) workflow.WorkflowDeclaration[ConcurrencyInput, TransformedOutput] {\n\t// ❓ Concurrency Strategy With Key\n\tvar maxRuns int32 = 1\n\tstrategy := types.GroupRoundRobin\n\n\tconcurrency := factory.NewTask(\n\t\tcreate.StandaloneTask{\n\t\t\tName: "simple-concurrency",\n\t\t\tConcurrency: []*types.Concurrency{\n\t\t\t\t{\n\t\t\t\t\tExpression:    "input.GroupKey",\n\t\t\t\t\tMaxRuns:       &maxRuns,\n\t\t\t\t\tLimitStrategy: &strategy,\n\t\t\t\t},\n\t\t\t},\n\t\t}, func(ctx worker.HatchetContext, input ConcurrencyInput) (*TransformedOutput, error) {\n\t\t\t// Random sleep between 200ms and 1000ms\n\t\t\ttime.Sleep(time.Duration(200+rand.Intn(800)) * time.Millisecond)\n\n\t\t\treturn &TransformedOutput{\n\t\t\t\tTransformedMessage: input.Message,\n\t\t\t}, nil\n\t\t},\n\t\thatchet,\n\t)\n\t// !!\n\n\treturn concurrency\n}\n\nfunc MultipleConcurrencyKeys(hatchet v1.HatchetClient) workflow.WorkflowDeclaration[ConcurrencyInput, TransformedOutput] {\n\t// ❓ Multiple Concurrency Keys\n\tstrategy := types.GroupRoundRobin\n\tvar maxRuns int32 = 20\n\n\tconcurrency := factory.NewTask(\n\t\tcreate.StandaloneTask{\n\t\t\tName: "simple-concurrency",\n\t\t\tConcurrency: []*types.Concurrency{\n\t\t\t\t{\n\t\t\t\t\tExpression:    "input.Tier",\n\t\t\t\t\tMaxRuns:       &maxRuns,\n\t\t\t\t\tLimitStrategy: &strategy,\n\t\t\t\t},\n\t\t\t\t{\n\t\t\t\t\tExpression:    "input.Account",\n\t\t\t\t\tMaxRuns:       &maxRuns,\n\t\t\t\t\tLimitStrategy: &strategy,\n\t\t\t\t},\n\t\t\t},\n\t\t}, func(ctx worker.HatchetContext, input ConcurrencyInput) (*TransformedOutput, error) {\n\t\t\t// Random sleep between 200ms and 1000ms\n\t\t\ttime.Sleep(time.Duration(200+rand.Intn(800)) * time.Millisecond)\n\n\t\t\treturn &TransformedOutput{\n\t\t\t\tTransformedMessage: input.Message,\n\t\t\t}, nil\n\t\t},\n\t\thatchet,\n\t)\n\t// !!\n\n\treturn concurrency\n}\n',
      language: 'go',
      source: 'examples/go/workflows/concurrency-rr.go',
      highlights: {},
    },
  L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL2dvL3dvcmtmbG93cy9kYWctd2l0aC1jb25kaXRpb25zLmdv:
    {
      content:
        'package v1_workflows\n\nimport (\n\t"fmt"\n\n\t"github.com/hatchet-dev/hatchet/pkg/client/create"\n\tv1 "github.com/hatchet-dev/hatchet/pkg/v1"\n\t"github.com/hatchet-dev/hatchet/pkg/v1/factory"\n\t"github.com/hatchet-dev/hatchet/pkg/v1/workflow"\n\t"github.com/hatchet-dev/hatchet/pkg/worker"\n)\n\ntype DagWithConditionsInput struct {\n\tMessage string\n}\n\ntype DagWithConditionsResult struct {\n\tStep1 SimpleOutput\n\tStep2 SimpleOutput\n}\n\ntype conditionOpts = create.WorkflowTask[DagWithConditionsInput, DagWithConditionsResult]\n\nfunc DagWithConditionsWorkflow(hatchet v1.HatchetClient) workflow.WorkflowDeclaration[DagWithConditionsInput, DagWithConditionsResult] {\n\n\tsimple := factory.NewWorkflow[DagWithConditionsInput, DagWithConditionsResult](\n\t\tcreate.WorkflowCreateOpts[DagWithConditionsInput]{\n\t\t\tName: "simple-dag",\n\t\t},\n\t\thatchet,\n\t)\n\n\tstep1 := simple.Task(\n\t\tconditionOpts{\n\t\t\tName: "Step1",\n\t\t}, func(ctx worker.HatchetContext, input DagWithConditionsInput) (interface{}, error) {\n\t\t\treturn &SimpleOutput{\n\t\t\t\tStep: 1,\n\t\t\t}, nil\n\t\t},\n\t)\n\n\tsimple.Task(\n\t\tconditionOpts{\n\t\t\tName: "Step2",\n\t\t\tParents: []create.NamedTask{\n\t\t\t\tstep1,\n\t\t\t},\n\t\t}, func(ctx worker.HatchetContext, input DagWithConditionsInput) (interface{}, error) {\n\n\t\t\tvar step1Output SimpleOutput\n\t\t\terr := ctx.ParentOutput(step1, &step1Output)\n\t\t\tif err != nil {\n\t\t\t\treturn nil, err\n\t\t\t}\n\n\t\t\tfmt.Println(step1Output.Step)\n\n\t\t\treturn &SimpleOutput{\n\t\t\t\tStep: 2,\n\t\t\t}, nil\n\t\t},\n\t)\n\n\treturn simple\n}\n',
      language: 'go',
      source: 'examples/go/workflows/dag-with-conditions.go',
      highlights: {},
    },
  L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL2dvL3dvcmtmbG93cy9kYWcuZ28_:
    {
      content:
        'package v1_workflows\n\nimport (\n\t"github.com/hatchet-dev/hatchet/pkg/client/create"\n\tv1 "github.com/hatchet-dev/hatchet/pkg/v1"\n\t"github.com/hatchet-dev/hatchet/pkg/v1/factory"\n\t"github.com/hatchet-dev/hatchet/pkg/v1/workflow"\n\t"github.com/hatchet-dev/hatchet/pkg/worker"\n)\n\ntype DagInput struct {\n\tMessage string\n}\n\ntype SimpleOutput struct {\n\tStep int\n}\n\ntype DagResult struct {\n\tStep1 SimpleOutput\n\tStep2 SimpleOutput\n}\n\nfunc DagWorkflow(hatchet v1.HatchetClient) workflow.WorkflowDeclaration[DagInput, DagResult] {\n\t// ❓ Declaring a Workflow\n\tsimple := factory.NewWorkflow[DagInput, DagResult](\n\t\tcreate.WorkflowCreateOpts[DagInput]{\n\t\t\tName: "simple-dag",\n\t\t},\n\t\thatchet,\n\t)\n\t// ‼️\n\n\t// ❓ Defining a Task\n\tsimple.Task(\n\t\tcreate.WorkflowTask[DagInput, DagResult]{\n\t\t\tName: "step",\n\t\t}, func(ctx worker.HatchetContext, input DagInput) (interface{}, error) {\n\t\t\treturn &SimpleOutput{\n\t\t\t\tStep: 1,\n\t\t\t}, nil\n\t\t},\n\t)\n\t// ‼️\n\n\t// ❓ Adding a Task with a parent\n\tstep1 := simple.Task(\n\t\tcreate.WorkflowTask[DagInput, DagResult]{\n\t\t\tName: "step-1",\n\t\t}, func(ctx worker.HatchetContext, input DagInput) (interface{}, error) {\n\t\t\treturn &SimpleOutput{\n\t\t\t\tStep: 1,\n\t\t\t}, nil\n\t\t},\n\t)\n\n\tsimple.Task(\n\t\tcreate.WorkflowTask[DagInput, DagResult]{\n\t\t\tName: "step-2",\n\t\t\tParents: []create.NamedTask{\n\t\t\t\tstep1,\n\t\t\t},\n\t\t}, func(ctx worker.HatchetContext, input DagInput) (interface{}, error) {\n\t\t\t// Get the output of the parent task\n\t\t\tvar step1Output SimpleOutput\n\t\t\terr := ctx.ParentOutput(step1, &step1Output)\n\t\t\tif err != nil {\n\t\t\t\treturn nil, err\n\t\t\t}\n\n\t\t\treturn &SimpleOutput{\n\t\t\t\tStep: 2,\n\t\t\t}, nil\n\t\t},\n\t)\n\t// ‼️\n\n\treturn simple\n}\n',
      language: 'go',
      source: 'examples/go/workflows/dag.go',
      highlights: {},
    },
  L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL2dvL3dvcmtmbG93cy9kdXJhYmxlLWV2ZW50Lmdv:
    {
      content:
        'package v1_workflows\n\nimport (\n\t"github.com/hatchet-dev/hatchet/pkg/client/create"\n\tv1 "github.com/hatchet-dev/hatchet/pkg/v1"\n\t"github.com/hatchet-dev/hatchet/pkg/v1/factory"\n\t"github.com/hatchet-dev/hatchet/pkg/v1/workflow"\n\t"github.com/hatchet-dev/hatchet/pkg/worker"\n)\n\ntype DurableEventInput struct {\n\tMessage string\n}\n\ntype EventData struct {\n\tMessage string\n}\n\ntype DurableEventOutput struct {\n\tData EventData\n}\n\nfunc DurableEvent(hatchet v1.HatchetClient) workflow.WorkflowDeclaration[DurableEventInput, DurableEventOutput] {\n\t// ❓ Durable Event\n\tdurableEventTask := factory.NewDurableTask(\n\t\tcreate.StandaloneTask{\n\t\t\tName: "durable-event",\n\t\t},\n\t\tfunc(ctx worker.DurableHatchetContext, input DurableEventInput) (*DurableEventOutput, error) {\n\t\t\teventData, err := ctx.WaitForEvent("user:update", "")\n\n\t\t\tif err != nil {\n\t\t\t\treturn nil, err\n\t\t\t}\n\n\t\t\tv := EventData{}\n\t\t\terr = eventData.Unmarshal(&v)\n\n\t\t\tif err != nil {\n\t\t\t\treturn nil, err\n\t\t\t}\n\n\t\t\treturn &DurableEventOutput{\n\t\t\t\tData: v,\n\t\t\t}, nil\n\t\t},\n\t\thatchet,\n\t)\n\t// !!\n\n\tfactory.NewDurableTask(\n\t\tcreate.StandaloneTask{\n\t\t\tName: "durable-event",\n\t\t},\n\t\tfunc(ctx worker.DurableHatchetContext, input DurableEventInput) (*DurableEventOutput, error) {\n\t\t\t// ❓ Durable Event With Filter\n\t\t\teventData, err := ctx.WaitForEvent("user:update", "input.user_id == \'1234\'")\n\t\t\t// !!\n\n\t\t\tif err != nil {\n\t\t\t\treturn nil, err\n\t\t\t}\n\n\t\t\tv := EventData{}\n\t\t\terr = eventData.Unmarshal(&v)\n\n\t\t\tif err != nil {\n\t\t\t\treturn nil, err\n\t\t\t}\n\n\t\t\treturn &DurableEventOutput{\n\t\t\t\tData: v,\n\t\t\t}, nil\n\t\t},\n\t\thatchet,\n\t)\n\n\treturn durableEventTask\n}\n',
      language: 'go',
      source: 'examples/go/workflows/durable-event.go',
      highlights: {},
    },
  L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL2dvL3dvcmtmbG93cy9kdXJhYmxlLXNsZWVwLmdv:
    {
      content:
        'package v1_workflows\n\nimport (\n\t"strings"\n\t"time"\n\n\t"github.com/hatchet-dev/hatchet/pkg/client/create"\n\tv1 "github.com/hatchet-dev/hatchet/pkg/v1"\n\t"github.com/hatchet-dev/hatchet/pkg/v1/factory"\n\t"github.com/hatchet-dev/hatchet/pkg/v1/workflow"\n\t"github.com/hatchet-dev/hatchet/pkg/worker"\n)\n\ntype DurableSleepInput struct {\n\tMessage string\n}\n\ntype DurableSleepOutput struct {\n\tTransformedMessage string\n}\n\nfunc DurableSleep(hatchet v1.HatchetClient) workflow.WorkflowDeclaration[DurableSleepInput, DurableSleepOutput] {\n\t// ❓ Durable Sleep\n\tsimple := factory.NewDurableTask(\n\t\tcreate.StandaloneTask{\n\t\t\tName: "durable-sleep",\n\t\t},\n\t\tfunc(ctx worker.DurableHatchetContext, input DurableSleepInput) (*DurableSleepOutput, error) {\n\t\t\t_, err := ctx.SleepFor(10 * time.Second)\n\n\t\t\tif err != nil {\n\t\t\t\treturn nil, err\n\t\t\t}\n\n\t\t\treturn &DurableSleepOutput{\n\t\t\t\tTransformedMessage: strings.ToLower(input.Message),\n\t\t\t}, nil\n\t\t},\n\t\thatchet,\n\t)\n\t// !!\n\n\treturn simple\n}\n',
      language: 'go',
      source: 'examples/go/workflows/durable-sleep.go',
      highlights: {},
    },
  L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL2dvL3dvcmtmbG93cy9ub24tcmV0cnlhYmxlLWVycm9yLmdv:
    {
      content:
        'package v1_workflows\n\nimport (\n\t"errors"\n\n\t"github.com/hatchet-dev/hatchet/pkg/client/create"\n\tv1 "github.com/hatchet-dev/hatchet/pkg/v1"\n\t"github.com/hatchet-dev/hatchet/pkg/v1/factory"\n\t"github.com/hatchet-dev/hatchet/pkg/v1/workflow"\n\t"github.com/hatchet-dev/hatchet/pkg/worker"\n)\n\ntype NonRetryableInput struct{}\ntype NonRetryableResult struct{}\n\n// NonRetryableError returns a workflow which throws a non-retryable error\nfunc NonRetryableError(hatchet v1.HatchetClient) workflow.WorkflowDeclaration[NonRetryableInput, NonRetryableResult] {\n\t// ❓ Non Retryable Error\n\tretries := factory.NewTask(\n\t\tcreate.StandaloneTask{\n\t\t\tName:    "non-retryable-task",\n\t\t\tRetries: 3,\n\t\t}, func(ctx worker.HatchetContext, input NonRetryableInput) (*NonRetryableResult, error) {\n\t\t\treturn nil, worker.NewNonRetryableError(errors.New("intentional failure"))\n\t\t},\n\t\thatchet,\n\t)\n\t// ‼️\n\n\treturn retries\n}\n',
      language: 'go',
      source: 'examples/go/workflows/non-retryable-error.go',
      highlights: {},
    },
  L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL2dvL3dvcmtmbG93cy9vbi1jcm9uLmdv:
    {
      content:
        'package v1_workflows\n\nimport (\n\t"strings"\n\n\t"github.com/hatchet-dev/hatchet/pkg/client/create"\n\tv1 "github.com/hatchet-dev/hatchet/pkg/v1"\n\t"github.com/hatchet-dev/hatchet/pkg/v1/factory"\n\t"github.com/hatchet-dev/hatchet/pkg/v1/workflow"\n\t"github.com/hatchet-dev/hatchet/pkg/worker"\n)\n\ntype OnCronInput struct {\n\tMessage string `json:"Message"`\n}\n\ntype JobResult struct {\n\tTransformedMessage string `json:"TransformedMessage"`\n}\n\ntype OnCronOutput struct {\n\tJob JobResult `json:"job"`\n}\n\n// ❓ Workflow Definition Cron Trigger\nfunc OnCron(hatchet v1.HatchetClient) workflow.WorkflowDeclaration[OnCronInput, OnCronOutput] {\n\t// Create a standalone task that transforms a message\n\tcronTask := factory.NewTask(\n\t\tcreate.StandaloneTask{\n\t\t\tName: "on-cron-task",\n\t\t\t// 👀 add a cron expression\n\t\t\tOnCron: []string{"0 0 * * *"}, // Run every day at midnight\n\t\t},\n\t\tfunc(ctx worker.HatchetContext, input OnCronInput) (*OnCronOutput, error) {\n\t\t\treturn &OnCronOutput{\n\t\t\t\tJob: JobResult{\n\t\t\t\t\tTransformedMessage: strings.ToLower(input.Message),\n\t\t\t\t},\n\t\t\t}, nil\n\t\t},\n\t\thatchet,\n\t)\n\n\treturn cronTask\n}\n\n// !!\n',
      language: 'go',
      source: 'examples/go/workflows/on-cron.go',
      highlights: {},
    },
  L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL2dvL3dvcmtmbG93cy9vbi1ldmVudC5nbw__:
    {
      content:
        'package v1_workflows\n\nimport (\n\t"strings"\n\n\t"github.com/hatchet-dev/hatchet/pkg/client/create"\n\tv1 "github.com/hatchet-dev/hatchet/pkg/v1"\n\t"github.com/hatchet-dev/hatchet/pkg/v1/factory"\n\t"github.com/hatchet-dev/hatchet/pkg/v1/workflow"\n\t"github.com/hatchet-dev/hatchet/pkg/worker"\n)\n\ntype EventInput struct {\n\tMessage string\n}\n\ntype LowerTaskOutput struct {\n\tTransformedMessage string\n}\n\ntype UpperTaskOutput struct {\n\tTransformedMessage string\n}\n\n// ❓ Run workflow on event\nconst SimpleEvent = "simple-event:create"\n\nfunc Lower(hatchet v1.HatchetClient) workflow.WorkflowDeclaration[EventInput, LowerTaskOutput] {\n\treturn factory.NewTask(\n\t\tcreate.StandaloneTask{\n\t\t\tName: "lower",\n\t\t\t// 👀 Declare the event that will trigger the workflow\n\t\t\tOnEvents: []string{SimpleEvent},\n\t\t}, func(ctx worker.HatchetContext, input EventInput) (*LowerTaskOutput, error) {\n\t\t\t// Transform the input message to lowercase\n\t\t\treturn &LowerTaskOutput{\n\t\t\t\tTransformedMessage: strings.ToLower(input.Message),\n\t\t\t}, nil\n\t\t},\n\t\thatchet,\n\t)\n}\n\n// !!\n\nfunc Upper(hatchet v1.HatchetClient) workflow.WorkflowDeclaration[EventInput, UpperTaskOutput] {\n\treturn factory.NewTask(\n\t\tcreate.StandaloneTask{\n\t\t\tName:     "upper",\n\t\t\tOnEvents: []string{SimpleEvent},\n\t\t},\n\t\tfunc(ctx worker.HatchetContext, input EventInput) (*UpperTaskOutput, error) {\n\t\t\treturn &UpperTaskOutput{\n\t\t\t\tTransformedMessage: strings.ToUpper(input.Message),\n\t\t\t}, nil\n\t\t},\n\t\thatchet,\n\t)\n}\n',
      language: 'go',
      source: 'examples/go/workflows/on-event.go',
      highlights: {},
    },
  L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL2dvL3dvcmtmbG93cy9vbi1mYWlsdXJlLmdv:
    {
      content:
        'package v1_workflows\n\nimport (\n\t"errors"\n\n\t"github.com/hatchet-dev/hatchet/pkg/client/create"\n\tv1 "github.com/hatchet-dev/hatchet/pkg/v1"\n\t"github.com/hatchet-dev/hatchet/pkg/v1/factory"\n\t"github.com/hatchet-dev/hatchet/pkg/v1/workflow"\n\t"github.com/hatchet-dev/hatchet/pkg/worker"\n)\n\ntype AlwaysFailsOutput struct {\n\tTransformedMessage string\n}\n\ntype OnFailureOutput struct {\n\tFailureRan bool\n}\n\ntype OnFailureSuccessResult struct {\n\tAlwaysFails AlwaysFailsOutput\n}\n\nfunc OnFailure(hatchet v1.HatchetClient) workflow.WorkflowDeclaration[any, OnFailureSuccessResult] {\n\n\tsimple := factory.NewWorkflow[any, OnFailureSuccessResult](\n\t\tcreate.WorkflowCreateOpts[any]{\n\t\t\tName: "on-failure",\n\t\t},\n\t\thatchet,\n\t)\n\n\tsimple.Task(\n\t\tcreate.WorkflowTask[any, OnFailureSuccessResult]{\n\t\t\tName: "AlwaysFails",\n\t\t},\n\t\tfunc(ctx worker.HatchetContext, _ any) (interface{}, error) {\n\t\t\treturn &AlwaysFailsOutput{\n\t\t\t\tTransformedMessage: "always fails",\n\t\t\t}, errors.New("always fails")\n\t\t},\n\t)\n\n\tsimple.OnFailure(\n\t\tcreate.WorkflowOnFailureTask[any, OnFailureSuccessResult]{},\n\t\tfunc(ctx worker.HatchetContext, _ any) (interface{}, error) {\n\t\t\treturn &OnFailureOutput{\n\t\t\t\tFailureRan: true,\n\t\t\t}, nil\n\t\t},\n\t)\n\n\treturn simple\n}\n',
      language: 'go',
      source: 'examples/go/workflows/on-failure.go',
      highlights: {},
    },
  L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL2dvL3dvcmtmbG93cy9wcmlvcml0eS5nbw__:
    {
      content:
        'package v1_workflows\n\nimport (\n\t"time"\n\n\t"github.com/hatchet-dev/hatchet/pkg/client/create"\n\tv1 "github.com/hatchet-dev/hatchet/pkg/v1"\n\t"github.com/hatchet-dev/hatchet/pkg/v1/factory"\n\t"github.com/hatchet-dev/hatchet/pkg/v1/workflow"\n\t"github.com/hatchet-dev/hatchet/pkg/worker"\n)\n\ntype PriorityInput struct {\n\tUserId string `json:"userId"`\n}\n\ntype PriorityOutput struct {\n\tTransformedMessage string `json:"TransformedMessage"`\n}\n\ntype Result struct {\n\tStep PriorityOutput\n}\n\nfunc Priority(hatchet v1.HatchetClient) workflow.WorkflowDeclaration[PriorityInput, Result] {\n\t// Create a standalone task that transforms a message\n\n\t// ❓ Default priority\n\tdefaultPriority := int32(1)\n\n\tworkflow := factory.NewWorkflow[PriorityInput, Result](\n\t\tcreate.WorkflowCreateOpts[PriorityInput]{\n\t\t\tName:            "priority",\n\t\t\tDefaultPriority: &defaultPriority,\n\t\t},\n\t\thatchet,\n\t)\n\t// ‼️\n\n\t// ❓ Defining a Task\n\tworkflow.Task(\n\t\tcreate.WorkflowTask[PriorityInput, Result]{\n\t\t\tName: "step",\n\t\t}, func(ctx worker.HatchetContext, input PriorityInput) (interface{}, error) {\n\t\t\ttime.Sleep(time.Second * 5)\n\t\t\treturn &PriorityOutput{\n\t\t\t\tTransformedMessage: input.UserId,\n\t\t\t}, nil\n\t\t},\n\t)\n\t// ‼️\n\treturn workflow\n}\n\n// !!\n',
      language: 'go',
      source: 'examples/go/workflows/priority.go',
      highlights: {},
    },
  L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL2dvL3dvcmtmbG93cy9yYXRlbGltaXQuZ28_:
    {
      content:
        'package v1_workflows\n\nimport (\n\t"strings"\n\n\t"github.com/hatchet-dev/hatchet/pkg/client/create"\n\t"github.com/hatchet-dev/hatchet/pkg/client/types"\n\tv1 "github.com/hatchet-dev/hatchet/pkg/v1"\n\t"github.com/hatchet-dev/hatchet/pkg/v1/factory"\n\t"github.com/hatchet-dev/hatchet/pkg/v1/features"\n\t"github.com/hatchet-dev/hatchet/pkg/v1/workflow"\n\t"github.com/hatchet-dev/hatchet/pkg/worker"\n)\n\ntype RateLimitInput struct {\n\tUserId string `json:"userId"`\n}\n\ntype RateLimitOutput struct {\n\tTransformedMessage string `json:"TransformedMessage"`\n}\n\nfunc upsertRateLimit(hatchet v1.HatchetClient) {\n\t// ❓ Upsert Rate Limit\n\thatchet.RateLimits().Upsert(\n\t\tfeatures.CreateRatelimitOpts{\n\t\t\tKey:      "api-service-rate-limit",\n\t\t\tLimit:    10,\n\t\t\tDuration: types.Second,\n\t\t},\n\t)\n\t// !!\n}\n\n// ❓ Static Rate Limit\nfunc StaticRateLimit(hatchet v1.HatchetClient) workflow.WorkflowDeclaration[RateLimitInput, RateLimitOutput] {\n\t// Create a standalone task that transforms a message\n\n\t// define the parameters for the rate limit\n\trateLimitKey := "api-service-rate-limit"\n\tunits := 1\n\n\trateLimitTask := factory.NewTask(\n\t\tcreate.StandaloneTask{\n\t\t\tName: "rate-limit-task",\n\t\t\t// 👀 add a static rate limit\n\t\t\tRateLimits: []*types.RateLimit{\n\t\t\t\t{\n\t\t\t\t\tKey:   rateLimitKey,\n\t\t\t\t\tUnits: &units,\n\t\t\t\t},\n\t\t\t},\n\t\t},\n\t\tfunc(ctx worker.HatchetContext, input RateLimitInput) (*RateLimitOutput, error) {\n\t\t\treturn &RateLimitOutput{\n\t\t\t\tTransformedMessage: strings.ToLower(input.UserId),\n\t\t\t}, nil\n\t\t},\n\t\thatchet,\n\t)\n\n\treturn rateLimitTask\n}\n\n// !!\n\n// ❓ Dynamic Rate Limit\nfunc RateLimit(hatchet v1.HatchetClient) workflow.WorkflowDeclaration[RateLimitInput, RateLimitOutput] {\n\t// Create a standalone task that transforms a message\n\n\t// define the parameters for the rate limit\n\texpression := "input.userId"\n\tunits := 1\n\tduration := types.Second\n\n\trateLimitTask := factory.NewTask(\n\t\tcreate.StandaloneTask{\n\t\t\tName: "rate-limit-task",\n\t\t\t// 👀 add a dynamic rate limit\n\t\t\tRateLimits: []*types.RateLimit{\n\t\t\t\t{\n\t\t\t\t\tKeyExpr:  &expression,\n\t\t\t\t\tUnits:    &units,\n\t\t\t\t\tDuration: &duration,\n\t\t\t\t},\n\t\t\t},\n\t\t},\n\t\tfunc(ctx worker.HatchetContext, input RateLimitInput) (*RateLimitOutput, error) {\n\t\t\treturn &RateLimitOutput{\n\t\t\t\tTransformedMessage: strings.ToLower(input.UserId),\n\t\t\t}, nil\n\t\t},\n\t\thatchet,\n\t)\n\n\treturn rateLimitTask\n}\n\n// !!\n',
      language: 'go',
      source: 'examples/go/workflows/ratelimit.go',
      highlights: {},
    },
  L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL2dvL3dvcmtmbG93cy9yZXRyaWVzLmdv:
    {
      content:
        'package v1_workflows\n\nimport (\n\t"errors"\n\t"fmt"\n\n\t"github.com/hatchet-dev/hatchet/pkg/client/create"\n\tv1 "github.com/hatchet-dev/hatchet/pkg/v1"\n\t"github.com/hatchet-dev/hatchet/pkg/v1/factory"\n\t"github.com/hatchet-dev/hatchet/pkg/v1/workflow"\n\t"github.com/hatchet-dev/hatchet/pkg/worker"\n)\n\ntype RetriesInput struct{}\ntype RetriesResult struct{}\n\n// Simple retries example that always fails\nfunc Retries(hatchet v1.HatchetClient) workflow.WorkflowDeclaration[RetriesInput, RetriesResult] {\n\t// ❓ Simple Step Retries\n\tretries := factory.NewTask(\n\t\tcreate.StandaloneTask{\n\t\t\tName:    "retries-task",\n\t\t\tRetries: 3,\n\t\t}, func(ctx worker.HatchetContext, input RetriesInput) (*RetriesResult, error) {\n\t\t\treturn nil, errors.New("intentional failure")\n\t\t},\n\t\thatchet,\n\t)\n\t// ‼️\n\n\treturn retries\n}\n\ntype RetriesWithCountInput struct{}\ntype RetriesWithCountResult struct {\n\tMessage string `json:"message"`\n}\n\n// Retries example that succeeds after a certain number of retries\nfunc RetriesWithCount(hatchet v1.HatchetClient) workflow.WorkflowDeclaration[RetriesWithCountInput, RetriesWithCountResult] {\n\t// ❓ Retries with Count\n\tretriesWithCount := factory.NewTask(\n\t\tcreate.StandaloneTask{\n\t\t\tName:    "fail-twice-task",\n\t\t\tRetries: 3,\n\t\t}, func(ctx worker.HatchetContext, input RetriesWithCountInput) (*RetriesWithCountResult, error) {\n\t\t\t// Get the current retry count\n\t\t\tretryCount := ctx.RetryCount()\n\n\t\t\tfmt.Printf("Retry count: %d\\n", retryCount)\n\n\t\t\tif retryCount < 2 {\n\t\t\t\treturn nil, errors.New("intentional failure")\n\t\t\t}\n\n\t\t\treturn &RetriesWithCountResult{\n\t\t\t\tMessage: "success",\n\t\t\t}, nil\n\t\t},\n\t\thatchet,\n\t)\n\t// ‼️\n\n\treturn retriesWithCount\n}\n\ntype BackoffInput struct{}\ntype BackoffResult struct{}\n\n// Retries example with simple backoff (no configuration in this API version)\nfunc WithBackoff(hatchet v1.HatchetClient) workflow.WorkflowDeclaration[BackoffInput, BackoffResult] {\n\t// ❓ Retries with Backoff\n\twithBackoff := factory.NewTask(\n\t\tcreate.StandaloneTask{\n\t\t\tName: "with-backoff-task",\n\t\t\t// 👀 Maximum number of seconds to wait between retries\n\t\t\tRetries: 3,\n\t\t\t// 👀 Factor to increase the wait time between retries.\n\t\t\tRetryBackoffFactor: 2,\n\t\t\t// 👀 Maximum number of seconds to wait between retries\n\t\t\t// This sequence will be 2s, 4s, 8s, 10s, 10s, 10s... due to the maxSeconds limit\n\t\t\tRetryMaxBackoffSeconds: 10,\n\t\t}, func(ctx worker.HatchetContext, input BackoffInput) (*BackoffResult, error) {\n\t\t\treturn nil, errors.New("intentional failure")\n\t\t},\n\t\thatchet,\n\t)\n\t// ‼️\n\n\treturn withBackoff\n}\n',
      language: 'go',
      source: 'examples/go/workflows/retries.go',
      highlights: {},
    },
  L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL2dvL3dvcmtmbG93cy9zaW1wbGUuZ28_:
    {
      content:
        'package v1_workflows\n\nimport (\n\t"context"\n\t"fmt"\n\t"strings"\n\n\t"github.com/hatchet-dev/hatchet/pkg/client/create"\n\tv1 "github.com/hatchet-dev/hatchet/pkg/v1"\n\t"github.com/hatchet-dev/hatchet/pkg/v1/factory"\n\tv1worker "github.com/hatchet-dev/hatchet/pkg/v1/worker"\n\t"github.com/hatchet-dev/hatchet/pkg/v1/workflow"\n\t"github.com/hatchet-dev/hatchet/pkg/worker"\n)\n\ntype SimpleInput struct {\n\tMessage string\n}\ntype SimpleResult struct {\n\tTransformedMessage string\n}\n\nfunc Simple(hatchet v1.HatchetClient) workflow.WorkflowDeclaration[SimpleInput, SimpleResult] {\n\n\t// Create a simple standalone task using the task factory\n\t// Note the use of typed generics for both input and output\n\n\t// ❓ Declaring a Task\n\tsimple := factory.NewTask(\n\t\tcreate.StandaloneTask{\n\t\t\tName: "simple-task",\n\t\t}, func(ctx worker.HatchetContext, input SimpleInput) (*SimpleResult, error) {\n\t\t\t// Transform the input message to lowercase\n\t\t\treturn &SimpleResult{\n\t\t\t\tTransformedMessage: strings.ToLower(input.Message),\n\t\t\t}, nil\n\t\t},\n\t\thatchet,\n\t)\n\t// ‼️\n\n\t// Example of running a task\n\t_ = func() error {\n\t\t// ❓ Running a Task\n\t\tresult, err := simple.Run(context.Background(), SimpleInput{Message: "Hello, World!"})\n\t\tif err != nil {\n\t\t\treturn err\n\t\t}\n\t\tfmt.Println(result.TransformedMessage)\n\t\t// ‼️\n\t\treturn nil\n\t}\n\n\t// Example of registering a task on a worker\n\t_ = func() error {\n\t\t// ❓ Declaring a Worker\n\t\tw, err := hatchet.Worker(v1worker.WorkerOpts{\n\t\t\tName: "simple-worker",\n\t\t\tWorkflows: []workflow.WorkflowBase{\n\t\t\t\tsimple,\n\t\t\t},\n\t\t})\n\t\tif err != nil {\n\t\t\treturn err\n\t\t}\n\t\terr = w.StartBlocking(context.Background())\n\t\tif err != nil {\n\t\t\treturn err\n\t\t}\n\t\t// ‼️\n\t\treturn nil\n\t}\n\n\treturn simple\n}\n\nfunc ParentTask(hatchet v1.HatchetClient) workflow.WorkflowDeclaration[SimpleInput, SimpleResult] {\n\n\t// ❓ Spawning Tasks from within a Task\n\tsimple := Simple(hatchet)\n\n\tparent := factory.NewTask(\n\t\tcreate.StandaloneTask{\n\t\t\tName: "parent-task",\n\t\t}, func(ctx worker.HatchetContext, input SimpleInput) (*SimpleResult, error) {\n\n\t\t\t// Run the child task\n\t\t\tchild, err := workflow.RunChildWorkflow(ctx, simple, SimpleInput{Message: input.Message})\n\t\t\tif err != nil {\n\t\t\t\treturn nil, err\n\t\t\t}\n\n\t\t\t// Transform the input message to lowercase\n\t\t\treturn &SimpleResult{\n\t\t\t\tTransformedMessage: child.TransformedMessage,\n\t\t\t}, nil\n\t\t},\n\t\thatchet,\n\t)\n\t// ‼️\n\n\treturn parent\n}\n',
      language: 'go',
      source: 'examples/go/workflows/simple.go',
      highlights: {},
    },
  L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL2dvL3dvcmtmbG93cy9zdGlja3kuZ28_:
    {
      content:
        'package v1_workflows\n\nimport (\n\t"fmt"\n\n\t"github.com/hatchet-dev/hatchet/pkg/client/create"\n\tv1 "github.com/hatchet-dev/hatchet/pkg/v1"\n\t"github.com/hatchet-dev/hatchet/pkg/v1/factory"\n\t"github.com/hatchet-dev/hatchet/pkg/v1/workflow"\n\t"github.com/hatchet-dev/hatchet/pkg/worker"\n)\n\ntype StickyInput struct{}\n\ntype StickyResult struct {\n\tResult string `json:"result"`\n}\n\ntype StickyDagResult struct {\n\tStickyTask1 StickyResult `json:"sticky-task-1"`\n\tStickyTask2 StickyResult `json:"sticky-task-2"`\n}\n\nfunc StickyDag(hatchet v1.HatchetClient) workflow.WorkflowDeclaration[StickyInput, StickyDagResult] {\n\tstickyDag := factory.NewWorkflow[StickyInput, StickyDagResult](\n\t\tcreate.WorkflowCreateOpts[StickyInput]{\n\t\t\tName: "sticky-dag",\n\t\t},\n\t\thatchet,\n\t)\n\n\tstickyDag.Task(\n\t\tcreate.WorkflowTask[StickyInput, StickyDagResult]{\n\t\t\tName: "sticky-task",\n\t\t},\n\t\tfunc(ctx worker.HatchetContext, input StickyInput) (interface{}, error) {\n\t\t\tworkerId := ctx.Worker().ID()\n\n\t\t\treturn &StickyResult{\n\t\t\t\tResult: workerId,\n\t\t\t}, nil\n\t\t},\n\t)\n\n\tstickyDag.Task(\n\t\tcreate.WorkflowTask[StickyInput, StickyDagResult]{\n\t\t\tName: "sticky-task-2",\n\t\t},\n\t\tfunc(ctx worker.HatchetContext, input StickyInput) (interface{}, error) {\n\t\t\tworkerId := ctx.Worker().ID()\n\n\t\t\treturn &StickyResult{\n\t\t\t\tResult: workerId,\n\t\t\t}, nil\n\t\t},\n\t)\n\n\treturn stickyDag\n}\n\nfunc Sticky(hatchet v1.HatchetClient) workflow.WorkflowDeclaration[StickyInput, StickyResult] {\n\tsticky := factory.NewTask(\n\t\tcreate.StandaloneTask{\n\t\t\tName:    "sticky-task",\n\t\t\tRetries: 3,\n\t\t}, func(ctx worker.HatchetContext, input StickyInput) (*StickyResult, error) {\n\t\t\t// Run a child workflow on the same worker\n\t\t\tchildWorkflow := Child(hatchet)\n\t\t\tsticky := true\n\t\t\tchildResult, err := childWorkflow.RunAsChild(ctx, ChildInput{N: 1}, workflow.RunAsChildOpts{\n\t\t\t\tSticky: &sticky,\n\t\t\t})\n\n\t\t\tif err != nil {\n\t\t\t\treturn nil, err\n\t\t\t}\n\n\t\t\treturn &StickyResult{\n\t\t\t\tResult: fmt.Sprintf("child-result-%d", childResult.Value),\n\t\t\t}, nil\n\t\t},\n\t\thatchet,\n\t)\n\n\treturn sticky\n}\n',
      language: 'go',
      source: 'examples/go/workflows/sticky.go',
      highlights: {},
    },
  L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL2dvL3dvcmtmbG93cy90aW1lb3V0cy5nbw__:
    {
      content:
        'package v1_workflows\n\nimport (\n\t"errors"\n\t"time"\n\n\t"github.com/hatchet-dev/hatchet/pkg/client/create"\n\tv1 "github.com/hatchet-dev/hatchet/pkg/v1"\n\t"github.com/hatchet-dev/hatchet/pkg/v1/factory"\n\t"github.com/hatchet-dev/hatchet/pkg/v1/workflow"\n\t"github.com/hatchet-dev/hatchet/pkg/worker"\n)\n\ntype TimeoutInput struct{}\ntype TimeoutResult struct {\n\tCompleted bool\n}\n\nfunc Timeout(hatchet v1.HatchetClient) workflow.WorkflowDeclaration[TimeoutInput, TimeoutResult] {\n\n\t// Create a task with a timeout of 3 seconds that tries to sleep for 10 seconds\n\ttimeout := factory.NewTask(\n\t\tcreate.StandaloneTask{\n\t\t\tName:             "timeout-task",\n\t\t\tExecutionTimeout: 3 * time.Second, // Task will timeout after 3 seconds\n\t\t}, func(ctx worker.HatchetContext, input TimeoutInput) (*TimeoutResult, error) {\n\t\t\t// Sleep for 10 seconds\n\t\t\ttime.Sleep(10 * time.Second)\n\n\t\t\t// Check if the context was cancelled due to timeout\n\t\t\tselect {\n\t\t\tcase <-ctx.Done():\n\t\t\t\treturn nil, errors.New("Task timed out")\n\t\t\tdefault:\n\t\t\t\t// Continue execution\n\t\t\t}\n\n\t\t\treturn &TimeoutResult{\n\t\t\t\tCompleted: true,\n\t\t\t}, nil\n\t\t},\n\t\thatchet,\n\t)\n\n\treturn timeout\n}\n',
      language: 'go',
      source: 'examples/go/workflows/timeouts.go',
      highlights: {},
    },
  L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL2dvL3pfdjAvYXNzaWdubWVudC1hZmZpbml0eS9tYWluLmdv:
    {
      content:
        'package main\n\nimport (\n\t"fmt"\n\n\t"github.com/joho/godotenv"\n\n\t"github.com/hatchet-dev/hatchet/pkg/cmdutils"\n)\n\ntype userCreateEvent struct {\n\tUsername string            `json:"username"`\n\tUserID   string            `json:"user_id"`\n\tData     map[string]string `json:"data"`\n}\n\ntype stepOneOutput struct {\n\tMessage string `json:"message"`\n}\n\nfunc main() {\n\terr := godotenv.Load()\n\tif err != nil {\n\t\tpanic(err)\n\t}\n\n\tch := cmdutils.InterruptChan()\n\tcleanup, err := run()\n\tif err != nil {\n\t\tpanic(err)\n\t}\n\n\t<-ch\n\n\tif err := cleanup(); err != nil {\n\t\tpanic(fmt.Errorf("cleanup() error = %v", err))\n\t}\n}\n',
      language: 'go',
      source: 'examples/go/z_v0/assignment-affinity/main.go',
      highlights: {},
    },
  L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL2dvL3pfdjAvYXNzaWdubWVudC1hZmZpbml0eS9ydW4uZ28_:
    {
      content:
        'package main\n\nimport (\n\t"context"\n\t"fmt"\n\t"log"\n\t"time"\n\n\t"github.com/hatchet-dev/hatchet/pkg/client"\n\t"github.com/hatchet-dev/hatchet/pkg/client/types"\n\t"github.com/hatchet-dev/hatchet/pkg/worker"\n)\n\nfunc run() (func() error, error) {\n\tc, err := client.New()\n\tif err != nil {\n\t\treturn nil, fmt.Errorf("error creating client: %w", err)\n\t}\n\n\tw, err := worker.NewWorker(\n\t\tworker.WithClient(\n\t\t\tc,\n\t\t),\n\t\tworker.WithLabels(map[string]interface{}{\n\t\t\t"model":  "fancy-ai-model-v2",\n\t\t\t"memory": 1024,\n\t\t}),\n\t)\n\tif err != nil {\n\t\treturn nil, fmt.Errorf("error creating worker: %w", err)\n\t}\n\n\terr = w.RegisterWorkflow(\n\t\t&worker.WorkflowJob{\n\t\t\tOn:          worker.Events("user:create:affinity"),\n\t\t\tName:        "affinity",\n\t\t\tDescription: "affinity",\n\t\t\tSteps: []*worker.WorkflowStep{\n\t\t\t\tworker.Fn(func(ctx worker.HatchetContext) (result *stepOneOutput, err error) {\n\n\t\t\t\t\tmodel := ctx.Worker().GetLabels()["model"]\n\n\t\t\t\t\tif model != "fancy-ai-model-v3" {\n\t\t\t\t\t\tctx.Worker().UpsertLabels(map[string]interface{}{\n\t\t\t\t\t\t\t"model": nil,\n\t\t\t\t\t\t})\n\t\t\t\t\t\t// Do something to load the model\n\t\t\t\t\t\tctx.Worker().UpsertLabels(map[string]interface{}{\n\t\t\t\t\t\t\t"model": "fancy-ai-model-v3",\n\t\t\t\t\t\t})\n\t\t\t\t\t}\n\n\t\t\t\t\treturn &stepOneOutput{\n\t\t\t\t\t\tMessage: ctx.Worker().ID(),\n\t\t\t\t\t}, nil\n\t\t\t\t}).\n\t\t\t\t\tSetName("step-one").\n\t\t\t\t\tSetDesiredLabels(map[string]*types.DesiredWorkerLabel{\n\t\t\t\t\t\t"model": {\n\t\t\t\t\t\t\tValue:  "fancy-ai-model-v3",\n\t\t\t\t\t\t\tWeight: 10,\n\t\t\t\t\t\t},\n\t\t\t\t\t\t"memory": {\n\t\t\t\t\t\t\tValue:      512,\n\t\t\t\t\t\t\tRequired:   true,\n\t\t\t\t\t\t\tComparator: types.ComparatorPtr(types.WorkerLabelComparator_GREATER_THAN),\n\t\t\t\t\t\t},\n\t\t\t\t\t}),\n\t\t\t},\n\t\t},\n\t)\n\tif err != nil {\n\t\treturn nil, fmt.Errorf("error registering workflow: %w", err)\n\t}\n\n\tgo func() {\n\t\tlog.Printf("pushing event")\n\n\t\ttestEvent := userCreateEvent{\n\t\t\tUsername: "echo-test",\n\t\t\tUserID:   "1234",\n\t\t\tData: map[string]string{\n\t\t\t\t"test": "test",\n\t\t\t},\n\t\t}\n\n\t\t// push an event\n\t\terr := c.Event().Push(\n\t\t\tcontext.Background(),\n\t\t\t"user:create:affinity",\n\t\t\ttestEvent,\n\t\t)\n\t\tif err != nil {\n\t\t\tpanic(fmt.Errorf("error pushing event: %w", err))\n\t\t}\n\n\t\ttime.Sleep(10 * time.Second)\n\t}()\n\n\tcleanup, err := w.Start()\n\tif err != nil {\n\t\treturn nil, fmt.Errorf("error starting worker: %w", err)\n\t}\n\n\treturn cleanup, nil\n}\n',
      language: 'go',
      source: 'examples/go/z_v0/assignment-affinity/run.go',
      highlights: {},
    },
  L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL2dvL3pfdjAvYXNzaWdubWVudC1zdGlja3kvbWFpbi5nbw__:
    {
      content:
        'package main\n\nimport (\n\t"fmt"\n\n\t"github.com/joho/godotenv"\n\n\t"github.com/hatchet-dev/hatchet/pkg/cmdutils"\n)\n\ntype userCreateEvent struct {\n\tUsername string            `json:"username"`\n\tUserID   string            `json:"user_id"`\n\tData     map[string]string `json:"data"`\n}\n\ntype stepOneOutput struct {\n\tMessage string `json:"message"`\n}\n\nfunc main() {\n\terr := godotenv.Load()\n\tif err != nil {\n\t\tpanic(err)\n\t}\n\n\tch := cmdutils.InterruptChan()\n\tcleanup, err := run()\n\tif err != nil {\n\t\tpanic(err)\n\t}\n\n\t<-ch\n\n\tif err := cleanup(); err != nil {\n\t\tpanic(fmt.Errorf("cleanup() error = %v", err))\n\t}\n}\n',
      language: 'go',
      source: 'examples/go/z_v0/assignment-sticky/main.go',
      highlights: {},
    },
  L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL2dvL3pfdjAvYXNzaWdubWVudC1zdGlja3kvcnVuLmdv:
    {
      content:
        'package main\n\nimport (\n\t"context"\n\t"fmt"\n\t"log"\n\t"time"\n\n\t"github.com/hatchet-dev/hatchet/pkg/client"\n\t"github.com/hatchet-dev/hatchet/pkg/client/types"\n\t"github.com/hatchet-dev/hatchet/pkg/worker"\n)\n\nfunc run() (func() error, error) {\n\tc, err := client.New()\n\tif err != nil {\n\t\treturn nil, fmt.Errorf("error creating client: %w", err)\n\t}\n\n\tw, err := worker.NewWorker(\n\t\tworker.WithClient(\n\t\t\tc,\n\t\t),\n\t)\n\tif err != nil {\n\t\treturn nil, fmt.Errorf("error creating worker: %w", err)\n\t}\n\n\t// ❓ StickyWorker\n\n\terr = w.RegisterWorkflow(\n\t\t&worker.WorkflowJob{\n\t\t\tOn:          worker.Events("user:create:sticky"),\n\t\t\tName:        "sticky",\n\t\t\tDescription: "sticky",\n\t\t\t// 👀 Specify a sticky strategy when declaring the workflow\n\t\t\tStickyStrategy: types.StickyStrategyPtr(types.StickyStrategy_HARD),\n\t\t\tSteps: []*worker.WorkflowStep{\n\t\t\t\tworker.Fn(func(ctx worker.HatchetContext) (result *stepOneOutput, err error) {\n\n\t\t\t\t\tsticky := true\n\n\t\t\t\t\t_, err = ctx.SpawnWorkflow("sticky-child", nil, &worker.SpawnWorkflowOpts{\n\t\t\t\t\t\tSticky: &sticky,\n\t\t\t\t\t})\n\n\t\t\t\t\tif err != nil {\n\t\t\t\t\t\treturn nil, fmt.Errorf("error spawning workflow: %w", err)\n\t\t\t\t\t}\n\n\t\t\t\t\treturn &stepOneOutput{\n\t\t\t\t\t\tMessage: ctx.Worker().ID(),\n\t\t\t\t\t}, nil\n\t\t\t\t}).SetName("step-one"),\n\t\t\t\tworker.Fn(func(ctx worker.HatchetContext) (result *stepOneOutput, err error) {\n\t\t\t\t\treturn &stepOneOutput{\n\t\t\t\t\t\tMessage: ctx.Worker().ID(),\n\t\t\t\t\t}, nil\n\t\t\t\t}).SetName("step-two").AddParents("step-one"),\n\t\t\t\tworker.Fn(func(ctx worker.HatchetContext) (result *stepOneOutput, err error) {\n\t\t\t\t\treturn &stepOneOutput{\n\t\t\t\t\t\tMessage: ctx.Worker().ID(),\n\t\t\t\t\t}, nil\n\t\t\t\t}).SetName("step-three").AddParents("step-two"),\n\t\t\t},\n\t\t},\n\t)\n\n\t// ‼️\n\n\tif err != nil {\n\t\treturn nil, fmt.Errorf("error registering workflow: %w", err)\n\t}\n\n\t// ❓ StickyChild\n\n\terr = w.RegisterWorkflow(\n\t\t&worker.WorkflowJob{\n\t\t\tOn:          worker.NoTrigger(),\n\t\t\tName:        "sticky-child",\n\t\t\tDescription: "sticky",\n\t\t\tSteps: []*worker.WorkflowStep{\n\t\t\t\tworker.Fn(func(ctx worker.HatchetContext) (result *stepOneOutput, err error) {\n\t\t\t\t\treturn &stepOneOutput{\n\t\t\t\t\t\tMessage: ctx.Worker().ID(),\n\t\t\t\t\t}, nil\n\t\t\t\t}).SetName("step-one"),\n\t\t\t},\n\t\t},\n\t)\n\n\t// ‼️\n\n\tif err != nil {\n\t\treturn nil, fmt.Errorf("error registering workflow: %w", err)\n\t}\n\n\tgo func() {\n\t\tlog.Printf("pushing event")\n\n\t\ttestEvent := userCreateEvent{\n\t\t\tUsername: "echo-test",\n\t\t\tUserID:   "1234",\n\t\t\tData: map[string]string{\n\t\t\t\t"test": "test",\n\t\t\t},\n\t\t}\n\n\t\t// push an event\n\t\terr := c.Event().Push(\n\t\t\tcontext.Background(),\n\t\t\t"user:create:sticky",\n\t\t\ttestEvent,\n\t\t)\n\t\tif err != nil {\n\t\t\tpanic(fmt.Errorf("error pushing event: %w", err))\n\t\t}\n\n\t\ttime.Sleep(10 * time.Second)\n\t}()\n\n\tcleanup, err := w.Start()\n\tif err != nil {\n\t\treturn nil, fmt.Errorf("error starting worker: %w", err)\n\t}\n\n\treturn cleanup, nil\n}\n',
      language: 'go',
      source: 'examples/go/z_v0/assignment-sticky/run.go',
      highlights: {},
    },
  L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL2dvL3pfdjAvYnVsa19pbXBvcnRzL21haW4uZ28_:
    {
      content:
        'package main\n\nimport (\n\t"context"\n\t"fmt"\n\t"log"\n\n\t"github.com/joho/godotenv"\n\n\t"github.com/hatchet-dev/hatchet/pkg/client"\n\t"github.com/hatchet-dev/hatchet/pkg/worker"\n)\n\ntype userCreateEvent struct {\n\tUsername string            `json:"username"`\n\tUserID   string            `json:"user_id"`\n\tData     map[string]string `json:"data"`\n}\n\ntype stepOneOutput struct {\n\tMessage string `json:"message"`\n}\n\nfunc main() {\n\terr := godotenv.Load()\n\tif err != nil {\n\t\tpanic(err)\n\t}\n\n\t_, err = run()\n\tif err != nil {\n\t\tpanic(err)\n\t}\n\n}\n\nfunc run() (func() error, error) {\n\tc, err := client.New()\n\n\tif err != nil {\n\t\treturn nil, fmt.Errorf("error creating client: %w", err)\n\t}\n\n\tw, err := worker.NewWorker(\n\t\tworker.WithClient(\n\t\t\tc,\n\t\t),\n\t)\n\tif err != nil {\n\t\treturn nil, fmt.Errorf("error creating worker: %w", err)\n\t}\n\n\ttestSvc := w.NewService("test")\n\n\terr = testSvc.RegisterWorkflow(\n\t\t&worker.WorkflowJob{\n\t\t\tOn:          worker.Events("user:create:bulk"),\n\t\t\tName:        "bulk",\n\t\t\tDescription: "This runs after an update to the user model.",\n\t\t\tSteps: []*worker.WorkflowStep{\n\t\t\t\tworker.Fn(func(ctx worker.HatchetContext) (result *stepOneOutput, err error) {\n\t\t\t\t\tinput := &userCreateEvent{}\n\n\t\t\t\t\terr = ctx.WorkflowInput(input)\n\n\t\t\t\t\tif err != nil {\n\t\t\t\t\t\treturn nil, err\n\t\t\t\t\t}\n\n\t\t\t\t\tlog.Printf("step-one")\n\n\t\t\t\t\treturn &stepOneOutput{\n\t\t\t\t\t\tMessage: "Username is: " + input.Username,\n\t\t\t\t\t}, nil\n\t\t\t\t},\n\t\t\t\t),\n\t\t\t},\n\t\t},\n\t)\n\tif err != nil {\n\t\treturn nil, fmt.Errorf("error registering workflow: %w", err)\n\t}\n\n\tvar events []client.EventWithAdditionalMetadata\n\n\t// 20000 times to test the bulk push\n\n\tfor i := 0; i < 20000; i++ {\n\t\ttestEvent := userCreateEvent{\n\t\t\tUsername: "echo-test",\n\t\t\tUserID:   "1234 " + fmt.Sprint(i),\n\t\t\tData: map[string]string{\n\t\t\t\t"test": "test " + fmt.Sprint(i),\n\t\t\t},\n\t\t}\n\t\tevents = append(events, client.EventWithAdditionalMetadata{\n\t\t\tEvent:              testEvent,\n\t\t\tAdditionalMetadata: map[string]string{"hello": "world " + fmt.Sprint(i)},\n\t\t\tKey:                "user:create:bulk",\n\t\t})\n\t}\n\n\tlog.Printf("pushing event user:create:bulk")\n\n\terr = c.Event().BulkPush(\n\t\tcontext.Background(),\n\t\tevents,\n\t)\n\tif err != nil {\n\t\tpanic(fmt.Errorf("error pushing event: %w", err))\n\t}\n\n\treturn nil, nil\n\n}\n',
      language: 'go',
      source: 'examples/go/z_v0/bulk_imports/main.go',
      highlights: {},
    },
  L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL2dvL3pfdjAvYnVsa193b3JrZmxvd3MvbWFpbi5nbw__:
    {
      content:
        'package main\n\nimport (\n\t"fmt"\n\t"log"\n\t"time"\n\n\t"github.com/joho/godotenv"\n\n\t"github.com/hatchet-dev/hatchet/pkg/client"\n\t"github.com/hatchet-dev/hatchet/pkg/client/types"\n\t"github.com/hatchet-dev/hatchet/pkg/cmdutils"\n\t"github.com/hatchet-dev/hatchet/pkg/worker"\n)\n\ntype userCreateEvent struct {\n\tUsername string            `json:"username"`\n\tUserID   string            `json:"user_id"`\n\tData     map[string]string `json:"data"`\n}\n\ntype stepOneOutput struct {\n\tMessage string `json:"message"`\n}\n\nfunc main() {\n\terr := godotenv.Load()\n\tif err != nil {\n\t\tpanic(err)\n\t}\n\n\tch := cmdutils.InterruptChan()\n\n\tif err != nil {\n\t\tpanic(err)\n\t}\n\tworkflowName := "simple-bulk-workflow"\n\tc, err := client.New()\n\n\tif err != nil {\n\t\tpanic(fmt.Errorf("error creating client: %w", err))\n\t}\n\n\t_, err = registerWorkflow(c, workflowName)\n\n\tif err != nil {\n\t\tpanic(fmt.Errorf("error registering workflow: %w", err))\n\t}\n\n\tquantity := 999\n\n\toverallStart := time.Now()\n\titerations := 10\n\tfor i := 0; i < iterations; i++ {\n\t\tstartTime := time.Now()\n\n\t\tfmt.Printf("Running the %dth bulk workflow \\n", i)\n\n\t\terr = runBulk(workflowName, quantity)\n\t\tif err != nil {\n\t\t\tpanic(err)\n\t\t}\n\t\tfmt.Printf("Time taken to queue %dth bulk workflow: %v\\n", i, time.Since(startTime))\n\t}\n\tfmt.Println("Overall time taken: ", time.Since(overallStart))\n\tfmt.Printf("That is %d workflows per second\\n", int(float64(quantity*iterations)/time.Since(overallStart).Seconds()))\n\tfmt.Println("Starting the worker")\n\n\t// err = runSingles(workflowName, quantity)\n\t// if err != nil {\n\t// \tpanic(err)\n\t// }\n\n\tif err != nil {\n\t\tpanic(fmt.Errorf("error creating client: %w", err))\n\t}\n\n\t// I want to start the wofklow worker here\n\n\tw, err := registerWorkflow(c, workflowName)\n\tif err != nil {\n\t\tpanic(fmt.Errorf("error creating worker: %w", err))\n\t}\n\n\tcleanup, err := w.Start()\n\tfmt.Println("Starting the worker")\n\n\tif err != nil {\n\t\tpanic(fmt.Errorf("error starting worker: %w", err))\n\t}\n\n\t<-ch\n\n\tif err := cleanup(); err != nil {\n\t\tpanic(fmt.Errorf("error cleaning up: %w", err))\n\t}\n\n}\n\nfunc getConcurrencyKey(ctx worker.HatchetContext) (string, error) {\n\treturn "my-key", nil\n}\n\nfunc registerWorkflow(c client.Client, workflowName string) (w *worker.Worker, err error) {\n\n\tw, err = worker.NewWorker(\n\t\tworker.WithClient(\n\t\t\tc,\n\t\t),\n\t)\n\tif err != nil {\n\t\treturn nil, fmt.Errorf("error creating worker: %w", err)\n\t}\n\n\terr = w.RegisterWorkflow(\n\t\t&worker.WorkflowJob{\n\t\t\tOn:          worker.Events("user:create:bulk-simple"),\n\t\t\tName:        workflowName,\n\t\t\tConcurrency: worker.Concurrency(getConcurrencyKey).MaxRuns(200).LimitStrategy(types.GroupRoundRobin),\n\t\t\tDescription: "This runs after an update to the user model.",\n\t\t\tSteps: []*worker.WorkflowStep{\n\t\t\t\tworker.Fn(func(ctx worker.HatchetContext) (result *stepOneOutput, err error) {\n\t\t\t\t\tinput := &userCreateEvent{}\n\n\t\t\t\t\terr = ctx.WorkflowInput(input)\n\n\t\t\t\t\tif err != nil {\n\t\t\t\t\t\treturn nil, err\n\t\t\t\t\t}\n\n\t\t\t\t\tlog.Printf("step-one")\n\n\t\t\t\t\treturn &stepOneOutput{\n\t\t\t\t\t\tMessage: "Username is: " + input.Username,\n\t\t\t\t\t}, nil\n\t\t\t\t},\n\t\t\t\t).SetName("step-one"),\n\t\t\t\tworker.Fn(func(ctx worker.HatchetContext) (result *stepOneOutput, err error) {\n\t\t\t\t\tinput := &stepOneOutput{}\n\t\t\t\t\terr = ctx.StepOutput("step-one", input)\n\n\t\t\t\t\tif err != nil {\n\t\t\t\t\t\treturn nil, err\n\t\t\t\t\t}\n\n\t\t\t\t\tlog.Printf("step-two")\n\n\t\t\t\t\treturn &stepOneOutput{\n\t\t\t\t\t\tMessage: "Above message is: " + input.Message,\n\t\t\t\t\t}, nil\n\t\t\t\t}).SetName("step-two").AddParents("step-one"),\n\t\t\t},\n\t\t},\n\t)\n\tif err != nil {\n\t\treturn nil, fmt.Errorf("error registering workflow: %w", err)\n\t}\n\treturn w, nil\n}\n',
      language: 'go',
      source: 'examples/go/z_v0/bulk_workflows/main.go',
      highlights: {},
    },
  L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL2dvL3pfdjAvYnVsa193b3JrZmxvd3MvcnVuLmdv:
    {
      content:
        'package main\n\nimport (\n\t"fmt"\n\t"log"\n\n\t"github.com/hatchet-dev/hatchet/pkg/client"\n)\n\nfunc runBulk(workflowName string, quantity int) error {\n\tc, err := client.New()\n\tif err != nil {\n\t\treturn fmt.Errorf("error creating client: %w", err)\n\t}\n\n\tlog.Printf("pushing %d workflows in bulk", quantity)\n\n\tvar workflows []*client.WorkflowRun\n\tfor i := 0; i < quantity; i++ {\n\t\tdata := map[string]interface{}{\n\t\t\t"username": fmt.Sprintf("echo-test-%d", i),\n\t\t\t"user_id":  fmt.Sprintf("1234-%d", i),\n\t\t}\n\t\tworkflows = append(workflows, &client.WorkflowRun{\n\t\t\tName:  workflowName,\n\t\t\tInput: data,\n\t\t\tOptions: []client.RunOptFunc{\n\t\t\t\t// setting a dedupe key so these shouldn\'t all run\n\t\t\t\tclient.WithRunMetadata(map[string]interface{}{\n\t\t\t\t\t// "dedupe": "dedupe1",\n\t\t\t\t}),\n\t\t\t},\n\t\t})\n\n\t}\n\n\touts, err := c.Admin().BulkRunWorkflow(workflows)\n\tif err != nil {\n\t\tpanic(fmt.Errorf("error pushing event: %w", err))\n\t}\n\n\tfor _, out := range outs {\n\t\tlog.Printf("workflow run id: %v", out)\n\t}\n\n\treturn nil\n\n}\n\nfunc runSingles(workflowName string, quantity int) error {\n\tc, err := client.New()\n\tif err != nil {\n\t\treturn fmt.Errorf("error creating client: %w", err)\n\t}\n\n\tlog.Printf("pushing %d single workflows", quantity)\n\n\tvar workflows []*client.WorkflowRun\n\tfor i := 0; i < quantity; i++ {\n\t\tdata := map[string]interface{}{\n\t\t\t"username": fmt.Sprintf("echo-test-%d", i),\n\t\t\t"user_id":  fmt.Sprintf("1234-%d", i),\n\t\t}\n\t\tworkflows = append(workflows, &client.WorkflowRun{\n\t\t\tName:  workflowName,\n\t\t\tInput: data,\n\t\t\tOptions: []client.RunOptFunc{\n\t\t\t\tclient.WithRunMetadata(map[string]interface{}{\n\t\t\t\t\t// "dedupe": "dedupe1",\n\t\t\t\t}),\n\t\t\t},\n\t\t})\n\t}\n\n\tfor _, wf := range workflows {\n\n\t\tgo func() {\n\t\t\tout, err := c.Admin().RunWorkflow(wf.Name, wf.Input, wf.Options...)\n\t\t\tif err != nil {\n\t\t\t\tpanic(fmt.Errorf("error pushing event: %w", err))\n\t\t\t}\n\n\t\t\tlog.Printf("workflow run id: %v", out)\n\t\t}()\n\n\t}\n\n\treturn nil\n}\n',
      language: 'go',
      source: 'examples/go/z_v0/bulk_workflows/run.go',
      highlights: {},
    },
  L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL2dvL3pfdjAvY2FuY2VsbGF0aW9uL21haW4uZ28_:
    {
      content:
        'package main\n\nimport (\n\t"fmt"\n\n\t"github.com/joho/godotenv"\n\n\t"github.com/hatchet-dev/hatchet/pkg/cmdutils"\n)\n\ntype userCreateEvent struct {\n\tUsername string            `json:"username"`\n\tUserID   string            `json:"user_id"`\n\tData     map[string]string `json:"data"`\n}\n\ntype stepOneOutput struct {\n\tMessage string `json:"message"`\n}\n\nfunc main() {\n\terr := godotenv.Load()\n\tif err != nil {\n\t\tpanic(err)\n\t}\n\n\tevents := make(chan string, 50)\n\tch := cmdutils.InterruptChan()\n\tcleanup, err := run(events)\n\tif err != nil {\n\t\tpanic(err)\n\t}\n\n\t<-ch\n\n\tif err := cleanup(); err != nil {\n\t\tpanic(fmt.Errorf("cleanup() error = %v", err))\n\t}\n}\n',
      language: 'go',
      source: 'examples/go/z_v0/cancellation/main.go',
      highlights: {},
    },
  L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL2dvL3pfdjAvY2FuY2VsbGF0aW9uL21haW5fZTJlX3Rlc3QuZ28_:
    {
      content:
        '//go:build e2e\n\npackage main\n\nimport (\n\t"context"\n\t"testing"\n\t"time"\n\n\t"github.com/stretchr/testify/assert"\n\n\t"github.com/hatchet-dev/hatchet/internal/testutils"\n)\n\nfunc TestCancellation(t *testing.T) {\n\ttestutils.Prepare(t)\n\n\tctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)\n\tdefer cancel()\n\n\tevents := make(chan string, 50)\n\n\tcleanup, err := run(events)\n\tif err != nil {\n\t\tt.Fatalf("run() error = %s", err)\n\t}\n\n\tvar items []string\n\nouter:\n\tfor {\n\t\tselect {\n\t\tcase item := <-events:\n\t\t\titems = append(items, item)\n\t\tcase <-ctx.Done():\n\t\t\tbreak outer\n\t\t}\n\t}\n\n\tassert.Equal(t, []string{\n\t\t"done",\n\t}, items)\n\n\tif err := cleanup(); err != nil {\n\t\tt.Fatalf("cleanup() error = %s", err)\n\t}\n}\n',
      language: 'go',
      source: 'examples/go/z_v0/cancellation/main_e2e_test.go',
      highlights: {},
    },
  L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL2dvL3pfdjAvY2FuY2VsbGF0aW9uL3J1bi5nbw__:
    {
      content:
        'package main\n\nimport (\n\t"context"\n\t"fmt"\n\t"log"\n\t"time"\n\n\t"github.com/google/uuid"\n\t"github.com/hatchet-dev/hatchet/pkg/client"\n\t"github.com/hatchet-dev/hatchet/pkg/client/rest"\n\t"github.com/hatchet-dev/hatchet/pkg/worker"\n)\n\nfunc run(events chan<- string) (func() error, error) {\n\tc, err := client.New()\n\tif err != nil {\n\t\treturn nil, fmt.Errorf("error creating client: %w", err)\n\t}\n\n\tw, err := worker.NewWorker(\n\t\tworker.WithClient(\n\t\t\tc,\n\t\t),\n\t)\n\tif err != nil {\n\t\treturn nil, fmt.Errorf("error creating worker: %w", err)\n\t}\n\n\terr = w.RegisterWorkflow(\n\t\t&worker.WorkflowJob{\n\t\t\tOn:          worker.Events("user:create:cancellation"),\n\t\t\tName:        "cancellation",\n\t\t\tDescription: "cancellation",\n\t\t\tSteps: []*worker.WorkflowStep{\n\t\t\t\tworker.Fn(func(ctx worker.HatchetContext) (result *stepOneOutput, err error) {\n\t\t\t\t\tselect {\n\t\t\t\t\tcase <-ctx.Done():\n\t\t\t\t\t\tevents <- "done"\n\t\t\t\t\t\tlog.Printf("context cancelled")\n\t\t\t\t\t\treturn nil, nil\n\t\t\t\t\tcase <-time.After(30 * time.Second):\n\t\t\t\t\t\tlog.Printf("workflow never cancelled")\n\t\t\t\t\t\treturn &stepOneOutput{\n\t\t\t\t\t\t\tMessage: "done",\n\t\t\t\t\t\t}, nil\n\t\t\t\t\t}\n\t\t\t\t}).SetName("step-one"),\n\t\t\t},\n\t\t},\n\t)\n\tif err != nil {\n\t\treturn nil, fmt.Errorf("error registering workflow: %w", err)\n\t}\n\n\tgo func() {\n\t\tlog.Printf("pushing event")\n\n\t\ttestEvent := userCreateEvent{\n\t\t\tUsername: "echo-test",\n\t\t\tUserID:   "1234",\n\t\t\tData: map[string]string{\n\t\t\t\t"test": "test",\n\t\t\t},\n\t\t}\n\n\t\t// push an event\n\t\terr := c.Event().Push(\n\t\t\tcontext.Background(),\n\t\t\t"user:create:cancellation",\n\t\t\ttestEvent,\n\t\t)\n\t\tif err != nil {\n\t\t\tpanic(fmt.Errorf("error pushing event: %w", err))\n\t\t}\n\n\t\ttime.Sleep(10 * time.Second)\n\n\t\tworkflowName := "cancellation"\n\n\t\tworkflows, err := c.API().WorkflowListWithResponse(context.Background(), uuid.MustParse(c.TenantId()), &rest.WorkflowListParams{\n\t\t\tName: &workflowName,\n\t\t})\n\n\t\tif err != nil {\n\t\t\tpanic(fmt.Errorf("error listing workflows: %w", err))\n\t\t}\n\n\t\tif workflows.JSON200 == nil {\n\t\t\tpanic(fmt.Errorf("no workflows found"))\n\t\t}\n\n\t\trows := *workflows.JSON200.Rows\n\n\t\tif len(rows) == 0 {\n\t\t\tpanic(fmt.Errorf("no workflows found"))\n\t\t}\n\n\t\tworkflowId := uuid.MustParse(rows[0].Metadata.Id)\n\n\t\tworkflowRuns, err := c.API().WorkflowRunListWithResponse(context.Background(), uuid.MustParse(c.TenantId()), &rest.WorkflowRunListParams{\n\t\t\tWorkflowId: &workflowId,\n\t\t})\n\n\t\tif err != nil {\n\t\t\tpanic(fmt.Errorf("error listing workflow runs: %w", err))\n\t\t}\n\n\t\tif workflowRuns.JSON200 == nil {\n\t\t\tpanic(fmt.Errorf("no workflow runs found"))\n\t\t}\n\n\t\tworkflowRunsRows := *workflowRuns.JSON200.Rows\n\n\t\t_, err = c.API().WorkflowRunCancelWithResponse(context.Background(), uuid.MustParse(c.TenantId()), rest.WorkflowRunsCancelRequest{\n\t\t\tWorkflowRunIds: []uuid.UUID{uuid.MustParse(workflowRunsRows[0].Metadata.Id)},\n\t\t})\n\n\t\tif err != nil {\n\t\t\tpanic(fmt.Errorf("error cancelling workflow run: %w", err))\n\t\t}\n\t}()\n\n\tcleanup, err := w.Start()\n\tif err != nil {\n\t\treturn nil, fmt.Errorf("error starting worker: %w", err)\n\t}\n\n\treturn cleanup, nil\n}\n',
      language: 'go',
      source: 'examples/go/z_v0/cancellation/run.go',
      highlights: {},
    },
  L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL2dvL3pfdjAvY29tcHV0ZS9tYWluLmdv:
    {
      content:
        'package main\n\nimport (\n\t"context"\n\t"fmt"\n\t"log"\n\n\t"github.com/joho/godotenv"\n\n\t"github.com/hatchet-dev/hatchet/pkg/client"\n\t"github.com/hatchet-dev/hatchet/pkg/client/compute"\n\t"github.com/hatchet-dev/hatchet/pkg/cmdutils"\n\t"github.com/hatchet-dev/hatchet/pkg/worker"\n)\n\ntype userCreateEvent struct {\n\tUsername string            `json:"username"`\n\tUserID   string            `json:"user_id"`\n\tData     map[string]string `json:"data"`\n}\n\ntype stepOneOutput struct {\n\tMessage string `json:"message"`\n}\n\nfunc main() {\n\terr := godotenv.Load()\n\tif err != nil {\n\t\tpanic(err)\n\t}\n\n\tevents := make(chan string, 50)\n\tinterrupt := cmdutils.InterruptChan()\n\n\tcleanup, err := run(events)\n\tif err != nil {\n\t\tpanic(err)\n\t}\n\n\t<-interrupt\n\n\tif err := cleanup(); err != nil {\n\t\tpanic(fmt.Errorf("error cleaning up: %w", err))\n\t}\n}\n\nfunc run(events chan<- string) (func() error, error) {\n\tc, err := client.New()\n\n\tif err != nil {\n\t\treturn nil, fmt.Errorf("error creating client: %w", err)\n\t}\n\n\tw, err := worker.NewWorker(\n\t\tworker.WithClient(\n\t\t\tc,\n\t\t),\n\t)\n\tif err != nil {\n\t\treturn nil, fmt.Errorf("error creating worker: %w", err)\n\t}\n\n\tpool := "test-pool"\n\tbasicCompute := compute.Compute{\n\t\tPool:        &pool,\n\t\tNumReplicas: 1,\n\t\tCPUs:        1,\n\t\tMemoryMB:    1024,\n\t\tCPUKind:     compute.ComputeKindSharedCPU,\n\t\tRegions:     []compute.Region{compute.Region("ewr")},\n\t}\n\n\tperformancePool := "performance-pool"\n\tperformanceCompute := compute.Compute{\n\t\tPool:        &performancePool,\n\t\tNumReplicas: 1,\n\t\tCPUs:        2,\n\t\tMemoryMB:    1024,\n\t\tCPUKind:     compute.ComputeKindPerformanceCPU,\n\t\tRegions:     []compute.Region{compute.Region("ewr")},\n\t}\n\n\terr = w.RegisterWorkflow(\n\t\t&worker.WorkflowJob{\n\t\t\tOn:          worker.Events("user:create:simple"),\n\t\t\tName:        "simple",\n\t\t\tDescription: "This runs after an update to the user model.",\n\t\t\tConcurrency: worker.Expression("input.user_id"),\n\t\t\tSteps: []*worker.WorkflowStep{\n\t\t\t\tworker.Fn(func(ctx worker.HatchetContext) (result *stepOneOutput, err error) {\n\t\t\t\t\tinput := &userCreateEvent{}\n\n\t\t\t\t\terr = ctx.WorkflowInput(input)\n\n\t\t\t\t\tif err != nil {\n\t\t\t\t\t\treturn nil, err\n\t\t\t\t\t}\n\n\t\t\t\t\tlog.Printf("step-one")\n\t\t\t\t\tevents <- "step-one"\n\n\t\t\t\t\treturn &stepOneOutput{\n\t\t\t\t\t\tMessage: "Username is: " + input.Username,\n\t\t\t\t\t}, nil\n\t\t\t\t},\n\t\t\t\t).SetName("step-one").SetCompute(&basicCompute),\n\t\t\t\tworker.Fn(func(ctx worker.HatchetContext) (result *stepOneOutput, err error) {\n\t\t\t\t\tinput := &stepOneOutput{}\n\t\t\t\t\terr = ctx.StepOutput("step-one", input)\n\n\t\t\t\t\tif err != nil {\n\t\t\t\t\t\treturn nil, err\n\t\t\t\t\t}\n\n\t\t\t\t\tlog.Printf("step-two")\n\t\t\t\t\tevents <- "step-two"\n\n\t\t\t\t\treturn &stepOneOutput{\n\t\t\t\t\t\tMessage: "Above message is: " + input.Message,\n\t\t\t\t\t}, nil\n\t\t\t\t}).SetName("step-two").AddParents("step-one").SetCompute(&performanceCompute),\n\t\t\t},\n\t\t},\n\t)\n\tif err != nil {\n\t\treturn nil, fmt.Errorf("error registering workflow: %w", err)\n\t}\n\n\tgo func() {\n\t\ttestEvent := userCreateEvent{\n\t\t\tUsername: "echo-test",\n\t\t\tUserID:   "1234",\n\t\t\tData: map[string]string{\n\t\t\t\t"test": "test",\n\t\t\t},\n\t\t}\n\n\t\tlog.Printf("pushing event user:create:simple")\n\t\t// push an event\n\t\terr := c.Event().Push(\n\t\t\tcontext.Background(),\n\t\t\t"user:create:simple",\n\t\t\ttestEvent,\n\t\t\tclient.WithEventMetadata(map[string]string{\n\t\t\t\t"hello": "world",\n\t\t\t}),\n\t\t)\n\t\tif err != nil {\n\t\t\tpanic(fmt.Errorf("error pushing event: %w", err))\n\t\t}\n\t}()\n\n\tcleanup, err := w.Start()\n\tif err != nil {\n\t\tpanic(err)\n\t}\n\n\treturn cleanup, nil\n}\n',
      language: 'go',
      source: 'examples/go/z_v0/compute/main.go',
      highlights: {},
    },
  L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL2dvL3pfdjAvY29uY3VycmVuY3kvbWFpbi5nbw__:
    {
      content:
        'package main\n\nimport (\n\t"fmt"\n\t"log"\n\t"time"\n\n\t"github.com/joho/godotenv"\n\n\t"github.com/hatchet-dev/hatchet/pkg/client"\n\t"github.com/hatchet-dev/hatchet/pkg/client/types"\n\t"github.com/hatchet-dev/hatchet/pkg/cmdutils"\n\t"github.com/hatchet-dev/hatchet/pkg/worker"\n)\n\ntype userCreateEvent struct {\n\tUsername string            `json:"username"`\n\tUserID   string            `json:"user_id"`\n\tData     map[string]string `json:"data"`\n}\n\ntype stepOneOutput struct {\n\tMessage string `json:"message"`\n}\n\nfunc main() {\n\terr := godotenv.Load()\n\tif err != nil {\n\t\tpanic(err)\n\t}\n\n\tevents := make(chan string, 50)\n\tinterrupt := cmdutils.InterruptChan()\n\n\tcleanup, err := run(events)\n\tif err != nil {\n\t\tpanic(err)\n\t}\n\n\t<-interrupt\n\n\tif err := cleanup(); err != nil {\n\n\t\tpanic(fmt.Errorf("error cleaning up: %w", err))\n\t}\n}\n\nfunc run(events chan<- string) (func() error, error) {\n\tc, err := client.New()\n\n\tif err != nil {\n\t\treturn nil, fmt.Errorf("error creating client: %w", err)\n\t}\n\n\tw, err := worker.NewWorker(\n\t\tworker.WithClient(\n\t\t\tc,\n\t\t),\n\t)\n\tif err != nil {\n\t\treturn nil, fmt.Errorf("error creating worker: %w", err)\n\t}\n\n\terr = w.RegisterWorkflow(\n\t\t&worker.WorkflowJob{\n\t\t\tOn:          worker.Events("user:create:concurrency"),\n\t\t\tName:        "simple-concurrency",\n\t\t\tDescription: "This runs to test concurrency.",\n\t\t\tConcurrency: worker.Expression("\'concurrency\'").MaxRuns(1).LimitStrategy(types.GroupRoundRobin),\n\t\t\tSteps: []*worker.WorkflowStep{\n\t\t\t\tworker.Fn(func(ctx worker.HatchetContext) (result *stepOneOutput, err error) {\n\t\t\t\t\tinput := &userCreateEvent{}\n\n\t\t\t\t\terr = ctx.WorkflowInput(input)\n\n\t\t\t\t\t// we sleep to simulate a long running task\n\t\t\t\t\ttime.Sleep(10 * time.Second)\n\n\t\t\t\t\tif err != nil {\n\n\t\t\t\t\t\treturn nil, err\n\t\t\t\t\t}\n\n\t\t\t\t\tif ctx.Err() != nil {\n\t\t\t\t\t\treturn nil, ctx.Err()\n\t\t\t\t\t}\n\n\t\t\t\t\tlog.Printf("step-one")\n\t\t\t\t\tevents <- "step-one"\n\n\t\t\t\t\treturn &stepOneOutput{\n\t\t\t\t\t\tMessage: "Username is: " + input.Username,\n\t\t\t\t\t}, nil\n\t\t\t\t},\n\t\t\t\t).SetName("step-one"),\n\t\t\t\tworker.Fn(func(ctx worker.HatchetContext) (result *stepOneOutput, err error) {\n\t\t\t\t\tinput := &stepOneOutput{}\n\t\t\t\t\terr = ctx.StepOutput("step-one", input)\n\n\t\t\t\t\tif err != nil {\n\t\t\t\t\t\treturn nil, err\n\t\t\t\t\t}\n\n\t\t\t\t\tif ctx.Err() != nil {\n\t\t\t\t\t\treturn nil, ctx.Err()\n\t\t\t\t\t}\n\n\t\t\t\t\tlog.Printf("step-two")\n\t\t\t\t\tevents <- "step-two"\n\n\t\t\t\t\treturn &stepOneOutput{\n\t\t\t\t\t\tMessage: "Above message is: " + input.Message,\n\t\t\t\t\t}, nil\n\t\t\t\t}).SetName("step-two").AddParents("step-one"),\n\t\t\t},\n\t\t},\n\t)\n\tif err != nil {\n\t\treturn nil, fmt.Errorf("error registering workflow: %w", err)\n\t}\n\ttestEvent := userCreateEvent{\n\t\tUsername: "echo-test",\n\t\tUserID:   "1234",\n\t\tData: map[string]string{\n\t\t\t"test": "test",\n\t\t},\n\t}\n\tgo func() {\n\t\t// do this 10 times to test concurrency\n\t\tfor i := 0; i < 10; i++ {\n\n\t\t\twfr_id, err := c.Admin().RunWorkflow("simple-concurrency", testEvent)\n\n\t\t\tlog.Println("Starting workflow run id: ", wfr_id)\n\n\t\t\tif err != nil {\n\t\t\t\tpanic(fmt.Errorf("error running workflow: %w", err))\n\t\t\t}\n\n\t\t}\n\t}()\n\n\tcleanup, err := w.Start()\n\tif err != nil {\n\t\tpanic(err)\n\t}\n\n\treturn cleanup, nil\n}\n',
      language: 'go',
      source: 'examples/go/z_v0/concurrency/main.go',
      highlights: {},
    },
  L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL2dvL3pfdjAvY29uY3VycmVuY3kvbWFpbl9lMmVfdGVzdC5nbw__:
    {
      content:
        '//go:build e2e\n\npackage main\n\nimport (\n\t"context"\n\t"testing"\n\t"time"\n\n\t"github.com/stretchr/testify/assert"\n\n\t"github.com/hatchet-dev/hatchet/internal/testutils"\n)\n\nfunc TestConcurrency(t *testing.T) {\n\tt.Skip("skipping concurency test for now")\n\n\ttestutils.Prepare(t)\n\n\tctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)\n\tdefer cancel()\n\n\tevents := make(chan string, 50)\n\n\tcleanup, err := run(events)\n\tif err != nil {\n\t\tt.Fatalf("/run() error = %v", err)\n\t}\n\n\tvar items []string\n\nouter:\n\tfor {\n\t\tselect {\n\t\tcase item := <-events:\n\t\t\titems = append(items, item)\n\t\t\tif len(items) > 2 {\n\t\t\t\tbreak outer\n\t\t\t}\n\t\tcase <-ctx.Done():\n\t\t\tbreak outer\n\t\t}\n\t}\n\n\tassert.Equal(t, []string{\n\t\t"step-one",\n\t\t"step-two",\n\t}, items)\n\n\tif err := cleanup(); err != nil {\n\t\tt.Fatalf("cleanup() error = %v", err)\n\t}\n\n}\n',
      language: 'go',
      source: 'examples/go/z_v0/concurrency/main_e2e_test.go',
      highlights: {},
    },
  L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL2dvL3pfdjAvY3Jvbi9tYWluLmdv:
    {
      content:
        'package main\n\nimport (\n\t"context"\n\t"fmt"\n\n\t"github.com/joho/godotenv"\n\n\t"github.com/hatchet-dev/hatchet/pkg/client"\n\t"github.com/hatchet-dev/hatchet/pkg/cmdutils"\n\t"github.com/hatchet-dev/hatchet/pkg/worker"\n)\n\n// ❓ Workflow Definition Cron Trigger\n// ... normal workflow definition\ntype printOutput struct{}\n\nfunc print(ctx context.Context) (result *printOutput, err error) {\n\tfmt.Println("called print:print")\n\n\treturn &printOutput{}, nil\n}\n\n// ,\nfunc main() {\n\t// ... initialize client and worker\n\terr := godotenv.Load()\n\n\tif err != nil {\n\t\tpanic(err)\n\t}\n\n\tclient, err := client.New()\n\n\tif err != nil {\n\t\tpanic(err)\n\t}\n\n\tw, err := worker.NewWorker(\n\t\tworker.WithClient(\n\t\t\tclient,\n\t\t),\n\t)\n\n\tif err != nil {\n\t\tpanic(err)\n\t}\n\n\t// ,\n\terr = w.RegisterWorkflow(\n\t\t&worker.WorkflowJob{\n\t\t\t// 👀 define the cron expression to run every minute\n\t\t\tOn:          worker.Cron("* * * * *"),\n\t\t\tName:        "cron-workflow",\n\t\t\tDescription: "Demonstrates a simple cron workflow",\n\t\t\tSteps: []*worker.WorkflowStep{\n\t\t\t\tworker.Fn(print),\n\t\t\t},\n\t\t},\n\t)\n\n\tif err != nil {\n\t\tpanic(err)\n\t}\n\n\t// ... start worker\n\n\tinterrupt := cmdutils.InterruptChan()\n\n\tcleanup, err := w.Start()\n\n\tif err != nil {\n\t\tpanic(err)\n\t}\n\n\t<-interrupt\n\n\tif err := cleanup(); err != nil {\n\t\tpanic(fmt.Errorf("error cleaning up: %w", err))\n\t}\n\n\t// ,\n}\n\n// !!\n',
      language: 'go',
      source: 'examples/go/z_v0/cron/main.go',
      highlights: {},
    },
  L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL2dvL3pfdjAvY3Jvbi1wcm9ncmFtbWF0aWMvbWFpbi5nbw__:
    {
      content:
        'package main\n\nimport (\n\t"context"\n\t"fmt"\n\n\t"github.com/joho/godotenv"\n\n\t"github.com/hatchet-dev/hatchet/pkg/client"\n\t"github.com/hatchet-dev/hatchet/pkg/cmdutils"\n\t"github.com/hatchet-dev/hatchet/pkg/worker"\n)\n\n// ❓ Create\n// ... normal workflow definition\ntype printOutput struct{}\n\nfunc print(ctx context.Context) (result *printOutput, err error) {\n\tfmt.Println("called print:print")\n\n\treturn &printOutput{}, nil\n}\n\n// ,\nfunc main() {\n\t// ... initialize client, worker and workflow\n\terr := godotenv.Load()\n\n\tif err != nil {\n\t\tpanic(err)\n\t}\n\n\tc, err := client.New()\n\n\tif err != nil {\n\t\tpanic(err)\n\t}\n\n\tw, err := worker.NewWorker(\n\t\tworker.WithClient(\n\t\t\tc,\n\t\t),\n\t)\n\n\tif err != nil {\n\t\tpanic(err)\n\t}\n\n\terr = w.RegisterWorkflow(\n\t\t&worker.WorkflowJob{\n\t\t\tOn:          worker.NoTrigger(),\n\t\t\tName:        "cron-workflow",\n\t\t\tDescription: "Demonstrates a simple cron workflow",\n\t\t\tSteps: []*worker.WorkflowStep{\n\t\t\t\tworker.Fn(print),\n\t\t\t},\n\t\t},\n\t)\n\n\tif err != nil {\n\t\tpanic(err)\n\t}\n\n\tinterrupt := cmdutils.InterruptChan()\n\n\tcleanup, err := w.Start()\n\n\tif err != nil {\n\t\tpanic(err)\n\t}\n\n\t// ,\n\n\tgo func() {\n\t\t// 👀 define the cron expression to run every minute\n\t\tcron, err := c.Cron().Create(\n\t\t\tcontext.Background(),\n\t\t\t"cron-workflow",\n\t\t\t&client.CronOpts{\n\t\t\t\tName:       "every-minute",\n\t\t\t\tExpression: "* * * * *",\n\t\t\t\tInput: map[string]interface{}{\n\t\t\t\t\t"message": "Hello, world!",\n\t\t\t\t},\n\t\t\t\tAdditionalMetadata: map[string]string{},\n\t\t\t},\n\t\t)\n\n\t\tif err != nil {\n\t\t\tpanic(err)\n\t\t}\n\n\t\tfmt.Println(*cron.Name, cron.Cron)\n\t}()\n\n\t// ... wait for interrupt signal\n\n\t<-interrupt\n\n\tif err := cleanup(); err != nil {\n\t\tpanic(fmt.Errorf("error cleaning up: %w", err))\n\t}\n\n\t// ,\n}\n\n// !!\n\nfunc ListCrons() {\n\n\tc, err := client.New()\n\n\tif err != nil {\n\t\tpanic(err)\n\t}\n\n\t// ❓ List\n\tcrons, err := c.Cron().List(context.Background())\n\t// !!\n\n\tif err != nil {\n\t\tpanic(err)\n\t}\n\n\tfor _, cron := range *crons.Rows {\n\t\tfmt.Println(cron.Cron, *cron.Name)\n\t}\n}\n\nfunc DeleteCron(id string) {\n\tc, err := client.New()\n\n\tif err != nil {\n\t\tpanic(err)\n\t}\n\n\t// ❓ Delete\n\t// 👀 id is the cron\'s metadata id, can get it via cron.Metadata.Id\n\terr = c.Cron().Delete(context.Background(), id)\n\t// !!\n\n\tif err != nil {\n\t\tpanic(err)\n\t}\n\n}\n',
      language: 'go',
      source: 'examples/go/z_v0/cron-programmatic/main.go',
      highlights: {},
    },
  L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL2dvL3pfdjAvZGFnL21haW4uZ28_:
    {
      content:
        'package main\n\nimport (\n\t"context"\n\t"fmt"\n\t"log"\n\t"time"\n\n\t"github.com/joho/godotenv"\n\n\t"github.com/hatchet-dev/hatchet/pkg/client"\n\t"github.com/hatchet-dev/hatchet/pkg/cmdutils"\n\t"github.com/hatchet-dev/hatchet/pkg/worker"\n)\n\ntype userCreateEvent struct {\n\tUsername string            `json:"username"`\n\tUserID   string            `json:"user_id"`\n\tData     map[string]string `json:"data"`\n}\n\ntype stepOutput struct {\n\tMessage string `json:"message"`\n}\n\nfunc main() {\n\terr := godotenv.Load()\n\tif err != nil {\n\t\tpanic(err)\n\t}\n\n\tevents := make(chan string, 50)\n\tif err := run(cmdutils.InterruptChan(), events); err != nil {\n\t\tpanic(err)\n\t}\n}\n\nfunc run(ch <-chan interface{}, events chan<- string) error {\n\tc, err := client.New()\n\n\tif err != nil {\n\t\treturn fmt.Errorf("error creating client: %w", err)\n\t}\n\n\tw, err := worker.NewWorker(\n\t\tworker.WithClient(\n\t\t\tc,\n\t\t),\n\t\tworker.WithMaxRuns(1),\n\t)\n\tif err != nil {\n\t\treturn fmt.Errorf("error creating worker: %w", err)\n\t}\n\n\ttestSvc := w.NewService("test")\n\n\terr = testSvc.On(\n\t\tworker.Events("user:create:simple"),\n\t\t&worker.WorkflowJob{\n\t\t\tName:        "post-user-update",\n\t\t\tDescription: "This runs after an update to the user model.",\n\t\t\tSteps: []*worker.WorkflowStep{\n\t\t\t\tworker.Fn(func(ctx worker.HatchetContext) (result *stepOutput, err error) {\n\t\t\t\t\tinput := &userCreateEvent{}\n\t\t\t\t\tctx.WorkflowInput(input)\n\n\t\t\t\t\ttime.Sleep(1 * time.Second)\n\n\t\t\t\t\treturn &stepOutput{\n\t\t\t\t\t\tMessage: "Step 1 got username: " + input.Username,\n\t\t\t\t\t}, nil\n\t\t\t\t},\n\t\t\t\t).SetName("step-one"),\n\t\t\t\tworker.Fn(func(ctx worker.HatchetContext) (result *stepOutput, err error) {\n\t\t\t\t\tinput := &userCreateEvent{}\n\t\t\t\t\tctx.WorkflowInput(input)\n\n\t\t\t\t\ttime.Sleep(2 * time.Second)\n\n\t\t\t\t\treturn &stepOutput{\n\t\t\t\t\t\tMessage: "Step 2 got username: " + input.Username,\n\t\t\t\t\t}, nil\n\t\t\t\t}).SetName("step-two"),\n\t\t\t\tworker.Fn(func(ctx worker.HatchetContext) (result *stepOutput, err error) {\n\t\t\t\t\tinput := &userCreateEvent{}\n\t\t\t\t\tctx.WorkflowInput(input)\n\n\t\t\t\t\tstep1Out := &stepOutput{}\n\t\t\t\t\tctx.StepOutput("step-one", step1Out)\n\n\t\t\t\t\tstep2Out := &stepOutput{}\n\t\t\t\t\tctx.StepOutput("step-two", step2Out)\n\n\t\t\t\t\ttime.Sleep(3 * time.Second)\n\n\t\t\t\t\treturn &stepOutput{\n\t\t\t\t\t\tMessage: "Username was: " + input.Username + ", Step 3: has parents 1 and 2" + step1Out.Message + ", " + step2Out.Message,\n\t\t\t\t\t}, nil\n\t\t\t\t}).SetName("step-three").AddParents("step-one", "step-two"),\n\t\t\t\tworker.Fn(func(ctx worker.HatchetContext) (result *stepOutput, err error) {\n\t\t\t\t\tstep1Out := &stepOutput{}\n\t\t\t\t\tctx.StepOutput("step-one", step1Out)\n\n\t\t\t\t\tstep3Out := &stepOutput{}\n\t\t\t\t\tctx.StepOutput("step-three", step3Out)\n\n\t\t\t\t\ttime.Sleep(4 * time.Second)\n\n\t\t\t\t\treturn &stepOutput{\n\t\t\t\t\t\tMessage: "Step 4: has parents 1 and 3" + step1Out.Message + ", " + step3Out.Message,\n\t\t\t\t\t}, nil\n\t\t\t\t}).SetName("step-four").AddParents("step-one", "step-three"),\n\t\t\t\tworker.Fn(func(ctx worker.HatchetContext) (result *stepOutput, err error) {\n\t\t\t\t\tstep4Out := &stepOutput{}\n\t\t\t\t\tctx.StepOutput("step-four", step4Out)\n\n\t\t\t\t\ttime.Sleep(5 * time.Second)\n\n\t\t\t\t\treturn &stepOutput{\n\t\t\t\t\t\tMessage: "Step 5: has parent 4" + step4Out.Message,\n\t\t\t\t\t}, nil\n\t\t\t\t}).SetName("step-five").AddParents("step-four"),\n\t\t\t},\n\t\t},\n\t)\n\tif err != nil {\n\t\treturn fmt.Errorf("error registering workflow: %w", err)\n\t}\n\n\tinterruptCtx, cancel := cmdutils.InterruptContextFromChan(ch)\n\tdefer cancel()\n\n\tcleanup, err := w.Start()\n\tif err != nil {\n\t\treturn fmt.Errorf("error starting worker: %w", err)\n\t}\n\n\ttestEvent := userCreateEvent{\n\t\tUsername: "echo-test",\n\t\tUserID:   "1234",\n\t\tData: map[string]string{\n\t\t\t"test": "test",\n\t\t},\n\t}\n\n\tlog.Printf("pushing event user:create:simple")\n\n\t// push an event\n\terr = c.Event().Push(\n\t\tcontext.Background(),\n\t\t"user:create:simple",\n\t\ttestEvent,\n\t)\n\n\tif err != nil {\n\t\treturn fmt.Errorf("error pushing event: %w", err)\n\t}\n\n\tfor {\n\t\tselect {\n\t\tcase <-interruptCtx.Done():\n\t\t\treturn cleanup()\n\t\tdefault:\n\t\t\ttime.Sleep(time.Second)\n\t\t}\n\t}\n}\n',
      language: 'go',
      source: 'examples/go/z_v0/dag/main.go',
      highlights: {},
    },
  L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL2dvL3pfdjAvZGVwcmVjYXRlZC9yZXF1ZXVlL21haW4uZ28_:
    {
      content:
        'package main\n\nimport (\n\t"context"\n\t"fmt"\n\t"time"\n\n\t"github.com/joho/godotenv"\n\n\t"github.com/hatchet-dev/hatchet/pkg/client"\n\t"github.com/hatchet-dev/hatchet/pkg/cmdutils"\n\t"github.com/hatchet-dev/hatchet/pkg/worker"\n)\n\ntype sampleEvent struct{}\n\ntype requeueInput struct{}\n\nfunc main() {\n\terr := godotenv.Load()\n\n\tif err != nil {\n\t\tpanic(err)\n\t}\n\n\tclient, err := client.New(\n\t\tclient.InitWorkflows(),\n\t)\n\n\tif err != nil {\n\t\tpanic(err)\n\t}\n\n\tworker, err := worker.NewWorker(\n\t\tworker.WithClient(\n\t\t\tclient,\n\t\t),\n\t)\n\n\tif err != nil {\n\t\tpanic(err)\n\t}\n\n\terr = worker.RegisterAction("requeue:requeue", func(ctx context.Context, input *requeueInput) (result any, err error) {\n\t\treturn map[string]interface{}{}, nil\n\t})\n\n\tif err != nil {\n\t\tpanic(err)\n\t}\n\n\tinterruptCtx, cancel := cmdutils.InterruptContextFromChan(cmdutils.InterruptChan())\n\tdefer cancel()\n\n\tevent := sampleEvent{}\n\n\t// push an event\n\terr = client.Event().Push(\n\t\tcontext.Background(),\n\t\t"example:event",\n\t\tevent,\n\t)\n\n\tif err != nil {\n\t\tpanic(err)\n\t}\n\n\t// wait to register the worker for 10 seconds, to let the requeuer kick in\n\ttime.Sleep(10 * time.Second)\n\tcleanup, err := worker.Start()\n\tif err != nil {\n\t\tpanic(err)\n\t}\n\n\tfor {\n\t\tselect {\n\t\tcase <-interruptCtx.Done():\n\t\t\tif err := cleanup(); err != nil {\n\t\t\t\tpanic(fmt.Errorf("error cleaning up: %w", err))\n\t\t\t}\n\t\t\treturn\n\t\tdefault:\n\t\t\ttime.Sleep(time.Second)\n\t\t}\n\t}\n}\n',
      language: 'go',
      source: 'examples/go/z_v0/deprecated/requeue/main.go',
      highlights: {},
    },
  L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL2dvL3pfdjAvZGVwcmVjYXRlZC9zY2hlZHVsZS10aW1lb3V0L21haW4uZ28_:
    {
      content:
        'package main\n\nimport (\n\t"context"\n\t"fmt"\n\t"time"\n\n\t"github.com/hatchet-dev/hatchet/pkg/client"\n\t"github.com/joho/godotenv"\n)\n\ntype sampleEvent struct{}\n\ntype timeoutInput struct{}\n\nfunc main() {\n\terr := godotenv.Load()\n\n\tif err != nil {\n\t\tpanic(err)\n\t}\n\n\tclient, err := client.New(\n\t\tclient.InitWorkflows(),\n\t)\n\n\tif err != nil {\n\t\tpanic(err)\n\t}\n\n\tevent := sampleEvent{}\n\n\t// push an event\n\terr = client.Event().Push(\n\t\tcontext.Background(),\n\t\t"user:create",\n\t\tevent,\n\t)\n\n\tif err != nil {\n\t\tpanic(err)\n\t}\n\n\ttime.Sleep(35 * time.Second)\n\n\tfmt.Println("step should have timed out")\n}\n',
      language: 'go',
      source: 'examples/go/z_v0/deprecated/schedule-timeout/main.go',
      highlights: {},
    },
  L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL2dvL3pfdjAvZGVwcmVjYXRlZC90aW1lb3V0L21haW4uZ28_:
    {
      content:
        'package main\n\nimport (\n\t"context"\n\t"fmt"\n\t"time"\n\n\t"github.com/joho/godotenv"\n\n\t"github.com/hatchet-dev/hatchet/pkg/client"\n\t"github.com/hatchet-dev/hatchet/pkg/cmdutils"\n\t"github.com/hatchet-dev/hatchet/pkg/worker"\n)\n\ntype sampleEvent struct{}\n\ntype timeoutInput struct{}\n\nfunc main() {\n\terr := godotenv.Load()\n\n\tif err != nil {\n\t\tpanic(err)\n\t}\n\n\tclient, err := client.New(\n\t\tclient.InitWorkflows(),\n\t)\n\n\tif err != nil {\n\t\tpanic(err)\n\t}\n\n\tworker, err := worker.NewWorker(\n\t\tworker.WithClient(\n\t\t\tclient,\n\t\t),\n\t)\n\n\tif err != nil {\n\t\tpanic(err)\n\t}\n\n\terr = worker.RegisterAction("timeout:timeout", func(ctx context.Context, input *timeoutInput) (result any, err error) {\n\t\t// wait for context done signal\n\t\ttimeStart := time.Now().UTC()\n\t\t<-ctx.Done()\n\t\tfmt.Println("context cancelled in ", time.Since(timeStart).Seconds(), " seconds")\n\n\t\treturn map[string]interface{}{}, nil\n\t})\n\n\tif err != nil {\n\t\tpanic(err)\n\t}\n\n\tinterruptCtx, cancel := cmdutils.InterruptContextFromChan(cmdutils.InterruptChan())\n\tdefer cancel()\n\n\tcleanup, err := worker.Start()\n\tif err != nil {\n\t\tpanic(fmt.Errorf("error starting worker: %w", err))\n\t}\n\n\tevent := sampleEvent{}\n\n\t// push an event\n\terr = client.Event().Push(\n\t\tcontext.Background(),\n\t\t"user:create",\n\t\tevent,\n\t)\n\n\tif err != nil {\n\t\tpanic(err)\n\t}\n\n\tfor {\n\t\tselect {\n\t\tcase <-interruptCtx.Done():\n\t\t\tif err := cleanup(); err != nil {\n\t\t\t\tpanic(fmt.Errorf("error cleaning up: %w", err))\n\t\t\t}\n\t\t\treturn\n\t\tdefault:\n\t\t\ttime.Sleep(time.Second)\n\t\t}\n\t}\n}\n',
      language: 'go',
      source: 'examples/go/z_v0/deprecated/timeout/main.go',
      highlights: {},
    },
  L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL2dvL3pfdjAvZGVwcmVjYXRlZC95YW1sL21haW4uZ28_:
    {
      content:
        'package main\n\nimport (\n\t"context"\n\t"fmt"\n\t"time"\n\n\t"github.com/joho/godotenv"\n\n\t"github.com/hatchet-dev/hatchet/pkg/client"\n\t"github.com/hatchet-dev/hatchet/pkg/cmdutils"\n\t"github.com/hatchet-dev/hatchet/pkg/worker"\n)\n\ntype userCreateEvent struct {\n\tUsername string            `json:"username"`\n\tUserId   string            `json:"user_id"`\n\tData     map[string]string `json:"data"`\n}\n\ntype actionInput struct {\n\tMessage string `json:"message"`\n}\n\ntype actionOut struct {\n\tMessage string `json:"message"`\n}\n\nfunc echo(ctx context.Context, input *actionInput) (result *actionOut, err error) {\n\treturn &actionOut{\n\t\tMessage: input.Message,\n\t}, nil\n}\n\nfunc object(ctx context.Context, input *userCreateEvent) error {\n\treturn nil\n}\n\nfunc main() {\n\terr := godotenv.Load()\n\n\tif err != nil {\n\t\tpanic(err)\n\t}\n\n\tclient, err := client.New(\n\t\tclient.InitWorkflows(),\n\t)\n\n\tif err != nil {\n\t\tpanic(err)\n\t}\n\n\tworker, err := worker.NewWorker(\n\t\tworker.WithClient(\n\t\t\tclient,\n\t\t),\n\t)\n\n\tif err != nil {\n\t\tpanic(err)\n\t}\n\n\techoSvc := worker.NewService("echo")\n\n\terr = echoSvc.RegisterAction(echo)\n\n\tif err != nil {\n\t\tpanic(err)\n\t}\n\n\terr = echoSvc.RegisterAction(object)\n\n\tif err != nil {\n\t\tpanic(err)\n\t}\n\n\tch := cmdutils.InterruptChan()\n\n\tcleanup, err := worker.Start()\n\tif err != nil {\n\t\tpanic(fmt.Errorf("error starting worker: %w", err))\n\t}\n\n\ttestEvent := userCreateEvent{\n\t\tUsername: "echo-test",\n\t\tUserId:   "1234",\n\t\tData: map[string]string{\n\t\t\t"test": "test",\n\t\t},\n\t}\n\n\ttime.Sleep(1 * time.Second)\n\n\t// push an event\n\terr = client.Event().Push(\n\t\tcontext.Background(),\n\t\t"user:create",\n\t\ttestEvent,\n\t)\n\n\tif err != nil {\n\t\tpanic(err)\n\t}\n\n\t<-ch\n\n\tif err := cleanup(); err != nil {\n\t\tpanic(fmt.Errorf("error cleaning up worker: %w", err))\n\t}\n}\n',
      language: 'go',
      source: 'examples/go/z_v0/deprecated/yaml/main.go',
      highlights: {},
    },
  L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL2dvL3pfdjAvZXJyb3JzLXRlc3QvbWFpbi5nbw__:
    {
      content:
        'package main\n\nimport (\n\t"context"\n\t"fmt"\n\t"os"\n\n\t"github.com/joho/godotenv"\n\n\t"github.com/hatchet-dev/hatchet/pkg/client"\n\t"github.com/hatchet-dev/hatchet/pkg/cmdutils"\n\t"github.com/hatchet-dev/hatchet/pkg/errors/sentry"\n\t"github.com/hatchet-dev/hatchet/pkg/worker"\n)\n\ntype userCreateEvent struct {\n\tUsername string            `json:"username"`\n\tUserId   string            `json:"user_id"`\n\tData     map[string]string `json:"data"`\n}\n\ntype stepOneOutput struct {\n\tMessage string `json:"message"`\n}\n\nfunc StepOne(ctx context.Context) (result *stepOneOutput, err error) {\n\treturn nil, fmt.Errorf("this is an error")\n}\n\nfunc main() {\n\terr := godotenv.Load()\n\n\tif err != nil {\n\t\tpanic(err)\n\t}\n\n\tclient, err := client.New()\n\n\tif err != nil {\n\t\tpanic(err)\n\t}\n\n\tsentryAlerter, err := sentry.NewSentryAlerter(&sentry.SentryAlerterOpts{\n\t\tDSN:         os.Getenv("SENTRY_DSN"),\n\t\tEnvironment: os.Getenv("SENTRY_ENVIRONMENT"),\n\t})\n\n\tif err != nil {\n\t\tpanic(err)\n\t}\n\n\tw, err := worker.NewWorker(\n\t\tworker.WithClient(\n\t\t\tclient,\n\t\t),\n\t\tworker.WithErrorAlerter(sentryAlerter),\n\t)\n\n\tif err != nil {\n\t\tpanic(err)\n\t}\n\n\terr = w.On(worker.Event("user:create"), &worker.WorkflowJob{\n\t\tName:        "failing-workflow",\n\t\tDescription: "This is a failing workflow.",\n\t\tSteps: []*worker.WorkflowStep{\n\t\t\t{\n\t\t\t\tFunction: StepOne,\n\t\t\t},\n\t\t},\n\t})\n\n\tif err != nil {\n\t\tpanic(err)\n\t}\n\n\t// err = worker.RegisterAction("echo:echo", func(ctx context.Context, input *actionInput) (result any, err error) {\n\t// \treturn map[string]interface{}{\n\t// \t\t"message": input.Message,\n\t// \t}, nil\n\t// })\n\n\t// if err != nil {\n\t// \tpanic(err)\n\t// }\n\n\t// err = worker.RegisterAction("echo:object", func(ctx context.Context, input *actionInput) (result any, err error) {\n\t// \treturn nil, nil\n\t// })\n\n\t// if err != nil {\n\t// \tpanic(err)\n\t// }\n\n\tch := cmdutils.InterruptChan()\n\n\tcleanup, err := w.Start()\n\tif err != nil {\n\t\tpanic(fmt.Errorf("error starting worker: %w", err))\n\t}\n\n\ttestEvent := userCreateEvent{\n\t\tUsername: "echo-test",\n\t\tUserId:   "1234",\n\t\tData: map[string]string{\n\t\t\t"test": "test",\n\t\t},\n\t}\n\n\t// push an event\n\terr = client.Event().Push(\n\t\tcontext.Background(),\n\t\t"user:create",\n\t\ttestEvent,\n\t)\n\n\tif err != nil {\n\t\tpanic(err)\n\t}\n\n\t<-ch\n\n\tif err := cleanup(); err != nil {\n\t\tpanic(fmt.Errorf("error cleaning up: %w", err))\n\t}\n}\n',
      language: 'go',
      source: 'examples/go/z_v0/errors-test/main.go',
      highlights: {},
    },
  L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL2dvL3pfdjAvbGltaXQtY29uY3VycmVuY3kvY2FuY2VsLWluLXByb2dyZXNzL21haW4uZ28_:
    {
      content:
        'package main\n\nimport (\n\t"context"\n\t"fmt"\n\t"time"\n\n\t"github.com/joho/godotenv"\n\n\t"github.com/hatchet-dev/hatchet/pkg/client"\n\t"github.com/hatchet-dev/hatchet/pkg/cmdutils"\n\t"github.com/hatchet-dev/hatchet/pkg/worker"\n)\n\ntype concurrencyLimitEvent struct {\n\tIndex int `json:"index"`\n}\n\ntype stepOneOutput struct {\n\tMessage string `json:"message"`\n}\n\nfunc main() {\n\terr := godotenv.Load()\n\tif err != nil {\n\t\tpanic(err)\n\t}\n\n\tevents := make(chan string, 50)\n\tif err := run(cmdutils.InterruptChan(), events); err != nil {\n\t\tpanic(err)\n\t}\n}\n\nfunc getConcurrencyKey(ctx worker.HatchetContext) (string, error) {\n\treturn "user-create", nil\n}\n\nfunc run(ch <-chan interface{}, events chan<- string) error {\n\tc, err := client.New()\n\n\tif err != nil {\n\t\treturn fmt.Errorf("error creating client: %w", err)\n\t}\n\n\tw, err := worker.NewWorker(\n\t\tworker.WithClient(\n\t\t\tc,\n\t\t),\n\t)\n\tif err != nil {\n\t\treturn fmt.Errorf("error creating worker: %w", err)\n\t}\n\n\ttestSvc := w.NewService("test")\n\n\terr = testSvc.On(\n\t\tworker.Events("concurrency-test-event"),\n\t\t&worker.WorkflowJob{\n\t\t\tName:        "concurrency-limit",\n\t\t\tDescription: "This limits concurrency to 1 run at a time.",\n\t\t\tConcurrency: worker.Concurrency(getConcurrencyKey).MaxRuns(1),\n\t\t\tSteps: []*worker.WorkflowStep{\n\t\t\t\tworker.Fn(func(ctx worker.HatchetContext) (result *stepOneOutput, err error) {\n\t\t\t\t\t<-ctx.Done()\n\t\t\t\t\tfmt.Println("context done, returning")\n\t\t\t\t\treturn nil, nil\n\t\t\t\t},\n\t\t\t\t).SetName("step-one"),\n\t\t\t},\n\t\t},\n\t)\n\tif err != nil {\n\t\treturn fmt.Errorf("error registering workflow: %w", err)\n\t}\n\n\tinterruptCtx, cancel := cmdutils.InterruptContextFromChan(ch)\n\tdefer cancel()\n\n\tcleanup, err := w.Start()\n\tif err != nil {\n\t\treturn fmt.Errorf("error starting worker: %w", err)\n\t}\n\n\tgo func() {\n\t\t// sleep with interrupt context\n\t\tselect {\n\t\tcase <-interruptCtx.Done(): // context cancelled\n\t\t\tfmt.Println("interrupted")\n\t\t\tif err := cleanup(); err != nil {\n\t\t\t\tpanic(fmt.Errorf("error cleaning up: %w", err))\n\t\t\t}\n\t\t\treturn\n\t\tcase <-time.After(2 * time.Second): // timeout\n\t\t}\n\n\t\tfirstEvent := concurrencyLimitEvent{\n\t\t\tIndex: 0,\n\t\t}\n\n\t\t// push an event\n\t\terr = c.Event().Push(\n\t\t\tcontext.Background(),\n\t\t\t"concurrency-test-event",\n\t\t\tfirstEvent,\n\t\t)\n\n\t\tif err != nil {\n\t\t\tpanic(err)\n\t\t}\n\n\t\tselect {\n\t\tcase <-interruptCtx.Done(): // context cancelled\n\t\t\tfmt.Println("interrupted")\n\t\t\treturn\n\t\tcase <-time.After(10 * time.Second): // timeout\n\t\t}\n\n\t\t// push a second event\n\t\terr = c.Event().Push(\n\t\t\tcontext.Background(),\n\t\t\t"concurrency-test-event",\n\t\t\tconcurrencyLimitEvent{\n\t\t\t\tIndex: 1,\n\t\t\t},\n\t\t)\n\n\t\tif err != nil {\n\t\t\tpanic(err)\n\t\t}\n\t}()\n\n\tfor {\n\t\tselect {\n\t\tcase <-interruptCtx.Done():\n\t\t\treturn nil\n\t\tdefault:\n\t\t\ttime.Sleep(time.Second)\n\t\t}\n\t}\n}\n',
      language: 'go',
      source: 'examples/go/z_v0/limit-concurrency/cancel-in-progress/main.go',
      highlights: {},
    },
  L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL2dvL3pfdjAvbGltaXQtY29uY3VycmVuY3kvZ3JvdXAtcm91bmQtcm9iaW4vbWFpbi5nbw__:
    {
      content:
        'package main\n\nimport (\n\t"context"\n\t"fmt"\n\t"time"\n\n\t"github.com/joho/godotenv"\n\n\t"github.com/hatchet-dev/hatchet/pkg/client"\n\t"github.com/hatchet-dev/hatchet/pkg/client/types"\n\t"github.com/hatchet-dev/hatchet/pkg/cmdutils"\n\t"github.com/hatchet-dev/hatchet/pkg/worker"\n)\n\ntype concurrencyLimitEvent struct {\n\tUserId int `json:"user_id"`\n}\n\ntype stepOneOutput struct {\n\tMessage string `json:"message"`\n}\n\nfunc main() {\n\terr := godotenv.Load()\n\tif err != nil {\n\t\tpanic(err)\n\t}\n\n\tevents := make(chan string, 50)\n\tif err := run(cmdutils.InterruptChan(), events); err != nil {\n\t\tpanic(err)\n\t}\n}\n\nfunc getConcurrencyKey(ctx worker.HatchetContext) (string, error) {\n\tinput := &concurrencyLimitEvent{}\n\terr := ctx.WorkflowInput(input)\n\n\tif err != nil {\n\t\treturn "", fmt.Errorf("error getting input: %w", err)\n\t}\n\n\treturn fmt.Sprintf("%d", input.UserId), nil\n}\n\nfunc run(ch <-chan interface{}, events chan<- string) error {\n\tc, err := client.New()\n\n\tif err != nil {\n\t\treturn fmt.Errorf("error creating client: %w", err)\n\t}\n\n\tw, err := worker.NewWorker(\n\t\tworker.WithClient(\n\t\t\tc,\n\t\t),\n\t)\n\tif err != nil {\n\t\treturn fmt.Errorf("error creating worker: %w", err)\n\t}\n\n\ttestSvc := w.NewService("test")\n\n\terr = testSvc.On(\n\t\tworker.Events("concurrency-test-event-rr"),\n\t\t&worker.WorkflowJob{\n\t\t\tName:        "concurrency-limit-round-robin",\n\t\t\tDescription: "This limits concurrency to 2 runs at a time.",\n\t\t\tConcurrency: worker.Concurrency(getConcurrencyKey).MaxRuns(2).LimitStrategy(types.GroupRoundRobin),\n\t\t\tSteps: []*worker.WorkflowStep{\n\t\t\t\tworker.Fn(func(ctx worker.HatchetContext) (result *stepOneOutput, err error) {\n\t\t\t\t\tinput := &concurrencyLimitEvent{}\n\n\t\t\t\t\terr = ctx.WorkflowInput(input)\n\n\t\t\t\t\tif err != nil {\n\t\t\t\t\t\treturn nil, fmt.Errorf("error getting input: %w", err)\n\t\t\t\t\t}\n\n\t\t\t\t\tfmt.Println("received event", input.UserId)\n\n\t\t\t\t\ttime.Sleep(5 * time.Second)\n\n\t\t\t\t\tfmt.Println("processed event", input.UserId)\n\n\t\t\t\t\treturn nil, nil\n\t\t\t\t},\n\t\t\t\t).SetName("step-one"),\n\t\t\t},\n\t\t},\n\t)\n\tif err != nil {\n\t\treturn fmt.Errorf("error registering workflow: %w", err)\n\t}\n\n\tinterruptCtx, cancel := cmdutils.InterruptContextFromChan(ch)\n\tdefer cancel()\n\n\tcleanup, err := w.Start()\n\tif err != nil {\n\t\treturn fmt.Errorf("error starting worker: %w", err)\n\t}\n\n\tgo func() {\n\t\t// sleep with interrupt context\n\t\tselect {\n\t\tcase <-interruptCtx.Done(): // context cancelled\n\t\t\tfmt.Println("interrupted")\n\t\t\treturn\n\t\tcase <-time.After(2 * time.Second): // timeout\n\t\t}\n\n\t\tfor i := 0; i < 20; i++ {\n\t\t\tvar event concurrencyLimitEvent\n\n\t\t\tif i < 10 {\n\t\t\t\tevent = concurrencyLimitEvent{0}\n\t\t\t} else {\n\t\t\t\tevent = concurrencyLimitEvent{1}\n\t\t\t}\n\n\t\t\tc.Event().Push(context.Background(), "concurrency-test-event-rr", event)\n\t\t}\n\n\t\tselect {\n\t\tcase <-interruptCtx.Done(): // context cancelled\n\t\t\tfmt.Println("interrupted")\n\t\t\treturn\n\t\tcase <-time.After(10 * time.Second): //timeout\n\t\t}\n\t}()\n\n\tfor {\n\t\tselect {\n\t\tcase <-interruptCtx.Done():\n\t\t\tif err := cleanup(); err != nil {\n\t\t\t\treturn fmt.Errorf("error cleaning up: %w", err)\n\t\t\t}\n\t\t\treturn nil\n\t\tdefault:\n\t\t\ttime.Sleep(time.Second)\n\t\t}\n\t}\n}\n',
      language: 'go',
      source: 'examples/go/z_v0/limit-concurrency/group-round-robin/main.go',
      highlights: {},
    },
  L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL2dvL3pfdjAvbGltaXQtY29uY3VycmVuY3kvZ3JvdXAtcm91bmQtcm9iaW4tYWR2YW5jZWQvbWFpbi5nbw__:
    {
      content:
        'package main\n\nimport (\n\t"context"\n\t"fmt"\n\t"sync"\n\t"time"\n\n\t"github.com/joho/godotenv"\n\n\t"github.com/hatchet-dev/hatchet/pkg/client"\n\t"github.com/hatchet-dev/hatchet/pkg/client/types"\n\t"github.com/hatchet-dev/hatchet/pkg/cmdutils"\n\t"github.com/hatchet-dev/hatchet/pkg/worker"\n)\n\ntype concurrencyLimitEvent struct {\n\tConcurrencyKey string `json:"concurrency_key"`\n\tUserId         int    `json:"user_id"`\n}\n\ntype stepOneOutput struct {\n\tMessage                 string `json:"message"`\n\tConcurrencyWhenFinished int    `json:"concurrency_when_finished"`\n}\n\nfunc main() {\n\terr := godotenv.Load()\n\tif err != nil {\n\t\tpanic(err)\n\t}\n\n\tctx, cancel := cmdutils.NewInterruptContext()\n\tdefer cancel()\n\n\tif err := run(ctx); err != nil {\n\t\tpanic(err)\n\t}\n}\n\nfunc getConcurrencyKey(ctx worker.HatchetContext) (string, error) {\n\treturn "concurrency", nil\n}\n\nvar done = make(chan struct{})\nvar errChan = make(chan error)\n\nvar workflowCount int\nvar countMux sync.Mutex\n\nfunc run(ctx context.Context) error {\n\tc, err := client.New()\n\n\tif err != nil {\n\t\treturn fmt.Errorf("error creating client: %w", err)\n\t}\n\n\tw, err := worker.NewWorker(\n\t\tworker.WithClient(\n\t\t\tc,\n\t\t),\n\t)\n\tif err != nil {\n\t\treturn fmt.Errorf("error creating worker: %w", err)\n\t}\n\n\t// runningCount := 0\n\n\tcountMux := sync.Mutex{}\n\n\tvar countMap = make(map[string]int)\n\tmaxConcurrent := 2\n\n\terr = w.RegisterWorkflow(\n\n\t\t&worker.WorkflowJob{\n\t\t\tName:        "concurrency-limit-round-robin-existing-workflows",\n\t\t\tDescription: "This limits concurrency to maxConcurrent runs at a time.",\n\t\t\tOn:          worker.Events("test:concurrency-limit-round-robin-existing-workflows"),\n\t\t\tConcurrency: worker.Expression("input.concurrency_key").MaxRuns(int32(maxConcurrent)).LimitStrategy(types.GroupRoundRobin),\n\n\t\t\tSteps: []*worker.WorkflowStep{\n\t\t\t\tworker.Fn(func(ctx worker.HatchetContext) (result *stepOneOutput, err error) {\n\t\t\t\t\tinput := &concurrencyLimitEvent{}\n\n\t\t\t\t\terr = ctx.WorkflowInput(input)\n\n\t\t\t\t\tif err != nil {\n\t\t\t\t\t\treturn nil, fmt.Errorf("error getting input: %w", err)\n\t\t\t\t\t}\n\t\t\t\t\tconcurrencyKey := input.ConcurrencyKey\n\t\t\t\t\tcountMux.Lock()\n\n\t\t\t\t\tif countMap[concurrencyKey]+1 > maxConcurrent {\n\t\t\t\t\t\tcountMux.Unlock()\n\t\t\t\t\t\te := fmt.Errorf("concurrency limit exceeded for %d we have %d workers running", input.UserId, countMap[concurrencyKey])\n\t\t\t\t\t\terrChan <- e\n\t\t\t\t\t\treturn nil, e\n\t\t\t\t\t}\n\t\t\t\t\tcountMap[concurrencyKey]++\n\n\t\t\t\t\tcountMux.Unlock()\n\n\t\t\t\t\tfmt.Println("received event", input.UserId)\n\n\t\t\t\t\ttime.Sleep(10 * time.Second)\n\n\t\t\t\t\tfmt.Println("processed event", input.UserId)\n\n\t\t\t\t\tcountMux.Lock()\n\t\t\t\t\tcountMap[concurrencyKey]--\n\t\t\t\t\tcountMux.Unlock()\n\n\t\t\t\t\tdone <- struct{}{}\n\n\t\t\t\t\treturn &stepOneOutput{}, nil\n\t\t\t\t},\n\t\t\t\t).SetName("step-one"),\n\t\t\t},\n\t\t},\n\t)\n\tif err != nil {\n\t\treturn fmt.Errorf("error registering workflow: %w", err)\n\t}\n\n\tgo func() {\n\t\tvar workflowRuns []*client.WorkflowRun\n\n\t\tfor i := 0; i < 1; i++ {\n\t\t\tworkflowCount++\n\t\t\tevent := concurrencyLimitEvent{\n\t\t\t\tConcurrencyKey: "key",\n\t\t\t\tUserId:         i,\n\t\t\t}\n\t\t\tworkflowRuns = append(workflowRuns, &client.WorkflowRun{\n\t\t\t\tName:  "concurrency-limit-round-robin-existing-workflows",\n\t\t\t\tInput: event,\n\t\t\t})\n\n\t\t}\n\n\t\t// create a second one with a different key\n\n\t\t// so the bug we are testing here is that total concurrency for any one group should be 2\n\t\t// but if we have more than one group we end up with 4 running when only 2 + 1 are eligible to run\n\n\t\tfor i := 0; i < 3; i++ {\n\t\t\tworkflowCount++\n\n\t\t\tevent := concurrencyLimitEvent{\n\t\t\t\tConcurrencyKey: "secondKey",\n\t\t\t\tUserId:         i,\n\t\t\t}\n\t\t\tworkflowRuns = append(workflowRuns, &client.WorkflowRun{\n\t\t\t\tName:  "concurrency-limit-round-robin-existing-workflows",\n\t\t\t\tInput: event,\n\t\t\t})\n\n\t\t}\n\n\t\t_, err := c.Admin().BulkRunWorkflow(workflowRuns)\n\t\tif err != nil {\n\t\t\tfmt.Println("error running workflow", err)\n\t\t}\n\n\t\tfmt.Println("ran workflows")\n\n\t}()\n\n\ttime.Sleep(2 * time.Second)\n\tcleanup, err := w.Start()\n\tif err != nil {\n\t\treturn fmt.Errorf("error starting worker: %w", err)\n\t}\n\tdefer cleanup()\n\n\tfor {\n\t\tselect {\n\t\tcase <-ctx.Done():\n\t\t\treturn nil\n\t\tcase <-time.After(20 * time.Second):\n\t\t\treturn fmt.Errorf("timeout")\n\t\tcase err := <-errChan:\n\t\t\treturn err\n\t\tcase <-done:\n\t\t\tcountMux.Lock()\n\t\t\tworkflowCount--\n\t\t\tcountMux.Unlock()\n\t\t\tif workflowCount == 0 {\n\t\t\t\ttime.Sleep(1 * time.Second)\n\t\t\t\treturn nil\n\t\t\t}\n\n\t\t}\n\t}\n}\n',
      language: 'go',
      source:
        'examples/go/z_v0/limit-concurrency/group-round-robin-advanced/main.go',
      highlights: {},
    },
  L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL2dvL3pfdjAvbGltaXQtY29uY3VycmVuY3kvZ3JvdXAtcm91bmQtcm9iaW4tYWR2YW5jZWQvbWFpbl9lMmVfdGVzdC5nbw__:
    {
      content:
        '//go:build e2e\n\npackage main\n\nimport (\n\t"context"\n\t"os"\n\t"os/signal"\n\t"testing"\n\t"time"\n\n\t"github.com/hatchet-dev/hatchet/internal/testutils"\n)\n\nfunc TestAdvancedConcurrency(t *testing.T) {\n\tt.Skip("skipping concurency test for now")\n\ttestutils.Prepare(t)\n\n\tctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)\n\tdefer cancel()\n\n\tch := make(chan os.Signal, 1)\n\tsignal.Notify(ch, os.Interrupt)\n\tgo func() {\n\t\t<-ctx.Done()\n\t\tch <- os.Interrupt\n\t}()\n\n\terr := run(ctx)\n\n\tif err != nil {\n\t\tt.Fatalf("/run() error = %v", err)\n\t}\n\n}\n',
      language: 'go',
      source:
        'examples/go/z_v0/limit-concurrency/group-round-robin-advanced/main_e2e_test.go',
      highlights: {},
    },
  L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL2dvL3pfdjAvbG9hZHRlc3QvY2xpL2NsaV9lMmVfdGVzdC5nbw__:
    {
      content:
        '//go:build load\n\npackage main\n\nimport (\n\t"context"\n\t"log"\n\t"sync"\n\t"testing"\n\t"time"\n\n\t"go.uber.org/goleak"\n\n\t"github.com/hatchet-dev/hatchet/internal/testutils"\n\t"github.com/hatchet-dev/hatchet/pkg/config/shared"\n\t"github.com/hatchet-dev/hatchet/pkg/logger"\n)\n\nfunc TestLoadCLI(t *testing.T) {\n\ttestutils.Prepare(t)\n\n\ttype args struct {\n\t\tduration        time.Duration\n\t\teventsPerSecond int\n\t\tdelay           time.Duration\n\t\twait            time.Duration\n\t\tworkerDelay     time.Duration\n\t\tconcurrency     int\n\t}\n\n\tl = logger.NewStdErr(\n\t\t&shared.LoggerConfigFile{\n\t\t\tLevel:  "warn",\n\t\t\tFormat: "console",\n\t\t},\n\t\t"loadtest",\n\t)\n\n\ttests := []struct {\n\t\tname    string\n\t\targs    args\n\t\twantErr bool\n\t}{{\n\t\tname: "test with high step delay",\n\t\targs: args{\n\t\t\tduration:        10 * time.Second,\n\t\t\teventsPerSecond: 10,\n\t\t\tdelay:           10 * time.Second,\n\t\t\twait:            60 * time.Second,\n\t\t\tconcurrency:     0,\n\t\t},\n\t}, {\n\t\tname: "test simple with unlimited concurrency",\n\t\targs: args{\n\t\t\tduration:        10 * time.Second,\n\t\t\teventsPerSecond: 10,\n\t\t\tdelay:           0 * time.Second,\n\t\t\twait:            60 * time.Second,\n\t\t\tconcurrency:     0,\n\t\t},\n\t}, {\n\t\tname: "test for many queued events and little worker throughput",\n\t\targs: args{\n\t\t\tduration:        60 * time.Second,\n\t\t\teventsPerSecond: 100,\n\t\t\tdelay:           0 * time.Second,\n\t\t\tworkerDelay:     60 * time.Second,\n\t\t\twait:            240 * time.Second,\n\t\t\tconcurrency:     0,\n\t\t},\n\t}}\n\n\tctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)\n\n\tsetup := sync.WaitGroup{}\n\n\tgo func() {\n\t\tsetup.Add(1)\n\t\tlog.Printf("setup start")\n\t\ttestutils.SetupEngine(ctx, t)\n\t\tsetup.Done()\n\t\tlog.Printf("setup end")\n\t}()\n\n\t// TODO instead of waiting, figure out when the engine setup is complete\n\ttime.Sleep(15 * time.Second)\n\n\tfor _, tt := range tests {\n\t\tt.Run(tt.name, func(t *testing.T) {\n\t\t\tif err := do(tt.args.duration, tt.args.eventsPerSecond, tt.args.delay, tt.args.wait, tt.args.concurrency, tt.args.workerDelay, 100, 0.0, "0kb", 1); (err != nil) != tt.wantErr {\n\t\t\t\tt.Errorf("do() error = %v, wantErr %v", err, tt.wantErr)\n\t\t\t}\n\t\t})\n\t}\n\n\tcancel()\n\n\tlog.Printf("test complete")\n\tsetup.Wait()\n\tlog.Printf("cleanup complete")\n\n\tgoleak.VerifyNone(\n\t\tt,\n\t\t// worker\n\t\tgoleak.IgnoreTopFunction("go.opencensus.io/stats/view.(*worker).start"),\n\t\tgoleak.IgnoreTopFunction("google.golang.org/grpc/internal/grpcsync.(*CallbackSerializer).run"),\n\t\tgoleak.IgnoreTopFunction("internal/poll.runtime_pollWait"),\n\t\tgoleak.IgnoreTopFunction("google.golang.org/grpc/internal/transport.(*controlBuffer).get"),\n\t\t// all engine related packages\n\t\tgoleak.IgnoreTopFunction("github.com/jackc/pgx/v5/pgxpool.(*Pool).backgroundHealthCheck"),\n\t\tgoleak.IgnoreTopFunction("github.com/rabbitmq/amqp091-go.(*Connection).heartbeater"),\n\t\tgoleak.IgnoreTopFunction("github.com/rabbitmq/amqp091-go.(*consumers).buffer"),\n\t\tgoleak.IgnoreTopFunction("google.golang.org/grpc/internal/transport.(*http2Server).keepalive"),\n\t)\n}\n',
      language: 'go',
      source: 'examples/go/z_v0/loadtest/cli/cli_e2e_test.go',
      highlights: {},
    },
  L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL2dvL3pfdjAvbG9hZHRlc3QvY2xpL2RvLmdv:
    {
      content:
        'package main\n\nimport (\n\t"context"\n\t"fmt"\n\t"log"\n\t"time"\n)\n\ntype avgResult struct {\n\tcount int64\n\tavg   time.Duration\n}\n\nfunc do(duration time.Duration, eventsPerSecond int, delay time.Duration, wait time.Duration, concurrency int, workerDelay time.Duration, slots int, failureRate float32, payloadSize string, eventFanout int) error {\n\tl.Info().Msgf("testing with duration=%s, eventsPerSecond=%d, delay=%s, wait=%s, concurrency=%d", duration, eventsPerSecond, delay, wait, concurrency)\n\n\tctx, cancel := context.WithCancel(context.Background())\n\tdefer cancel()\n\n\tafter := 10 * time.Second\n\n\tgo func() {\n\t\ttime.Sleep(duration + after + wait + 5*time.Second)\n\t\tcancel()\n\t}()\n\n\tch := make(chan int64, 2)\n\tdurations := make(chan time.Duration, eventsPerSecond)\n\n\t// Compute running average for executed durations using a rolling average.\n\tdurationsResult := make(chan avgResult)\n\tgo func() {\n\t\tvar count int64\n\t\tvar avg time.Duration\n\t\tfor d := range durations {\n\t\t\tcount++\n\t\t\tif count == 1 {\n\t\t\t\tavg = d\n\t\t\t} else {\n\t\t\t\tavg = avg + (d-avg)/time.Duration(count)\n\t\t\t}\n\t\t}\n\t\tdurationsResult <- avgResult{count: count, avg: avg}\n\t}()\n\n\tgo func() {\n\t\tif workerDelay > 0 {\n\t\t\tl.Info().Msgf("wait %s before starting the worker", workerDelay)\n\t\t\ttime.Sleep(workerDelay)\n\t\t}\n\t\tl.Info().Msg("starting worker now")\n\t\tcount, uniques := run(ctx, delay, durations, concurrency, slots, failureRate, eventFanout)\n\t\tclose(durations)\n\t\tch <- count\n\t\tch <- uniques\n\t}()\n\n\ttime.Sleep(after)\n\n\tscheduled := make(chan time.Duration, eventsPerSecond)\n\n\t// Compute running average for scheduled times using a rolling average.\n\tscheduledResult := make(chan avgResult)\n\tgo func() {\n\t\tvar count int64\n\t\tvar avg time.Duration\n\t\tfor d := range scheduled {\n\t\t\tcount++\n\t\t\tif count == 1 {\n\t\t\t\tavg = d\n\t\t\t} else {\n\t\t\t\tavg = avg + (d-avg)/time.Duration(count)\n\t\t\t}\n\t\t}\n\t\tscheduledResult <- avgResult{count: count, avg: avg}\n\t}()\n\n\temitted := emit(ctx, eventsPerSecond, duration, scheduled, payloadSize)\n\tclose(scheduled)\n\n\texecuted := <-ch\n\tuniques := <-ch\n\n\tfinalDurationResult := <-durationsResult\n\tfinalScheduledResult := <-scheduledResult\n\n\tlog.Printf("ℹ️ emitted %d, executed %d, uniques %d, using %d events/s", emitted, executed, uniques, eventsPerSecond)\n\n\tif executed == 0 {\n\t\treturn fmt.Errorf("❌ no events executed")\n\t}\n\n\tlog.Printf("ℹ️ final average duration per executed event: %s", finalDurationResult.avg)\n\tlog.Printf("ℹ️ final average scheduling time per event: %s", finalScheduledResult.avg)\n\n\tif int64(eventFanout)*emitted != executed {\n\t\tlog.Printf("⚠️ warning: emitted and executed counts do not match: %d != %d", int64(eventFanout)*emitted, executed)\n\t}\n\n\tif int64(eventFanout)*emitted != uniques {\n\t\treturn fmt.Errorf("❌ emitted and unique executed counts do not match: %d != %d", int64(eventFanout)*emitted, uniques)\n\t}\n\n\tlog.Printf("✅ success")\n\n\treturn nil\n}\n',
      language: 'go',
      source: 'examples/go/z_v0/loadtest/cli/do.go',
      highlights: {},
    },
  L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL2dvL3pfdjAvbG9hZHRlc3QvY2xpL2VtaXQuZ28_:
    {
      content:
        'package main\n\nimport (\n\t"context"\n\t"fmt"\n\t"strconv"\n\t"strings"\n\t"sync"\n\t"sync/atomic"\n\t"time"\n\n\t"github.com/hatchet-dev/hatchet/pkg/client"\n)\n\ntype Event struct {\n\tID        int64     `json:"id"`\n\tCreatedAt time.Time `json:"created_at"`\n\tPayload   string    `json:"payload"`\n}\n\nfunc parseSize(s string) int {\n\ts = strings.ToLower(strings.TrimSpace(s))\n\tvar multiplier int\n\tif strings.HasSuffix(s, "kb") {\n\t\tmultiplier = 1024\n\t\ts = strings.TrimSuffix(s, "kb")\n\t} else if strings.HasSuffix(s, "mb") {\n\t\tmultiplier = 1024 * 1024\n\t\ts = strings.TrimSuffix(s, "mb")\n\t} else {\n\t\tmultiplier = 1\n\t}\n\tnum, err := strconv.Atoi(strings.TrimSpace(s))\n\tif err != nil {\n\t\tpanic(fmt.Errorf("invalid size argument: %w", err))\n\t}\n\treturn num * multiplier\n}\n\nfunc emit(ctx context.Context, amountPerSecond int, duration time.Duration, scheduled chan<- time.Duration, payloadArg string) int64 {\n\tc, err := client.New()\n\tif err != nil {\n\t\tpanic(err)\n\t}\n\n\tvar id int64\n\n\t// Precompute payload data.\n\tpayloadSize := parseSize(payloadArg)\n\tpayloadData := strings.Repeat("a", payloadSize)\n\n\t// Create a buffered channel for events.\n\tjobCh := make(chan Event, amountPerSecond*2)\n\n\t// Worker pool to handle event pushes.\n\tnumWorkers := 10\n\tvar wg sync.WaitGroup\n\tfor i := 0; i < numWorkers; i++ {\n\t\twg.Add(1)\n\t\tgo func() {\n\t\t\tdefer wg.Done()\n\t\t\tfor ev := range jobCh {\n\t\t\t\tl.Info().Msgf("pushing event %d", ev.ID)\n\t\t\t\terr := c.Event().Push(context.Background(), "load-test:event", ev, client.WithEventMetadata(map[string]string{\n\t\t\t\t\t"event_id": fmt.Sprintf("%d", ev.ID),\n\t\t\t\t}))\n\t\t\t\tif err != nil {\n\t\t\t\t\tpanic(fmt.Errorf("error pushing event: %w", err))\n\t\t\t\t}\n\t\t\t\ttook := time.Since(ev.CreatedAt)\n\t\t\t\tl.Info().Msgf("pushed event %d took %s", ev.ID, took)\n\t\t\t\tscheduled <- took\n\t\t\t}\n\t\t}()\n\t}\n\n\tticker := time.NewTicker(time.Second / time.Duration(amountPerSecond))\n\tdefer ticker.Stop()\n\ttimer := time.NewTimer(duration)\n\tdefer timer.Stop()\n\nloop:\n\tfor {\n\t\tselect {\n\t\tcase <-ctx.Done():\n\t\t\tl.Info().Msg("done emitting events due to interruption")\n\t\t\tbreak loop\n\t\tcase <-timer.C:\n\t\t\tl.Info().Msg("done emitting events due to timer")\n\t\t\tbreak loop\n\t\tcase <-ticker.C:\n\t\t\tnewID := atomic.AddInt64(&id, 1)\n\t\t\tev := Event{\n\t\t\t\tID:        newID,\n\t\t\t\tCreatedAt: time.Now(),\n\t\t\t\tPayload:   payloadData,\n\t\t\t}\n\t\t\tselect {\n\t\t\tcase jobCh <- ev:\n\t\t\tcase <-ctx.Done():\n\t\t\t\tbreak loop\n\t\t\t}\n\t\t}\n\t}\n\n\tclose(jobCh)\n\twg.Wait()\n\treturn id\n}\n',
      language: 'go',
      source: 'examples/go/z_v0/loadtest/cli/emit.go',
      highlights: {},
    },
  L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL2dvL3pfdjAvbG9hZHRlc3QvY2xpL21haW4uZ28_:
    {
      content:
        'package main\n\nimport (\n\t"log"\n\t"os"\n\t"time"\n\n\t"github.com/rs/zerolog"\n\t"github.com/spf13/cobra"\n\n\t"github.com/hatchet-dev/hatchet/pkg/config/shared"\n\t"github.com/hatchet-dev/hatchet/pkg/logger"\n\n\t"net/http"\n\t_ "net/http/pprof"\n)\n\nvar l zerolog.Logger\n\nfunc main() {\n\tvar events int\n\tvar concurrency int\n\tvar duration time.Duration\n\tvar wait time.Duration\n\tvar delay time.Duration\n\tvar workerDelay time.Duration\n\tvar logLevel string\n\tvar slots int\n\tvar failureRate float32\n\tvar payloadSize string\n\tvar eventFanout int\n\n\tvar loadtest = &cobra.Command{\n\t\tUse: "loadtest",\n\t\tRun: func(cmd *cobra.Command, args []string) {\n\t\t\tl = logger.NewStdErr(\n\t\t\t\t&shared.LoggerConfigFile{\n\t\t\t\t\tLevel:  logLevel,\n\t\t\t\t\tFormat: "console",\n\t\t\t\t},\n\t\t\t\t"loadtest",\n\t\t\t)\n\n\t\t\t// enable pprof if requested\n\t\t\tif os.Getenv("PPROF_ENABLED") == "true" {\n\t\t\t\tgo func() {\n\t\t\t\t\tlog.Println(http.ListenAndServe("localhost:6060", nil))\n\t\t\t\t}()\n\t\t\t}\n\n\t\t\tif err := do(duration, events, delay, wait, concurrency, workerDelay, slots, failureRate, payloadSize, eventFanout); err != nil {\n\t\t\t\tlog.Println(err)\n\t\t\t\tpanic("load test failed")\n\t\t\t}\n\t\t},\n\t}\n\n\tloadtest.Flags().IntVarP(&events, "events", "e", 10, "events per second")\n\tloadtest.Flags().IntVarP(&concurrency, "concurrency", "c", 0, "concurrency specifies the maximum events to run at the same time")\n\tloadtest.Flags().DurationVarP(&duration, "duration", "d", 10*time.Second, "duration specifies the total time to run the load test")\n\tloadtest.Flags().DurationVarP(&delay, "delay", "D", 0, "delay specifies the time to wait in each event to simulate slow tasks")\n\tloadtest.Flags().DurationVarP(&wait, "wait", "w", 10*time.Second, "wait specifies the total time to wait until events complete")\n\tloadtest.Flags().DurationVarP(&workerDelay, "workerDelay", "p", 0*time.Second, "workerDelay specifies the time to wait before starting the worker")\n\tloadtest.Flags().StringVarP(&logLevel, "level", "l", "info", "logLevel specifies the log level (debug, info, warn, error)")\n\tloadtest.Flags().IntVarP(&slots, "slots", "s", 0, "slots specifies the number of slots to use in the worker")\n\tloadtest.Flags().Float32VarP(&failureRate, "failureRate", "f", 0, "failureRate specifies the rate of failure for the worker")\n\tloadtest.Flags().StringVarP(&payloadSize, "payloadSize", "P", "0kb", "payload specifies the size of the payload to send")\n\tloadtest.Flags().IntVarP(&eventFanout, "eventFanout", "F", 1, "eventFanout specifies the number of events to fanout")\n\n\tcmd := &cobra.Command{Use: "app"}\n\tcmd.AddCommand(loadtest)\n\tif err := cmd.Execute(); err != nil {\n\t\tpanic(err)\n\t}\n}\n',
      language: 'go',
      source: 'examples/go/z_v0/loadtest/cli/main.go',
      highlights: {},
    },
  L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL2dvL3pfdjAvbG9hZHRlc3QvY2xpL3J1bi5nbw__:
    {
      content:
        'package main\n\nimport (\n\t"context"\n\t"fmt"\n\t"math/rand/v2"\n\t"sync"\n\t"time"\n\n\t"github.com/hatchet-dev/hatchet/pkg/client"\n\t"github.com/hatchet-dev/hatchet/pkg/worker"\n)\n\ntype stepOneOutput struct {\n\tMessage string `json:"message"`\n}\n\nfunc run(ctx context.Context, delay time.Duration, executions chan<- time.Duration, concurrency, slots int, failureRate float32, eventFanout int) (int64, int64) {\n\tc, err := client.New(\n\t\tclient.WithLogLevel("warn"),\n\t)\n\n\tif err != nil {\n\t\tpanic(err)\n\t}\n\n\tw, err := worker.NewWorker(\n\t\tworker.WithClient(\n\t\t\tc,\n\t\t),\n\t\tworker.WithLogLevel("warn"),\n\t\tworker.WithMaxRuns(slots),\n\t)\n\n\tif err != nil {\n\t\tpanic(err)\n\t}\n\n\tmx := sync.Mutex{}\n\tvar count int64\n\tvar uniques int64\n\tvar executed []int64\n\n\tvar concurrencyOpts *worker.WorkflowConcurrency\n\tif concurrency > 0 {\n\t\tconcurrencyOpts = worker.Expression("\'global\'").MaxRuns(int32(concurrency))\n\t}\n\n\tstep := func(ctx worker.HatchetContext) (result *stepOneOutput, err error) {\n\t\tvar input Event\n\t\terr = ctx.WorkflowInput(&input)\n\t\tif err != nil {\n\t\t\treturn nil, err\n\t\t}\n\n\t\ttook := time.Since(input.CreatedAt)\n\t\tl.Info().Msgf("executing %d took %s", input.ID, took)\n\n\t\tmx.Lock()\n\t\texecutions <- took\n\t\t// detect duplicate in executed slice\n\t\tvar duplicate bool\n\t\t// for i := 0; i < len(executed)-1; i++ {\n\t\t// \tif executed[i] == input.ID {\n\t\t// \t\tduplicate = true\n\t\t// \t\tbreak\n\t\t// \t}\n\t\t// }\n\t\tif duplicate {\n\t\t\tl.Warn().Str("step-run-id", ctx.StepRunId()).Msgf("duplicate %d", input.ID)\n\t\t}\n\t\tif !duplicate {\n\t\t\tuniques++\n\t\t}\n\t\tcount++\n\t\texecuted = append(executed, input.ID)\n\t\tmx.Unlock()\n\n\t\ttime.Sleep(delay)\n\n\t\tif failureRate > 0 {\n\t\t\tif rand.Float32() < failureRate {\n\t\t\t\treturn nil, fmt.Errorf("random failure")\n\t\t\t}\n\t\t}\n\n\t\treturn &stepOneOutput{\n\t\t\tMessage: "This ran at: " + time.Now().Format(time.RFC3339Nano),\n\t\t}, nil\n\t}\n\n\tfor i := range eventFanout {\n\t\terr = w.RegisterWorkflow(\n\t\t\t&worker.WorkflowJob{\n\t\t\t\tName:        fmt.Sprintf("load-test-%d", i),\n\t\t\t\tDescription: "Load testing",\n\t\t\t\tOn:          worker.Event("load-test:event"),\n\t\t\t\tConcurrency: concurrencyOpts,\n\t\t\t\t// ScheduleTimeout: "30s",\n\t\t\t\tSteps: []*worker.WorkflowStep{\n\t\t\t\t\tworker.Fn(step).SetName("step-one").SetTimeout("5m"),\n\t\t\t\t},\n\t\t\t},\n\t\t)\n\n\t\tif err != nil {\n\t\t\tpanic(err)\n\t\t}\n\t}\n\n\tcleanup, err := w.Start()\n\tif err != nil {\n\t\tpanic(fmt.Errorf("error starting worker: %w", err))\n\t}\n\n\t<-ctx.Done()\n\n\tif err := cleanup(); err != nil {\n\t\tpanic(fmt.Errorf("error cleaning up: %w", err))\n\t}\n\n\tmx.Lock()\n\tdefer mx.Unlock()\n\treturn count, uniques\n}\n',
      language: 'go',
      source: 'examples/go/z_v0/loadtest/cli/run.go',
      highlights: {},
    },
  L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL2dvL3pfdjAvbG9hZHRlc3QvZW1pdHRlci9tYWluLmdv:
    {
      content:
        'package main\n\nimport (\n\t"context"\n\t"fmt"\n\t"time"\n\n\t"github.com/hatchet-dev/hatchet/pkg/client"\n\t"github.com/hatchet-dev/hatchet/pkg/cmdutils"\n\t"github.com/joho/godotenv"\n)\n\ntype Event struct {\n\tID        uint64    `json:"id"`\n\tCreatedAt time.Time `json:"created_at"`\n}\n\ntype stepOneOutput struct {\n\tMessage string `json:"message"`\n}\n\nfunc StepOne(ctx context.Context, input *Event) (result *stepOneOutput, err error) {\n\tfmt.Println(input.ID, "delay", time.Since(input.CreatedAt))\n\n\treturn &stepOneOutput{\n\t\tMessage: "This ran at: " + time.Now().Format(time.RubyDate),\n\t}, nil\n}\n\nfunc main() {\n\terr := godotenv.Load()\n\n\tif err != nil {\n\t\tpanic(err)\n\t}\n\n\tclient, err := client.New()\n\n\tif err != nil {\n\t\tpanic(err)\n\t}\n\n\tinterruptCtx, cancel := cmdutils.InterruptContextFromChan(cmdutils.InterruptChan())\n\tdefer cancel()\n\n\tvar id uint64\n\tgo func() {\n\t\tfor {\n\t\t\tselect {\n\t\t\tcase <-time.After(5 * time.Second):\n\t\t\t\tfor i := 0; i < 100; i++ {\n\t\t\t\t\tid++\n\n\t\t\t\t\tev := Event{CreatedAt: time.Now(), ID: id}\n\t\t\t\t\tfmt.Println("pushed event", ev.ID)\n\t\t\t\t\terr = client.Event().Push(interruptCtx, "test:event", ev)\n\t\t\t\t\tif err != nil {\n\t\t\t\t\t\tpanic(err)\n\t\t\t\t\t}\n\t\t\t\t}\n\t\t\tcase <-interruptCtx.Done():\n\t\t\t\treturn\n\t\t\t}\n\t\t}\n\t}()\n\n\tfor {\n\t\tselect {\n\t\tcase <-interruptCtx.Done():\n\t\t\treturn\n\t\tdefault:\n\t\t\ttime.Sleep(time.Second)\n\t\t}\n\t}\n}\n',
      language: 'go',
      source: 'examples/go/z_v0/loadtest/emitter/main.go',
      highlights: {},
    },
  L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL2dvL3pfdjAvbG9hZHRlc3QvcmFtcHVwL2RvLmdv:
    {
      content:
        'package rampup\n\nimport (\n\t"context"\n\t"fmt"\n\t"log"\n\t"slices"\n\t"sync"\n\t"time"\n\n\t"github.com/rs/zerolog"\n)\n\nvar l zerolog.Logger\n\nfunc do(duration time.Duration, startEventsPerSecond, amount int, increase, delay, wait, maxAcceptableDuration, maxAcceptableSchedule time.Duration, includeDroppedEvents bool, concurrency int) error {\n\tl.Debug().Msgf("testing with duration=%s, amount=%d, increase=%d, delay=%s, wait=%s, concurrency=%d", duration, amount, increase, delay, wait, concurrency)\n\n\tctx, cancel := context.WithCancel(context.Background())\n\tdefer cancel()\n\n\tafter := 10 * time.Second\n\n\tgo func() {\n\t\ttime.Sleep(duration + after + wait + 5*time.Second)\n\t\tcancel()\n\t}()\n\n\thook := make(chan time.Duration, 1)\n\n\tscheduled := make(chan int64, 100000)\n\texecuted := make(chan int64, 100000)\n\n\tids := []int64{}\n\tidLock := sync.Mutex{}\n\n\tgo func() {\n\t\tfor s := range scheduled {\n\t\t\tl.Debug().Msgf("scheduled %d", s)\n\t\t\tidLock.Lock()\n\t\t\tids = append(ids, s)\n\t\t\tidLock.Unlock()\n\n\t\t\tgo func(s int64) {\n\t\t\t\ttime.Sleep(maxAcceptableDuration)\n\t\t\t\tidLock.Lock()\n\t\t\t\tdefer idLock.Unlock()\n\t\t\t\tfor _, e := range ids {\n\t\t\t\t\tif e == s {\n\t\t\t\t\t\tif includeDroppedEvents {\n\t\t\t\t\t\t\tpanic(fmt.Errorf("event %d did not execute in time", s))\n\t\t\t\t\t\t} else {\n\t\t\t\t\t\t\tl.Warn().Msgf("event %d did not execute in time", s)\n\t\t\t\t\t\t}\n\t\t\t\t\t}\n\t\t\t\t}\n\t\t\t}(s)\n\t\t}\n\t}()\n\n\tgo func() {\n\t\tfor e := range executed {\n\t\t\tl.Debug().Msgf("executed %d", e)\n\t\t\tidLock.Lock()\n\t\t\tids = slices.DeleteFunc(ids, func(s int64) bool {\n\t\t\t\treturn s == e\n\t\t\t})\n\t\t\tidLock.Unlock()\n\t\t}\n\t}()\n\n\tgo func() {\n\t\trun(ctx, delay, concurrency, maxAcceptableDuration, hook, executed)\n\t}()\n\n\temit(ctx, startEventsPerSecond, amount, increase, duration, maxAcceptableSchedule, hook, scheduled)\n\n\ttime.Sleep(after)\n\n\tlog.Printf("✅ success")\n\n\treturn nil\n}\n',
      language: 'go',
      source: 'examples/go/z_v0/loadtest/rampup/do.go',
      highlights: {},
    },
  L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL2dvL3pfdjAvbG9hZHRlc3QvcmFtcHVwL2VtaXQuZ28_:
    {
      content:
        'package rampup\n\nimport (\n\t"context"\n\t"fmt"\n\t"sync"\n\t"time"\n\n\t"github.com/hatchet-dev/hatchet/pkg/client"\n)\n\ntype Event struct {\n\tID        int64     `json:"id"`\n\tCreatedAt time.Time `json:"created_at"`\n}\n\nfunc emit(ctx context.Context, startEventsPerSecond, amount int, increase, duration, maxAcceptableSchedule time.Duration, hook <-chan time.Duration, scheduled chan<- int64) int64 {\n\tc, err := client.New()\n\n\tif err != nil {\n\t\tpanic(err)\n\t}\n\n\tvar id int64\n\tmx := sync.Mutex{}\n\tgo func() {\n\t\ttimer := time.After(duration)\n\t\tstart := time.Now()\n\n\t\tvar eventsPerSecond int\n\t\tgo func() {\n\t\t\ttook := <-hook\n\t\t\tpanic(fmt.Errorf("event took too long to schedule: %s at %d events/s", took, eventsPerSecond))\n\t\t}()\n\t\tfor {\n\t\t\t// emit amount * increase events per second\n\t\t\teventsPerSecond = startEventsPerSecond + (amount * int(time.Since(start).Seconds()) / int(increase.Seconds()))\n\t\t\tincrease += 1\n\t\t\tif eventsPerSecond < 1 {\n\t\t\t\teventsPerSecond = 1\n\t\t\t}\n\t\t\tl.Debug().Msgf("emitting %d events per second", eventsPerSecond)\n\t\t\tselect {\n\t\t\tcase <-time.After(time.Second / time.Duration(eventsPerSecond)):\n\t\t\t\tmx.Lock()\n\t\t\t\tid += 1\n\n\t\t\t\tgo func(id int64) {\n\t\t\t\t\tvar err error\n\t\t\t\t\tev := Event{CreatedAt: time.Now(), ID: id}\n\t\t\t\t\tl.Debug().Msgf("pushed event %d", ev.ID)\n\t\t\t\t\terr = c.Event().Push(context.Background(), "load-test:event", ev)\n\t\t\t\t\tif err != nil {\n\t\t\t\t\t\tpanic(fmt.Errorf("error pushing event: %w", err))\n\t\t\t\t\t}\n\t\t\t\t\ttook := time.Since(ev.CreatedAt)\n\t\t\t\t\tl.Debug().Msgf("pushed event %d took %s", ev.ID, took)\n\n\t\t\t\t\tif took > maxAcceptableSchedule {\n\t\t\t\t\t\tpanic(fmt.Errorf("event took too long to schedule: %s at %d events/s", took, eventsPerSecond))\n\t\t\t\t\t}\n\n\t\t\t\t\tscheduled <- id\n\t\t\t\t}(id)\n\n\t\t\t\tmx.Unlock()\n\t\t\tcase <-timer:\n\t\t\t\tl.Debug().Msgf("done emitting events due to timer at %d", id)\n\t\t\t\treturn\n\t\t\tcase <-ctx.Done():\n\t\t\t\tl.Debug().Msgf("done emitting events due to interruption at %d", id)\n\t\t\t\treturn\n\t\t\t}\n\t\t}\n\t}()\n\n\tfor {\n\t\tselect {\n\t\tcase <-ctx.Done():\n\t\t\tmx.Lock()\n\t\t\tdefer mx.Unlock()\n\t\t\treturn id\n\t\tdefault:\n\t\t\ttime.Sleep(time.Second)\n\t\t}\n\t}\n}\n',
      language: 'go',
      source: 'examples/go/z_v0/loadtest/rampup/emit.go',
      highlights: {},
    },
  L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL2dvL3pfdjAvbG9hZHRlc3QvcmFtcHVwL3JhbXBfdXBfZTJlX3Rlc3QuZ28_:
    {
      content:
        '//go:build load\n\npackage rampup\n\nimport (\n\t"context"\n\t"log"\n\t"os"\n\t"sync"\n\t"testing"\n\t"time"\n\n\t"github.com/hatchet-dev/hatchet/internal/testutils"\n\t"github.com/hatchet-dev/hatchet/pkg/config/shared"\n\t"github.com/hatchet-dev/hatchet/pkg/logger"\n)\n\nfunc TestRampUp(t *testing.T) {\n\ttestutils.Prepare(t)\n\n\ttype args struct {\n\t\tduration time.Duration\n\t\tincrease time.Duration\n\t\tamount   int\n\t\tdelay    time.Duration\n\t\twait     time.Duration\n\t\t// includeDroppedEvents is whether to fail on events that were dropped due to being scheduled too late\n\t\tincludeDroppedEvents bool\n\t\t// maxAcceptableDuration is the maximum acceptable duration for a single event to be scheduled (from start to finish)\n\t\tmaxAcceptableDuration time.Duration\n\t\t// maxAcceptableSchedule is the maximum acceptable time for an event to be purely scheduled, regardless of whether it will run or not\n\t\tmaxAcceptableSchedule time.Duration\n\t\tconcurrency           int\n\t\tstartEventsPerSecond  int\n\t}\n\n\tl = logger.NewStdErr(\n\t\t&shared.LoggerConfigFile{\n\t\t\tLevel:  "warn",\n\t\t\tFormat: "console",\n\t\t},\n\t\t"loadtest",\n\t)\n\n\t// get ramp up duration from env\n\tmaxAcceptableDurationSeconds := 2 * time.Second\n\n\tif os.Getenv("RAMP_UP_DURATION_TIMEOUT") != "" {\n\t\tvar parseErr error\n\t\tmaxAcceptableDurationSeconds, parseErr = time.ParseDuration(os.Getenv("RAMP_UP_DURATION_TIMEOUT"))\n\n\t\tif parseErr != nil {\n\t\t\tt.Fatalf("could not parse RAMP_UP_DURATION_TIMEOUT %s: %s", os.Getenv("RAMP_UP_DURATION_TIMEOUT"), parseErr)\n\t\t}\n\t}\n\n\tlog.Printf("TestRampUp with maxAcceptableDurationSeconds: %s", maxAcceptableDurationSeconds.String())\n\n\ttests := []struct {\n\t\tname    string\n\t\targs    args\n\t\twantErr bool\n\t}{{\n\t\tname: "normal test",\n\t\targs: args{\n\t\t\tstartEventsPerSecond:  1,\n\t\t\tduration:              300 * time.Second,\n\t\t\tincrease:              10 * time.Second,\n\t\t\tamount:                1,\n\t\t\tdelay:                 0 * time.Second,\n\t\t\twait:                  30 * time.Second,\n\t\t\tincludeDroppedEvents:  true,\n\t\t\tmaxAcceptableDuration: maxAcceptableDurationSeconds,\n\t\t\tmaxAcceptableSchedule: 2 * time.Second,\n\t\t\tconcurrency:           0,\n\t\t},\n\t}}\n\n\tctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)\n\n\tsetup := sync.WaitGroup{}\n\n\tgo func() {\n\t\tsetup.Add(1)\n\t\tlog.Printf("setup start")\n\t\ttestutils.SetupEngine(ctx, t)\n\t\tsetup.Done()\n\t\tlog.Printf("setup end")\n\t}()\n\n\t// TODO instead of waiting, figure out when the engine setup is complete\n\ttime.Sleep(15 * time.Second)\n\n\tfor _, tt := range tests {\n\t\tt.Run(tt.name, func(t *testing.T) {\n\n\t\t\tif err := do(tt.args.duration, tt.args.startEventsPerSecond, tt.args.amount, tt.args.increase, tt.args.delay, tt.args.wait, tt.args.maxAcceptableDuration, tt.args.maxAcceptableSchedule, tt.args.includeDroppedEvents, tt.args.concurrency); (err != nil) != tt.wantErr {\n\t\t\t\tt.Errorf("do() error = %v, wantErr %v", err, tt.wantErr)\n\t\t\t}\n\t\t})\n\t}\n\n\tcancel()\n\n\tlog.Printf("test complete")\n\tsetup.Wait()\n\tlog.Printf("cleanup complete")\n}\n',
      language: 'go',
      source: 'examples/go/z_v0/loadtest/rampup/ramp_up_e2e_test.go',
      highlights: {},
    },
  L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL2dvL3pfdjAvbG9hZHRlc3QvcmFtcHVwL3J1bi5nbw__:
    {
      content:
        'package rampup\n\nimport (\n\t"context"\n\t"fmt"\n\t"sync"\n\t"time"\n\n\t"github.com/hatchet-dev/hatchet/pkg/client"\n\t"github.com/hatchet-dev/hatchet/pkg/worker"\n)\n\ntype stepOneOutput struct {\n\tMessage string `json:"message"`\n}\n\nfunc getConcurrencyKey(ctx worker.HatchetContext) (string, error) {\n\treturn "my-key", nil\n}\n\nfunc run(ctx context.Context, delay time.Duration, concurrency int, maxAcceptableDuration time.Duration, hook chan<- time.Duration, executedCh chan<- int64) (int64, int64) {\n\tc, err := client.New(\n\t\tclient.WithLogLevel("warn"),\n\t)\n\n\tif err != nil {\n\t\tpanic(err)\n\t}\n\n\tw, err := worker.NewWorker(\n\t\tworker.WithClient(\n\t\t\tc,\n\t\t),\n\t\tworker.WithLogLevel("warn"),\n\t\tworker.WithMaxRuns(200),\n\t)\n\n\tif err != nil {\n\t\tpanic(err)\n\t}\n\n\tmx := sync.Mutex{}\n\tvar count int64\n\tvar uniques int64\n\tvar executed []int64\n\n\tvar concurrencyOpts *worker.WorkflowConcurrency\n\tif concurrency > 0 {\n\t\tconcurrencyOpts = worker.Concurrency(getConcurrencyKey).MaxRuns(int32(concurrency))\n\t}\n\n\terr = w.On(\n\t\tworker.Event("load-test:event"),\n\t\t&worker.WorkflowJob{\n\t\t\tName:        "load-test",\n\t\t\tDescription: "Load testing",\n\t\t\tConcurrency: concurrencyOpts,\n\t\t\tSteps: []*worker.WorkflowStep{\n\t\t\t\tworker.Fn(func(ctx worker.HatchetContext) (result *stepOneOutput, err error) {\n\t\t\t\t\tvar input Event\n\t\t\t\t\terr = ctx.WorkflowInput(&input)\n\t\t\t\t\tif err != nil {\n\t\t\t\t\t\treturn nil, err\n\t\t\t\t\t}\n\n\t\t\t\t\ttook := time.Since(input.CreatedAt)\n\n\t\t\t\t\tl.Debug().Msgf("executing %d took %s", input.ID, took)\n\n\t\t\t\t\tif took > maxAcceptableDuration {\n\t\t\t\t\t\thook <- took\n\t\t\t\t\t}\n\n\t\t\t\t\texecutedCh <- input.ID\n\n\t\t\t\t\tmx.Lock()\n\n\t\t\t\t\t// detect duplicate in executed slice\n\t\t\t\t\tvar duplicate bool\n\t\t\t\t\tfor i := 0; i < len(executed)-1; i++ {\n\t\t\t\t\t\tif executed[i] == input.ID {\n\t\t\t\t\t\t\tduplicate = true\n\t\t\t\t\t\t}\n\t\t\t\t\t}\n\t\t\t\t\tif duplicate {\n\t\t\t\t\t\tl.Warn().Str("step-run-id", ctx.StepRunId()).Msgf("duplicate %d", input.ID)\n\t\t\t\t\t} else {\n\t\t\t\t\t\tuniques += 1\n\t\t\t\t\t}\n\t\t\t\t\tcount += 1\n\t\t\t\t\texecuted = append(executed, input.ID)\n\t\t\t\t\tmx.Unlock()\n\n\t\t\t\t\ttime.Sleep(delay)\n\n\t\t\t\t\treturn &stepOneOutput{\n\t\t\t\t\t\tMessage: "This ran at: " + time.Now().Format(time.RFC3339Nano),\n\t\t\t\t\t}, nil\n\t\t\t\t}).SetName("step-one"),\n\t\t\t},\n\t\t},\n\t)\n\n\tif err != nil {\n\t\tpanic(err)\n\t}\n\n\tcleanup, err := w.Start()\n\tif err != nil {\n\t\tpanic(fmt.Errorf("error starting worker: %w", err))\n\t}\n\n\t<-ctx.Done()\n\n\tif err := cleanup(); err != nil {\n\t\tpanic(fmt.Errorf("error cleaning up: %w", err))\n\t}\n\n\tmx.Lock()\n\tdefer mx.Unlock()\n\treturn count, uniques\n}\n',
      language: 'go',
      source: 'examples/go/z_v0/loadtest/rampup/run.go',
      highlights: {},
    },
  L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL2dvL3pfdjAvbG9hZHRlc3Qvd29ya2VyL21haW4uZ28_:
    {
      content:
        'package main\n\nimport (\n\t"fmt"\n\t"time"\n\n\t"github.com/joho/godotenv"\n\n\t"github.com/hatchet-dev/hatchet/pkg/client"\n\t"github.com/hatchet-dev/hatchet/pkg/cmdutils"\n\t"github.com/hatchet-dev/hatchet/pkg/worker"\n)\n\ntype Event struct {\n\tID        uint64    `json:"id"`\n\tCreatedAt time.Time `json:"created_at"`\n}\n\ntype stepOneOutput struct {\n\tMessage string `json:"message"`\n}\n\nfunc StepOne(ctx worker.HatchetContext) (result *stepOneOutput, err error) {\n\tinput := &Event{}\n\n\terr = ctx.WorkflowInput(input)\n\n\tif err != nil {\n\t\treturn nil, err\n\t}\n\n\tfmt.Println(input.ID, "delay", time.Since(input.CreatedAt))\n\n\treturn &stepOneOutput{\n\t\tMessage: "This ran at: " + time.Now().Format(time.RubyDate),\n\t}, nil\n}\n\nfunc main() {\n\terr := godotenv.Load()\n\n\tif err != nil {\n\t\tpanic(err)\n\t}\n\n\tclient, err := client.New()\n\n\tif err != nil {\n\t\tpanic(err)\n\t}\n\n\tw, err := worker.NewWorker(\n\t\tworker.WithClient(\n\t\t\tclient,\n\t\t),\n\t)\n\n\tif err != nil {\n\t\tpanic(err)\n\t}\n\n\terr = w.On(\n\t\tworker.Event("test:event"),\n\t\t&worker.WorkflowJob{\n\t\t\tName:        "scheduled-workflow",\n\t\t\tDescription: "This runs at a scheduled time.",\n\t\t\tSteps: []*worker.WorkflowStep{\n\t\t\t\tworker.Fn(StepOne).SetName("step-one"),\n\t\t\t},\n\t\t},\n\t)\n\n\tif err != nil {\n\t\tpanic(err)\n\t}\n\n\tch := cmdutils.InterruptChan()\n\n\tcleanup, err := w.Start()\n\tif err != nil {\n\t\tpanic(fmt.Errorf("error starting worker: %w", err))\n\t}\n\n\t<-ch\n\n\tif err := cleanup(); err != nil {\n\t\tpanic(fmt.Errorf("error cleaning up: %w", err))\n\t}\n}\n',
      language: 'go',
      source: 'examples/go/z_v0/loadtest/worker/main.go',
      highlights: {},
    },
  L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL2dvL3pfdjAvbG9nZ2luZy9tYWluLmdv:
    {
      content:
        'package main\n\nimport (\n\t"context"\n\t"fmt"\n\t"log"\n\n\t"github.com/joho/godotenv"\n\n\t"github.com/hatchet-dev/hatchet/pkg/client"\n\t"github.com/hatchet-dev/hatchet/pkg/cmdutils"\n\t"github.com/hatchet-dev/hatchet/pkg/worker"\n)\n\ntype userCreateEvent struct {\n\tUsername string            `json:"username"`\n\tUserID   string            `json:"user_id"`\n\tData     map[string]string `json:"data"`\n}\n\ntype stepOneOutput struct {\n\tMessage string `json:"message"`\n}\n\nfunc main() {\n\terr := godotenv.Load()\n\tif err != nil {\n\t\tpanic(err)\n\t}\n\n\tevents := make(chan string, 50)\n\tinterrupt := cmdutils.InterruptChan()\n\n\tcleanup, err := run(events)\n\tif err != nil {\n\t\tpanic(err)\n\t}\n\n\t<-interrupt\n\n\tif err := cleanup(); err != nil {\n\t\tpanic(fmt.Errorf("error cleaning up: %w", err))\n\t}\n}\n\nfunc run(events chan<- string) (func() error, error) {\n\tc, err := client.New()\n\n\tif err != nil {\n\t\treturn nil, fmt.Errorf("error creating client: %w", err)\n\t}\n\n\tw, err := worker.NewWorker(\n\t\tworker.WithClient(\n\t\t\tc,\n\t\t),\n\t)\n\tif err != nil {\n\t\treturn nil, fmt.Errorf("error creating worker: %w", err)\n\t}\n\n\terr = w.RegisterWorkflow(\n\t\t&worker.WorkflowJob{\n\t\t\tOn:          worker.Events("user:log:simple"),\n\t\t\tName:        "simple",\n\t\t\tDescription: "This runs after an update to the user model.",\n\t\t\tConcurrency: worker.Expression("input.user_id"),\n\t\t\tSteps: []*worker.WorkflowStep{\n\t\t\t\tworker.Fn(func(ctx worker.HatchetContext) (result *stepOneOutput, err error) {\n\t\t\t\t\tinput := &userCreateEvent{}\n\n\t\t\t\t\terr = ctx.WorkflowInput(input)\n\n\t\t\t\t\tif err != nil {\n\t\t\t\t\t\treturn nil, err\n\t\t\t\t\t}\n\n\t\t\t\t\tlog.Printf("step-one")\n\t\t\t\t\tevents <- "step-one"\n\n\t\t\t\t\tfor i := 0; i < 1000; i++ {\n\t\t\t\t\t\tctx.Log(fmt.Sprintf("step-one: %d", i))\n\t\t\t\t\t}\n\n\t\t\t\t\treturn &stepOneOutput{\n\t\t\t\t\t\tMessage: "Username is: " + input.Username,\n\t\t\t\t\t}, nil\n\t\t\t\t},\n\t\t\t\t).SetName("step-one"),\n\t\t\t},\n\t\t},\n\t)\n\tif err != nil {\n\t\treturn nil, fmt.Errorf("error registering workflow: %w", err)\n\t}\n\n\tgo func() {\n\t\ttestEvent := userCreateEvent{\n\t\t\tUsername: "echo-test",\n\t\t\tUserID:   "1234",\n\t\t\tData: map[string]string{\n\t\t\t\t"test": "test",\n\t\t\t},\n\t\t}\n\n\t\tlog.Printf("pushing event user:create:simple")\n\t\t// push an event\n\t\terr := c.Event().Push(\n\t\t\tcontext.Background(),\n\t\t\t"user:log:simple",\n\t\t\ttestEvent,\n\t\t\tclient.WithEventMetadata(map[string]string{\n\t\t\t\t"hello": "world",\n\t\t\t}),\n\t\t)\n\t\tif err != nil {\n\t\t\tpanic(fmt.Errorf("error pushing event: %w", err))\n\t\t}\n\t}()\n\n\tcleanup, err := w.Start()\n\tif err != nil {\n\t\tpanic(err)\n\t}\n\n\treturn cleanup, nil\n}\n',
      language: 'go',
      source: 'examples/go/z_v0/logging/main.go',
      highlights: {},
    },
  L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL2dvL3pfdjAvbWFudWFsLXRyaWdnZXIvdHJpZ2dlci9tYWluLmdv:
    {
      content:
        'package main\n\nimport (\n\t"fmt"\n\t"time"\n\n\t"github.com/joho/godotenv"\n\n\t"github.com/hatchet-dev/hatchet/pkg/client"\n\t"github.com/hatchet-dev/hatchet/pkg/cmdutils"\n)\n\ntype userCreateEvent struct {\n\tUsername string            `json:"username"`\n\tUserID   string            `json:"user_id"`\n\tData     map[string]string `json:"data"`\n}\n\nfunc main() {\n\terr := godotenv.Load()\n\tif err != nil {\n\t\tpanic(err)\n\t}\n\n\tevents := make(chan string, 50)\n\tif err := run(cmdutils.InterruptChan(), events); err != nil {\n\t\tpanic(err)\n\t}\n}\n\nfunc run(ch <-chan interface{}, events chan<- string) error {\n\tc, err := client.New()\n\n\tif err != nil {\n\t\treturn fmt.Errorf("error creating client: %w", err)\n\t}\n\n\ttime.Sleep(1 * time.Second)\n\n\t// trigger workflow\n\tworkflow, err := c.Admin().RunWorkflow(\n\t\t"post-user-update",\n\t\t&userCreateEvent{\n\t\t\tUsername: "echo-test",\n\t\t\tUserID:   "1234",\n\t\t\tData: map[string]string{\n\t\t\t\t"test": "test",\n\t\t\t},\n\t\t},\n\t\tclient.WithRunMetadata(map[string]interface{}{\n\t\t\t"hello": "world",\n\t\t}),\n\t)\n\n\tif err != nil {\n\t\treturn fmt.Errorf("error running workflow: %w", err)\n\t}\n\n\tfmt.Println("workflow run id:", workflow.WorkflowRunId())\n\n\tinterruptCtx, cancel := cmdutils.InterruptContextFromChan(ch)\n\tdefer cancel()\n\n\terr = c.Subscribe().On(interruptCtx, workflow.WorkflowRunId(), func(event client.WorkflowEvent) error {\n\t\tfmt.Println(event.EventPayload)\n\n\t\treturn nil\n\t})\n\n\treturn err\n}\n',
      language: 'go',
      source: 'examples/go/z_v0/manual-trigger/trigger/main.go',
      highlights: {},
    },
  L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL2dvL3pfdjAvbWFudWFsLXRyaWdnZXIvd29ya2VyL21haW4uZ28_:
    {
      content:
        'package main\n\nimport (\n\t"fmt"\n\t"time"\n\n\t"github.com/joho/godotenv"\n\n\t"github.com/hatchet-dev/hatchet/pkg/client"\n\t"github.com/hatchet-dev/hatchet/pkg/cmdutils"\n\t"github.com/hatchet-dev/hatchet/pkg/worker"\n)\n\ntype userCreateEvent struct {\n\tUsername string            `json:"username"`\n\tUserID   string            `json:"user_id"`\n\tData     map[string]string `json:"data"`\n}\n\ntype stepOutput struct {\n\tMessage string `json:"message"`\n}\n\nfunc main() {\n\terr := godotenv.Load()\n\tif err != nil {\n\t\tpanic(err)\n\t}\n\n\tevents := make(chan string, 50)\n\tif err := run(cmdutils.InterruptChan(), events); err != nil {\n\t\tpanic(err)\n\t}\n}\n\nfunc run(ch <-chan interface{}, events chan<- string) error {\n\tc, err := client.New()\n\n\tif err != nil {\n\t\treturn fmt.Errorf("error creating client: %w", err)\n\t}\n\n\tw, err := worker.NewWorker(\n\t\tworker.WithClient(\n\t\t\tc,\n\t\t),\n\t)\n\tif err != nil {\n\t\treturn fmt.Errorf("error creating worker: %w", err)\n\t}\n\n\ttestSvc := w.NewService("test")\n\n\terr = testSvc.On(\n\t\tworker.Events("user:create:simple"),\n\t\t&worker.WorkflowJob{\n\t\t\tName:        "post-user-update",\n\t\t\tDescription: "This runs after an update to the user model.",\n\t\t\tSteps: []*worker.WorkflowStep{\n\t\t\t\tworker.Fn(func(ctx worker.HatchetContext) (result *stepOutput, err error) {\n\t\t\t\t\tinput := &userCreateEvent{}\n\t\t\t\t\tctx.WorkflowInput(input)\n\n\t\t\t\t\ttime.Sleep(1 * time.Second)\n\n\t\t\t\t\treturn &stepOutput{\n\t\t\t\t\t\tMessage: "Step 1 got username: " + input.Username,\n\t\t\t\t\t}, nil\n\t\t\t\t},\n\t\t\t\t).SetName("step-one"),\n\t\t\t\tworker.Fn(func(ctx worker.HatchetContext) (result *stepOutput, err error) {\n\t\t\t\t\tinput := &userCreateEvent{}\n\t\t\t\t\tctx.WorkflowInput(input)\n\n\t\t\t\t\ttime.Sleep(2 * time.Second)\n\n\t\t\t\t\treturn &stepOutput{\n\t\t\t\t\t\tMessage: "Step 2 got username: " + input.Username,\n\t\t\t\t\t}, nil\n\t\t\t\t}).SetName("step-two"),\n\t\t\t\tworker.Fn(func(ctx worker.HatchetContext) (result *stepOutput, err error) {\n\t\t\t\t\tstep1Out := &stepOutput{}\n\t\t\t\t\tctx.StepOutput("step-one", step1Out)\n\n\t\t\t\t\tstep2Out := &stepOutput{}\n\t\t\t\t\tctx.StepOutput("step-two", step2Out)\n\n\t\t\t\t\ttime.Sleep(3 * time.Second)\n\n\t\t\t\t\treturn &stepOutput{\n\t\t\t\t\t\tMessage: "Step 3: has parents 1 and 2:" + step1Out.Message + ", " + step2Out.Message,\n\t\t\t\t\t}, nil\n\t\t\t\t}).SetName("step-three").AddParents("step-one", "step-two"),\n\t\t\t\tworker.Fn(func(ctx worker.HatchetContext) (result *stepOutput, err error) {\n\t\t\t\t\tstep1Out := &stepOutput{}\n\t\t\t\t\tctx.StepOutput("step-one", step1Out)\n\n\t\t\t\t\tstep3Out := &stepOutput{}\n\t\t\t\t\tctx.StepOutput("step-three", step3Out)\n\n\t\t\t\t\ttime.Sleep(4 * time.Second)\n\n\t\t\t\t\treturn &stepOutput{\n\t\t\t\t\t\tMessage: "Step 4: has parents 1 and 3" + step1Out.Message + ", " + step3Out.Message,\n\t\t\t\t\t}, nil\n\t\t\t\t}).SetName("step-four").AddParents("step-one", "step-three"),\n\t\t\t\tworker.Fn(func(ctx worker.HatchetContext) (result *stepOutput, err error) {\n\t\t\t\t\tstep4Out := &stepOutput{}\n\t\t\t\t\tctx.StepOutput("step-four", step4Out)\n\n\t\t\t\t\ttime.Sleep(5 * time.Second)\n\n\t\t\t\t\treturn &stepOutput{\n\t\t\t\t\t\tMessage: "Step 5: has parent 4" + step4Out.Message,\n\t\t\t\t\t}, nil\n\t\t\t\t}).SetName("step-five").AddParents("step-four"),\n\t\t\t},\n\t\t},\n\t)\n\tif err != nil {\n\t\treturn fmt.Errorf("error registering workflow: %w", err)\n\t}\n\n\tcleanup, err := w.Start()\n\tif err != nil {\n\t\treturn fmt.Errorf("error starting worker: %w", err)\n\t}\n\n\t<-ch\n\n\tif err := cleanup(); err != nil {\n\t\treturn fmt.Errorf("error cleaning up: %w", err)\n\t}\n\n\treturn nil\n}\n',
      language: 'go',
      source: 'examples/go/z_v0/manual-trigger/worker/main.go',
      highlights: {},
    },
  L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL2dvL3pfdjAvbWlkZGxld2FyZS9tYWluLmdv:
    {
      content:
        'package main\n\nimport (\n\t"fmt"\n\n\t"github.com/joho/godotenv"\n\n\t"github.com/hatchet-dev/hatchet/pkg/cmdutils"\n)\n\ntype userCreateEvent struct {\n\tUsername string            `json:"username"`\n\tUserID   string            `json:"user_id"`\n\tData     map[string]string `json:"data"`\n}\n\ntype stepOneOutput struct {\n\tMessage string `json:"message"`\n}\n\nfunc main() {\n\terr := godotenv.Load()\n\tif err != nil {\n\t\tpanic(err)\n\t}\n\n\tevents := make(chan string, 50)\n\tch := cmdutils.InterruptChan()\n\tcleanup, err := run(events)\n\tif err != nil {\n\t\tpanic(err)\n\t}\n\n\t<-ch\n\n\tif err := cleanup(); err != nil {\n\t\tpanic(fmt.Errorf("cleanup() error = %v", err))\n\t}\n}\n',
      language: 'go',
      source: 'examples/go/z_v0/middleware/main.go',
      highlights: {},
    },
  L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL2dvL3pfdjAvbWlkZGxld2FyZS9tYWluX2UyZV90ZXN0Lmdv:
    {
      content:
        '//go:build e2e\n\npackage main\n\nimport (\n\t"context"\n\t"testing"\n\t"time"\n\n\t"github.com/stretchr/testify/assert"\n\n\t"github.com/hatchet-dev/hatchet/internal/testutils"\n)\n\nfunc TestMiddleware(t *testing.T) {\n\ttestutils.Prepare(t)\n\n\tctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)\n\tdefer cancel()\n\n\tevents := make(chan string, 50)\n\n\tcleanup, err := run(events)\n\tif err != nil {\n\t\tt.Fatalf("run() error = %s", err)\n\t}\n\n\tvar items []string\n\nouter:\n\tfor {\n\t\tselect {\n\t\tcase item := <-events:\n\t\t\titems = append(items, item)\n\t\tcase <-ctx.Done():\n\t\t\tbreak outer\n\t\t}\n\t}\n\n\tassert.Equal(t, []string{\n\t\t"1st-middleware",\n\t\t"2nd-middleware",\n\t\t"svc-middleware",\n\t\t"step-one",\n\t\t"testvalue",\n\t\t"svcvalue",\n\t\t"1st-middleware",\n\t\t"2nd-middleware",\n\t\t"svc-middleware",\n\t\t"step-two",\n\t}, items)\n\n\tif err := cleanup(); err != nil {\n\t\tt.Fatalf("cleanup() error = %s", err)\n\t}\n}\n',
      language: 'go',
      source: 'examples/go/z_v0/middleware/main_e2e_test.go',
      highlights: {},
    },
  L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL2dvL3pfdjAvbWlkZGxld2FyZS9ydW4uZ28_:
    {
      content:
        'package main\n\nimport (\n\t"context"\n\t"fmt"\n\t"log"\n\t"time"\n\n\t"github.com/hatchet-dev/hatchet/pkg/client"\n\t"github.com/hatchet-dev/hatchet/pkg/worker"\n)\n\nfunc run(events chan<- string) (func() error, error) {\n\tc, err := client.New()\n\tif err != nil {\n\t\treturn nil, fmt.Errorf("error creating client: %w", err)\n\t}\n\n\tw, err := worker.NewWorker(\n\t\tworker.WithClient(\n\t\t\tc,\n\t\t),\n\t)\n\tif err != nil {\n\t\treturn nil, fmt.Errorf("error creating worker: %w", err)\n\t}\n\n\tw.Use(func(ctx worker.HatchetContext, next func(worker.HatchetContext) error) error {\n\t\tlog.Printf("1st-middleware")\n\t\tevents <- "1st-middleware"\n\t\tctx.SetContext(context.WithValue(ctx.GetContext(), "testkey", "testvalue"))\n\t\treturn next(ctx)\n\t})\n\n\tw.Use(func(ctx worker.HatchetContext, next func(worker.HatchetContext) error) error {\n\t\tlog.Printf("2nd-middleware")\n\t\tevents <- "2nd-middleware"\n\n\t\t// time the function duration\n\t\tstart := time.Now()\n\t\terr := next(ctx)\n\t\tduration := time.Since(start)\n\t\tfmt.Printf("step function took %s\\n", duration)\n\t\treturn err\n\t})\n\n\ttestSvc := w.NewService("test")\n\n\ttestSvc.Use(func(ctx worker.HatchetContext, next func(worker.HatchetContext) error) error {\n\t\tevents <- "svc-middleware"\n\t\tctx.SetContext(context.WithValue(ctx.GetContext(), "svckey", "svcvalue"))\n\t\treturn next(ctx)\n\t})\n\n\terr = testSvc.On(\n\t\tworker.Events("user:create:middleware"),\n\t\t&worker.WorkflowJob{\n\t\t\tName:        "middleware",\n\t\t\tDescription: "This runs after an update to the user model.",\n\t\t\tSteps: []*worker.WorkflowStep{\n\t\t\t\tworker.Fn(func(ctx worker.HatchetContext) (result *stepOneOutput, err error) {\n\t\t\t\t\tinput := &userCreateEvent{}\n\n\t\t\t\t\terr = ctx.WorkflowInput(input)\n\n\t\t\t\t\tif err != nil {\n\t\t\t\t\t\treturn nil, err\n\t\t\t\t\t}\n\n\t\t\t\t\tlog.Printf("step-one")\n\t\t\t\t\tevents <- "step-one"\n\n\t\t\t\t\ttestVal := ctx.Value("testkey").(string)\n\t\t\t\t\tevents <- testVal\n\t\t\t\t\tsvcVal := ctx.Value("svckey").(string)\n\t\t\t\t\tevents <- svcVal\n\n\t\t\t\t\treturn &stepOneOutput{\n\t\t\t\t\t\tMessage: "Username is: " + input.Username,\n\t\t\t\t\t}, nil\n\t\t\t\t},\n\t\t\t\t).SetName("step-one"),\n\t\t\t\tworker.Fn(func(ctx worker.HatchetContext) (result *stepOneOutput, err error) {\n\t\t\t\t\tinput := &stepOneOutput{}\n\t\t\t\t\terr = ctx.StepOutput("step-one", input)\n\n\t\t\t\t\tif err != nil {\n\t\t\t\t\t\treturn nil, err\n\t\t\t\t\t}\n\n\t\t\t\t\tlog.Printf("step-two")\n\t\t\t\t\tevents <- "step-two"\n\n\t\t\t\t\treturn &stepOneOutput{\n\t\t\t\t\t\tMessage: "Above message is: " + input.Message,\n\t\t\t\t\t}, nil\n\t\t\t\t}).SetName("step-two").AddParents("step-one"),\n\t\t\t},\n\t\t},\n\t)\n\tif err != nil {\n\t\treturn nil, fmt.Errorf("error registering workflow: %w", err)\n\t}\n\n\tgo func() {\n\t\tlog.Printf("pushing event user:create:middleware")\n\n\t\ttestEvent := userCreateEvent{\n\t\t\tUsername: "echo-test",\n\t\t\tUserID:   "1234",\n\t\t\tData: map[string]string{\n\t\t\t\t"test": "test",\n\t\t\t},\n\t\t}\n\n\t\t// push an event\n\t\terr := c.Event().Push(\n\t\t\tcontext.Background(),\n\t\t\t"user:create:middleware",\n\t\t\ttestEvent,\n\t\t)\n\t\tif err != nil {\n\t\t\tpanic(fmt.Errorf("error pushing event: %w", err))\n\t\t}\n\t}()\n\n\tcleanup, err := w.Start()\n\tif err != nil {\n\t\treturn nil, fmt.Errorf("error starting worker: %w", err)\n\t}\n\n\treturn cleanup, nil\n}\n',
      language: 'go',
      source: 'examples/go/z_v0/middleware/run.go',
      highlights: {},
    },
  L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL2dvL3pfdjAvbmFtZXNwYWNlZC9tYWluLmdv:
    {
      content:
        'package main\n\nimport (\n\t"context"\n\t"fmt"\n\t"log"\n\n\t"github.com/joho/godotenv"\n\n\t"github.com/hatchet-dev/hatchet/pkg/client"\n\t"github.com/hatchet-dev/hatchet/pkg/cmdutils"\n\t"github.com/hatchet-dev/hatchet/pkg/worker"\n)\n\ntype userCreateEvent struct {\n\tUsername string            `json:"username"`\n\tUserID   string            `json:"user_id"`\n\tData     map[string]string `json:"data"`\n}\n\ntype stepOneOutput struct {\n\tMessage string `json:"message"`\n}\n\nfunc main() {\n\terr := godotenv.Load()\n\tif err != nil {\n\t\tpanic(err)\n\t}\n\n\tevents := make(chan string, 50)\n\tinterrupt := cmdutils.InterruptChan()\n\n\tcleanup, err := run(events)\n\tif err != nil {\n\t\tpanic(err)\n\t}\n\n\t<-interrupt\n\n\tif err := cleanup(); err != nil {\n\t\tpanic(fmt.Errorf("error cleaning up: %w", err))\n\t}\n}\n\nfunc getConcurrencyKey(ctx worker.HatchetContext) (string, error) {\n\treturn "user-create", nil\n}\n\nfunc run(events chan<- string) (func() error, error) {\n\tc, err := client.New(\n\t\tclient.WithNamespace("sample"),\n\t)\n\n\tif err != nil {\n\t\treturn nil, fmt.Errorf("error creating client: %w", err)\n\t}\n\n\tw, err := worker.NewWorker(\n\t\tworker.WithClient(\n\t\t\tc,\n\t\t),\n\t)\n\tif err != nil {\n\t\treturn nil, fmt.Errorf("error creating worker: %w", err)\n\t}\n\n\ttestSvc := w.NewService("test")\n\n\terr = testSvc.On(\n\t\tworker.Events("user:create:simple"),\n\t\t&worker.WorkflowJob{\n\t\t\tName:        "simple",\n\t\t\tDescription: "This runs after an update to the user model.",\n\t\t\tConcurrency: worker.Concurrency(getConcurrencyKey),\n\t\t\tSteps: []*worker.WorkflowStep{\n\t\t\t\tworker.Fn(func(ctx worker.HatchetContext) (result *stepOneOutput, err error) {\n\t\t\t\t\tinput := &userCreateEvent{}\n\n\t\t\t\t\terr = ctx.WorkflowInput(input)\n\n\t\t\t\t\tif err != nil {\n\t\t\t\t\t\treturn nil, err\n\t\t\t\t\t}\n\n\t\t\t\t\tlog.Printf("step-one")\n\t\t\t\t\tevents <- "step-one"\n\n\t\t\t\t\treturn &stepOneOutput{\n\t\t\t\t\t\tMessage: "Username is: " + input.Username,\n\t\t\t\t\t}, nil\n\t\t\t\t},\n\t\t\t\t).SetName("step-one"),\n\t\t\t\tworker.Fn(func(ctx worker.HatchetContext) (result *stepOneOutput, err error) {\n\t\t\t\t\tinput := &stepOneOutput{}\n\t\t\t\t\terr = ctx.StepOutput("step-one", input)\n\n\t\t\t\t\tif err != nil {\n\t\t\t\t\t\treturn nil, err\n\t\t\t\t\t}\n\n\t\t\t\t\tlog.Printf("step-two")\n\t\t\t\t\tevents <- "step-two"\n\n\t\t\t\t\treturn &stepOneOutput{\n\t\t\t\t\t\tMessage: "Above message is: " + input.Message,\n\t\t\t\t\t}, nil\n\t\t\t\t}).SetName("step-two").AddParents("step-one"),\n\t\t\t},\n\t\t},\n\t)\n\tif err != nil {\n\t\treturn nil, fmt.Errorf("error registering workflow: %w", err)\n\t}\n\n\tgo func() {\n\t\ttestEvent := userCreateEvent{\n\t\t\tUsername: "echo-test",\n\t\t\tUserID:   "1234",\n\t\t\tData: map[string]string{\n\t\t\t\t"test": "test",\n\t\t\t},\n\t\t}\n\n\t\tlog.Printf("pushing event user:create:simple")\n\t\t// push an event\n\t\terr := c.Event().Push(\n\t\t\tcontext.Background(),\n\t\t\t"user:create:simple",\n\t\t\ttestEvent,\n\t\t)\n\t\tif err != nil {\n\t\t\tpanic(fmt.Errorf("error pushing event: %w", err))\n\t\t}\n\t}()\n\n\tcleanup, err := w.Start()\n\tif err != nil {\n\t\tpanic(err)\n\t}\n\n\treturn cleanup, nil\n}\n',
      language: 'go',
      source: 'examples/go/z_v0/namespaced/main.go',
      highlights: {},
    },
  L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL2dvL3pfdjAvbm8tdGxzL21haW4uZ28_:
    {
      content:
        'package main\n\nimport (\n\t"fmt"\n\n\t"github.com/joho/godotenv"\n\n\t"github.com/hatchet-dev/hatchet/pkg/client"\n\t"github.com/hatchet-dev/hatchet/pkg/cmdutils"\n\t"github.com/hatchet-dev/hatchet/pkg/worker"\n)\n\ntype stepOutput struct{}\n\nfunc main() {\n\terr := godotenv.Load()\n\tif err != nil {\n\t\tpanic(err)\n\t}\n\n\tc, err := client.New()\n\n\tif err != nil {\n\t\tpanic(fmt.Sprintf("error creating client: %v", err))\n\t}\n\n\tw, err := worker.NewWorker(\n\t\tworker.WithClient(\n\t\t\tc,\n\t\t),\n\t\tworker.WithMaxRuns(1),\n\t)\n\tif err != nil {\n\t\tpanic(fmt.Sprintf("error creating worker: %v", err))\n\t}\n\n\ttestSvc := w.NewService("test")\n\n\terr = testSvc.On(\n\t\tworker.Events("simple"),\n\t\t&worker.WorkflowJob{\n\t\t\tName:        "simple-workflow",\n\t\t\tDescription: "Simple one-step workflow.",\n\t\t\tSteps: []*worker.WorkflowStep{\n\t\t\t\tworker.Fn(func(ctx worker.HatchetContext) (result *stepOutput, err error) {\n\t\t\t\t\tfmt.Println("executed step 1")\n\n\t\t\t\t\treturn &stepOutput{}, nil\n\t\t\t\t},\n\t\t\t\t).SetName("step-one"),\n\t\t\t},\n\t\t},\n\t)\n\tif err != nil {\n\t\tpanic(fmt.Sprintf("error registering workflow: %v", err))\n\t}\n\n\tinterruptCtx, cancel := cmdutils.InterruptContextFromChan(cmdutils.InterruptChan())\n\tdefer cancel()\n\n\tcleanup, err := w.Start()\n\tif err != nil {\n\t\tpanic(fmt.Sprintf("error starting worker: %v", err))\n\t}\n\n\t<-interruptCtx.Done()\n\tif err := cleanup(); err != nil {\n\t\tpanic(err)\n\t}\n}\n',
      language: 'go',
      source: 'examples/go/z_v0/no-tls/main.go',
      highlights: {},
    },
  L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL2dvL3pfdjAvb24tZmFpbHVyZS9tYWluLmdv:
    {
      content:
        'package main\n\nimport (\n\t"fmt"\n\t"time"\n\n\t"github.com/joho/godotenv"\n\n\t"github.com/hatchet-dev/hatchet/pkg/client"\n\t"github.com/hatchet-dev/hatchet/pkg/cmdutils"\n\t"github.com/hatchet-dev/hatchet/pkg/worker"\n)\n\ntype stepOneOutput struct {\n\tMessage string `json:"message"`\n}\n\n// ❓ OnFailure Step\n// This workflow will fail because the step will throw an error\n// we define an onFailure step to handle this case\n\nfunc StepOne(ctx worker.HatchetContext) (result *stepOneOutput, err error) {\n\t// 👀 this step will always raise an exception\n\treturn nil, fmt.Errorf("test on failure")\n}\n\nfunc OnFailure(ctx worker.HatchetContext) (result *stepOneOutput, err error) {\n\t// run cleanup code or notifications here\n\n\t// 👀 you can access the error from the failed step(s) like this\n\tfmt.Println(ctx.StepRunErrors())\n\n\treturn &stepOneOutput{\n\t\tMessage: "Failure!",\n\t}, nil\n}\n\nfunc main() {\n\t// ...\n\terr := godotenv.Load()\n\n\tif err != nil {\n\t\tpanic(err)\n\t}\n\n\tc, err := client.New()\n\n\tif err != nil {\n\t\tpanic(err)\n\t}\n\n\tw, err := worker.NewWorker(\n\t\tworker.WithClient(\n\t\t\tc,\n\t\t),\n\t)\n\n\tif err != nil {\n\t\tpanic(err)\n\t}\n\n\t// 👀 we define an onFailure step to handle this case\n\terr = w.On(\n\t\tworker.NoTrigger(),\n\t\t&worker.WorkflowJob{\n\t\t\tName:        "on-failure-workflow",\n\t\t\tDescription: "This runs at a scheduled time.",\n\t\t\tSteps: []*worker.WorkflowStep{\n\t\t\t\tworker.Fn(StepOne).SetName("step-one"),\n\t\t\t},\n\t\t\tOnFailure: &worker.WorkflowJob{\n\t\t\t\tName:        "scheduled-workflow-failure",\n\t\t\t\tDescription: "This runs when the scheduled workflow fails.",\n\t\t\t\tSteps: []*worker.WorkflowStep{\n\t\t\t\t\tworker.Fn(OnFailure).SetName("on-failure"),\n\t\t\t\t},\n\t\t\t},\n\t\t},\n\t)\n\n\t// ...\n\n\tif err != nil {\n\t\tpanic(err)\n\t}\n\n\tinterruptCtx, cancel := cmdutils.InterruptContextFromChan(cmdutils.InterruptChan())\n\tdefer cancel()\n\n\tcleanup, err := w.Start()\n\tif err != nil {\n\t\tpanic(fmt.Errorf("error cleaning up: %w", err))\n\t}\n\n\tfor {\n\t\tselect {\n\t\tcase <-interruptCtx.Done():\n\t\t\tif err := cleanup(); err != nil {\n\t\t\t\tpanic(fmt.Errorf("error cleaning up: %w", err))\n\t\t\t}\n\t\t\treturn\n\t\tdefault:\n\t\t\ttime.Sleep(time.Second)\n\t\t}\n\t}\n\t// ,\n}\n\n// ‼️\n',
      language: 'go',
      source: 'examples/go/z_v0/on-failure/main.go',
      highlights: {},
    },
  L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL2dvL3pfdjAvcHJvY2VkdXJhbC9tYWluLmdv:
    {
      content:
        'package main\n\nimport (\n\t"fmt"\n\t"sync"\n\t"time"\n\n\t"github.com/joho/godotenv"\n\t"golang.org/x/sync/errgroup"\n\n\t"github.com/hatchet-dev/hatchet/pkg/client"\n\t"github.com/hatchet-dev/hatchet/pkg/cmdutils"\n\t"github.com/hatchet-dev/hatchet/pkg/worker"\n)\n\nconst NUM_CHILDREN = 50\n\ntype proceduralChildInput struct {\n\tIndex int `json:"index"`\n}\n\ntype proceduralChildOutput struct {\n\tIndex int `json:"index"`\n}\n\ntype proceduralParentOutput struct {\n\tChildSum int `json:"child_sum"`\n}\n\nfunc main() {\n\terr := godotenv.Load()\n\tif err != nil {\n\t\tpanic(err)\n\t}\n\n\tevents := make(chan string, 5*NUM_CHILDREN)\n\tinterrupt := cmdutils.InterruptChan()\n\n\tcleanup, err := run(events)\n\tif err != nil {\n\t\tpanic(err)\n\t}\n\n\t<-interrupt\n\n\tif err := cleanup(); err != nil {\n\t\tpanic(fmt.Errorf("error cleaning up: %w", err))\n\t}\n}\n\nfunc run(events chan<- string) (func() error, error) {\n\tc, err := client.New()\n\n\tif err != nil {\n\t\treturn nil, fmt.Errorf("error creating client: %w", err)\n\t}\n\n\tw, err := worker.NewWorker(\n\t\tworker.WithClient(\n\t\t\tc,\n\t\t),\n\t)\n\tif err != nil {\n\t\treturn nil, fmt.Errorf("error creating worker: %w", err)\n\t}\n\n\ttestSvc := w.NewService("test")\n\n\terr = testSvc.On(\n\t\tworker.NoTrigger(),\n\t\t&worker.WorkflowJob{\n\t\t\tName:        "procedural-parent-workflow",\n\t\t\tDescription: "This is a test of procedural workflows.",\n\t\t\tSteps: []*worker.WorkflowStep{\n\t\t\t\tworker.Fn(\n\t\t\t\t\tfunc(ctx worker.HatchetContext) (result *proceduralParentOutput, err error) {\n\t\t\t\t\t\tchildWorkflows := make([]*client.Workflow, NUM_CHILDREN)\n\n\t\t\t\t\t\tfor i := 0; i < NUM_CHILDREN; i++ {\n\t\t\t\t\t\t\tchildInput := proceduralChildInput{\n\t\t\t\t\t\t\t\tIndex: i,\n\t\t\t\t\t\t\t}\n\n\t\t\t\t\t\t\tchildWorkflow, err := ctx.SpawnWorkflow("procedural-child-workflow", childInput, &worker.SpawnWorkflowOpts{\n\t\t\t\t\t\t\t\tAdditionalMetadata: &map[string]string{\n\t\t\t\t\t\t\t\t\t"childKey": "childValue",\n\t\t\t\t\t\t\t\t},\n\t\t\t\t\t\t\t})\n\n\t\t\t\t\t\t\tif err != nil {\n\t\t\t\t\t\t\t\treturn nil, err\n\t\t\t\t\t\t\t}\n\n\t\t\t\t\t\t\tchildWorkflows[i] = childWorkflow\n\n\t\t\t\t\t\t\tevents <- fmt.Sprintf("child-%d-started", i)\n\t\t\t\t\t\t}\n\n\t\t\t\t\t\teg := errgroup.Group{}\n\n\t\t\t\t\t\teg.SetLimit(NUM_CHILDREN)\n\n\t\t\t\t\t\tchildOutputs := make([]int, 0)\n\t\t\t\t\t\tchildOutputsMu := sync.Mutex{}\n\n\t\t\t\t\t\tfor i, childWorkflow := range childWorkflows {\n\t\t\t\t\t\t\teg.Go(func(i int, childWorkflow *client.Workflow) func() error {\n\t\t\t\t\t\t\t\treturn func() error {\n\t\t\t\t\t\t\t\t\tchildResult, err := childWorkflow.Result()\n\n\t\t\t\t\t\t\t\t\tif err != nil {\n\t\t\t\t\t\t\t\t\t\treturn err\n\t\t\t\t\t\t\t\t\t}\n\n\t\t\t\t\t\t\t\t\tchildOutput := proceduralChildOutput{}\n\n\t\t\t\t\t\t\t\t\terr = childResult.StepOutput("step-one", &childOutput)\n\n\t\t\t\t\t\t\t\t\tif err != nil {\n\t\t\t\t\t\t\t\t\t\treturn err\n\t\t\t\t\t\t\t\t\t}\n\n\t\t\t\t\t\t\t\t\tchildOutputsMu.Lock()\n\t\t\t\t\t\t\t\t\tchildOutputs = append(childOutputs, childOutput.Index)\n\t\t\t\t\t\t\t\t\tchildOutputsMu.Unlock()\n\n\t\t\t\t\t\t\t\t\tevents <- fmt.Sprintf("child-%d-completed", childOutput.Index)\n\n\t\t\t\t\t\t\t\t\treturn nil\n\n\t\t\t\t\t\t\t\t}\n\t\t\t\t\t\t\t}(i, childWorkflow))\n\t\t\t\t\t\t}\n\n\t\t\t\t\t\tfinishedCh := make(chan struct{})\n\n\t\t\t\t\t\tgo func() {\n\t\t\t\t\t\t\tdefer close(finishedCh)\n\t\t\t\t\t\t\terr = eg.Wait()\n\t\t\t\t\t\t}()\n\n\t\t\t\t\t\ttimer := time.NewTimer(60 * time.Second)\n\n\t\t\t\t\t\tselect {\n\t\t\t\t\t\tcase <-finishedCh:\n\t\t\t\t\t\t\tif err != nil {\n\t\t\t\t\t\t\t\treturn nil, err\n\t\t\t\t\t\t\t}\n\t\t\t\t\t\tcase <-timer.C:\n\t\t\t\t\t\t\tincomplete := make([]int, 0)\n\t\t\t\t\t\t\t// print non-complete children\n\t\t\t\t\t\t\tfor i := range childWorkflows {\n\t\t\t\t\t\t\t\tcompleted := false\n\t\t\t\t\t\t\t\tfor _, childOutput := range childOutputs {\n\t\t\t\t\t\t\t\t\tif childOutput == i {\n\t\t\t\t\t\t\t\t\t\tcompleted = true\n\t\t\t\t\t\t\t\t\t\tbreak\n\t\t\t\t\t\t\t\t\t}\n\t\t\t\t\t\t\t\t}\n\n\t\t\t\t\t\t\t\tif !completed {\n\t\t\t\t\t\t\t\t\tincomplete = append(incomplete, i)\n\t\t\t\t\t\t\t\t}\n\t\t\t\t\t\t\t}\n\n\t\t\t\t\t\t\treturn nil, fmt.Errorf("timed out waiting for the following child workflows to complete: %v", incomplete)\n\t\t\t\t\t\t}\n\n\t\t\t\t\t\tsum := 0\n\n\t\t\t\t\t\tfor _, childOutput := range childOutputs {\n\t\t\t\t\t\t\tsum += childOutput\n\t\t\t\t\t\t}\n\n\t\t\t\t\t\treturn &proceduralParentOutput{\n\t\t\t\t\t\t\tChildSum: sum,\n\t\t\t\t\t\t}, nil\n\t\t\t\t\t},\n\t\t\t\t).SetTimeout("10m"),\n\t\t\t},\n\t\t},\n\t)\n\n\tif err != nil {\n\t\treturn nil, fmt.Errorf("error registering workflow: %w", err)\n\t}\n\n\terr = testSvc.On(\n\t\tworker.NoTrigger(),\n\t\t&worker.WorkflowJob{\n\t\t\tName:        "procedural-child-workflow",\n\t\t\tDescription: "This is a test of procedural workflows.",\n\t\t\tSteps: []*worker.WorkflowStep{\n\t\t\t\tworker.Fn(\n\t\t\t\t\tfunc(ctx worker.HatchetContext) (result *proceduralChildOutput, err error) {\n\t\t\t\t\t\tinput := proceduralChildInput{}\n\n\t\t\t\t\t\terr = ctx.WorkflowInput(&input)\n\n\t\t\t\t\t\tif err != nil {\n\t\t\t\t\t\t\treturn nil, err\n\t\t\t\t\t\t}\n\n\t\t\t\t\t\treturn &proceduralChildOutput{\n\t\t\t\t\t\t\tIndex: input.Index,\n\t\t\t\t\t\t}, nil\n\t\t\t\t\t},\n\t\t\t\t).SetName("step-one"),\n\t\t\t},\n\t\t},\n\t)\n\n\tif err != nil {\n\t\treturn nil, fmt.Errorf("error registering workflow: %w", err)\n\t}\n\n\tgo func() {\n\t\ttime.Sleep(1 * time.Second)\n\n\t\t_, err := c.Admin().RunWorkflow("procedural-parent-workflow", nil)\n\n\t\tif err != nil {\n\t\t\tpanic(fmt.Errorf("error running workflow: %w", err))\n\t\t}\n\t}()\n\n\tcleanup, err := w.Start()\n\n\tif err != nil {\n\t\tpanic(err)\n\t}\n\n\treturn cleanup, nil\n}\n',
      language: 'go',
      source: 'examples/go/z_v0/procedural/main.go',
      highlights: {},
    },
  L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL2dvL3pfdjAvcHJvY2VkdXJhbC9tYWluX2UyZV90ZXN0Lmdv:
    {
      content:
        '//go:build e2e\n\npackage main\n\nimport (\n\t"context"\n\t"fmt"\n\t"testing"\n\t"time"\n\n\t"github.com/stretchr/testify/assert"\n\n\t"github.com/hatchet-dev/hatchet/internal/testutils"\n)\n\nfunc TestProcedural(t *testing.T) {\n\ttestutils.Prepare(t)\n\n\tctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)\n\tdefer cancel()\n\n\tevents := make(chan string, 5*NUM_CHILDREN)\n\n\tcleanup, err := run(events)\n\tif err != nil {\n\t\tt.Fatalf("/run() error = %v", err)\n\t}\n\n\tvar items []string\n\nouter:\n\tfor {\n\t\tselect {\n\t\tcase item := <-events:\n\t\t\titems = append(items, item)\n\t\tcase <-ctx.Done():\n\t\t\tbreak outer\n\t\t}\n\t}\n\n\texpected := []string{}\n\n\tfor i := 0; i < NUM_CHILDREN; i++ {\n\t\texpected = append(expected, fmt.Sprintf("child-%d-started", i))\n\t\texpected = append(expected, fmt.Sprintf("child-%d-completed", i))\n\t}\n\n\tassert.ElementsMatch(t, expected, items)\n\n\tif err := cleanup(); err != nil {\n\t\tt.Fatalf("cleanup() error = %v", err)\n\t}\n}\n',
      language: 'go',
      source: 'examples/go/z_v0/procedural/main_e2e_test.go',
      highlights: {},
    },
  L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL2dvL3pfdjAvcmF0ZS1saW1pdC9tYWluLmdv:
    {
      content:
        'package main\n\nimport (\n\t"fmt"\n\t"time"\n\n\t"github.com/joho/godotenv"\n\n\t"github.com/hatchet-dev/hatchet/pkg/client"\n\t"github.com/hatchet-dev/hatchet/pkg/client/types"\n\t"github.com/hatchet-dev/hatchet/pkg/cmdutils"\n\t"github.com/hatchet-dev/hatchet/pkg/worker"\n)\n\ntype rateLimitInput struct {\n\tIndex  int    `json:"index"`\n\tUserId string `json:"user_id"`\n}\n\ntype stepOneOutput struct {\n\tMessage string `json:"message"`\n}\n\nfunc StepOne(ctx worker.HatchetContext) (result *stepOneOutput, err error) {\n\tinput := &rateLimitInput{}\n\n\terr = ctx.WorkflowInput(input)\n\n\tif err != nil {\n\t\treturn nil, err\n\t}\n\n\tctx.StreamEvent([]byte(fmt.Sprintf("This is a stream event %d", input.Index)))\n\n\treturn &stepOneOutput{\n\t\tMessage: fmt.Sprintf("This ran at %s", time.Now().String()),\n\t}, nil\n}\n\nfunc main() {\n\terr := godotenv.Load()\n\n\tif err != nil {\n\t\tpanic(err)\n\t}\n\n\tc, err := client.New()\n\n\tif err != nil {\n\t\tpanic(err)\n\t}\n\n\terr = c.Admin().PutRateLimit("api1", &types.RateLimitOpts{\n\t\tMax:      12,\n\t\tDuration: types.Minute,\n\t})\n\n\tif err != nil {\n\t\tpanic(err)\n\t}\n\n\tw, err := worker.NewWorker(\n\t\tworker.WithClient(\n\t\t\tc,\n\t\t),\n\t)\n\n\tif err != nil {\n\t\tpanic(err)\n\t}\n\n\tunitExpr := "int(input.index) + 1"\n\tkeyExpr := "input.user_id"\n\tlimitValueExpr := "3"\n\n\terr = w.On(\n\t\tworker.NoTrigger(),\n\t\t&worker.WorkflowJob{\n\t\t\tName:        "rate-limit-workflow",\n\t\t\tDescription: "This illustrates rate limiting.",\n\t\t\tSteps: []*worker.WorkflowStep{\n\t\t\t\tworker.Fn(StepOne).SetName("step-one").SetRateLimit(\n\t\t\t\t\tworker.RateLimit{\n\t\t\t\t\t\tKey:            "per-user-rate-limit",\n\t\t\t\t\t\tKeyExpr:        &keyExpr,\n\t\t\t\t\t\tUnitsExpr:      &unitExpr,\n\t\t\t\t\t\tLimitValueExpr: &limitValueExpr,\n\t\t\t\t\t},\n\t\t\t\t),\n\t\t\t},\n\t\t},\n\t)\n\n\tif err != nil {\n\t\tpanic(err)\n\t}\n\n\tfor i := 0; i < 12; i++ {\n\t\tfor j := 0; j < 3; j++ {\n\t\t\t_, err = c.Admin().RunWorkflow("rate-limit-workflow", &rateLimitInput{\n\t\t\t\tIndex:  j,\n\t\t\t\tUserId: fmt.Sprintf("user-%d", i),\n\t\t\t})\n\n\t\t\tif err != nil {\n\t\t\t\tpanic(err)\n\t\t\t}\n\t\t}\n\t}\n\n\tinterrupt := cmdutils.InterruptChan()\n\n\tcleanup, err := w.Start()\n\tif err != nil {\n\t\tpanic(err)\n\t}\n\n\t<-interrupt\n\n\tif err := cleanup(); err != nil {\n\t\tpanic(fmt.Errorf("error cleaning up: %w", err))\n\t}\n}\n',
      language: 'go',
      source: 'examples/go/z_v0/rate-limit/main.go',
      highlights: {},
    },
  L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL2dvL3pfdjAvcmVnaXN0ZXItYWN0aW9uL21haW4uZ28_:
    {
      content:
        'package main\n\nimport (\n\t"context"\n\t"fmt"\n\t"time"\n\n\t"github.com/joho/godotenv"\n\n\t"github.com/hatchet-dev/hatchet/pkg/client"\n\t"github.com/hatchet-dev/hatchet/pkg/cmdutils"\n\t"github.com/hatchet-dev/hatchet/pkg/worker"\n)\n\ntype userCreateEvent struct {\n\tUsername string            `json:"username"`\n\tUserId   string            `json:"user_id"`\n\tData     map[string]string `json:"data"`\n}\n\ntype stepOneOutput struct {\n\tMessage string `json:"message"`\n}\n\nfunc StepOne(ctx context.Context, input *userCreateEvent) (result *stepOneOutput, err error) {\n\t// could get from context\n\t// testVal := ctx.Value("testkey").(string)\n\t// svcVal := ctx.Value("svckey").(string)\n\n\treturn &stepOneOutput{\n\t\tMessage: "Username is: " + input.Username,\n\t}, nil\n}\n\nfunc StepTwo(ctx context.Context, input *stepOneOutput) (result *stepOneOutput, err error) {\n\treturn &stepOneOutput{\n\t\tMessage: "Above message is: " + input.Message,\n\t}, nil\n}\n\nfunc main() {\n\terr := godotenv.Load()\n\n\tif err != nil {\n\t\tpanic(err)\n\t}\n\n\tclient, err := client.New()\n\n\tif err != nil {\n\t\tpanic(err)\n\t}\n\n\tw, err := worker.NewWorker(\n\t\tworker.WithClient(\n\t\t\tclient,\n\t\t),\n\t)\n\n\tif err != nil {\n\t\tpanic(err)\n\t}\n\n\ttestSvc := w.NewService("test")\n\n\ttestSvc.Use(func(ctx worker.HatchetContext, next func(worker.HatchetContext) error) error {\n\t\tctx.SetContext(context.WithValue(ctx.GetContext(), "testkey", "testvalue"))\n\t\treturn next(ctx)\n\t})\n\n\terr = testSvc.RegisterAction(StepOne, worker.WithActionName("step-one"))\n\n\tif err != nil {\n\t\tpanic(err)\n\t}\n\n\terr = testSvc.RegisterAction(StepTwo, worker.WithActionName("step-two"))\n\n\tif err != nil {\n\t\tpanic(err)\n\t}\n\n\terr = testSvc.On(\n\t\tworker.Events("user:create", "user:update"),\n\t\t&worker.WorkflowJob{\n\t\t\tName:        "post-user-update",\n\t\t\tDescription: "This runs after an update to the user model.",\n\t\t\tSteps: []*worker.WorkflowStep{\n\t\t\t\t// example of calling a registered action from the worker (includes service name)\n\t\t\t\tw.Call("test:step-one"),\n\t\t\t\t// example of calling a registered action from a service\n\t\t\t\ttestSvc.Call("step-two"),\n\t\t\t},\n\t\t},\n\t)\n\n\tif err != nil {\n\t\tpanic(err)\n\t}\n\n\t// err = worker.RegisterAction("echo:echo", func(ctx context.Context, input *actionInput) (result any, err error) {\n\t// \treturn map[string]interface{}{\n\t// \t\t"message": input.Message,\n\t// \t}, nil\n\t// })\n\n\t// if err != nil {\n\t// \tpanic(err)\n\t// }\n\n\t// err = worker.RegisterAction("echo:object", func(ctx context.Context, input *actionInput) (result any, err error) {\n\t// \treturn nil, nil\n\t// })\n\n\t// if err != nil {\n\t// \tpanic(err)\n\t// }\n\n\tinterrupt := cmdutils.InterruptChan()\n\n\tcleanup, err := w.Start()\n\tif err != nil {\n\t\tpanic(err)\n\t}\n\n\ttestEvent := userCreateEvent{\n\t\tUsername: "echo-test",\n\t\tUserId:   "1234",\n\t\tData: map[string]string{\n\t\t\t"test": "test",\n\t\t},\n\t}\n\n\t// push an event\n\terr = client.Event().Push(\n\t\tcontext.Background(),\n\t\t"user:create",\n\t\ttestEvent,\n\t)\n\n\tif err != nil {\n\t\tpanic(err)\n\t}\n\n\tfor {\n\t\tselect {\n\t\tcase <-interrupt:\n\t\t\tif err := cleanup(); err != nil {\n\t\t\t\tpanic(fmt.Errorf("error cleaning up: %w", err))\n\t\t\t}\n\t\tdefault:\n\t\t\ttime.Sleep(time.Second)\n\t\t}\n\t}\n}\n',
      language: 'go',
      source: 'examples/go/z_v0/register-action/main.go',
      highlights: {},
    },
  L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL2dvL3pfdjAvcmV0cmllcy9tYWluLmdv:
    {
      content:
        'package main\n\nimport (\n\t"context"\n\t"fmt"\n\t"log"\n\n\t"github.com/joho/godotenv"\n\n\t"github.com/hatchet-dev/hatchet/pkg/client"\n\t"github.com/hatchet-dev/hatchet/pkg/cmdutils"\n\t"github.com/hatchet-dev/hatchet/pkg/worker"\n)\n\ntype userCreateEvent struct {\n\tUsername string            `json:"username"`\n\tUserID   string            `json:"user_id"`\n\tData     map[string]string `json:"data"`\n}\n\ntype stepOneOutput struct {\n\tMessage string `json:"message"`\n}\n\nfunc main() {\n\terr := godotenv.Load()\n\tif err != nil {\n\t\tpanic(err)\n\t}\n\n\tevents := make(chan string, 50)\n\tif err := run(cmdutils.InterruptChan(), events); err != nil {\n\t\tpanic(err)\n\t}\n}\n\nfunc getConcurrencyKey(ctx worker.HatchetContext) (string, error) {\n\treturn "user-create", nil\n}\n\ntype retryWorkflow struct {\n\tretries int\n}\n\nfunc (r *retryWorkflow) StepOne(ctx worker.HatchetContext) (result *stepOneOutput, err error) {\n\tinput := &userCreateEvent{}\n\n\terr = ctx.WorkflowInput(input)\n\n\tif err != nil {\n\t\treturn nil, err\n\t}\n\n\tif r.retries < 2 {\n\t\tr.retries++\n\t\treturn nil, fmt.Errorf("error")\n\t}\n\n\tlog.Printf("finished step-one")\n\treturn &stepOneOutput{\n\t\tMessage: "Username is: " + input.Username,\n\t}, nil\n}\n\nfunc run(ch <-chan interface{}, events chan<- string) error {\n\tc, err := client.New()\n\n\tif err != nil {\n\t\treturn fmt.Errorf("error creating client: %w", err)\n\t}\n\n\tw, err := worker.NewWorker(\n\t\tworker.WithClient(\n\t\t\tc,\n\t\t),\n\t\tworker.WithMaxRuns(1),\n\t)\n\tif err != nil {\n\t\treturn fmt.Errorf("error creating worker: %w", err)\n\t}\n\n\ttestSvc := w.NewService("test")\n\n\twk := &retryWorkflow{}\n\n\terr = testSvc.On(\n\t\tworker.Events("user:create:simple"),\n\t\t&worker.WorkflowJob{\n\t\t\tName:        "simple",\n\t\t\tDescription: "This runs after an update to the user model.",\n\t\t\tConcurrency: worker.Concurrency(getConcurrencyKey),\n\t\t\tSteps: []*worker.WorkflowStep{\n\t\t\t\tworker.Fn(wk.StepOne).SetName("step-one").SetRetries(4),\n\t\t\t},\n\t\t},\n\t)\n\tif err != nil {\n\t\treturn fmt.Errorf("error registering workflow: %w", err)\n\t}\n\n\tcleanup, err := w.Start()\n\tif err != nil {\n\t\treturn fmt.Errorf("error starting worker: %w", err)\n\t}\n\n\ttestEvent := userCreateEvent{\n\t\tUsername: "echo-test",\n\t\tUserID:   "1234",\n\t\tData: map[string]string{\n\t\t\t"test": "test",\n\t\t},\n\t}\n\n\tlog.Printf("pushing event user:create:simple")\n\n\t// push an event\n\terr = c.Event().Push(\n\t\tcontext.Background(),\n\t\t"user:create:simple",\n\t\ttestEvent,\n\t)\n\n\tif err != nil {\n\t\treturn fmt.Errorf("error pushing event: %w", err)\n\t}\n\n\t<-ch\n\n\tif err := cleanup(); err != nil {\n\t\treturn fmt.Errorf("error cleaning up worker: %w", err)\n\t}\n\n\treturn nil\n}\n',
      language: 'go',
      source: 'examples/go/z_v0/retries/main.go',
      highlights: {},
    },
  L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL2dvL3pfdjAvcmV0cmllcy13aXRoLWJhY2tvZmYvbWFpbi5nbw__:
    {
      content:
        'package main\n\nimport (\n\t"fmt"\n\n\t"github.com/joho/godotenv"\n\n\t"github.com/hatchet-dev/hatchet/pkg/client"\n\t"github.com/hatchet-dev/hatchet/pkg/cmdutils"\n\t"github.com/hatchet-dev/hatchet/pkg/worker"\n)\n\ntype stepOneOutput struct {\n\tMessage string `json:"message"`\n}\n\n// ❓ Backoff\n\n// ... normal function definition\nfunc StepOne(ctx worker.HatchetContext) (result *stepOneOutput, err error) {\n\tif ctx.RetryCount() < 3 {\n\t\treturn nil, fmt.Errorf("failure")\n\t}\n\n\treturn &stepOneOutput{\n\t\tMessage: "Success!",\n\t}, nil\n}\n\n// ,\n\nfunc main() {\n\t// ...\n\terr := godotenv.Load()\n\n\tif err != nil {\n\t\tpanic(err)\n\t}\n\n\tc, err := client.New()\n\n\tif err != nil {\n\t\tpanic(err)\n\t}\n\n\tw, err := worker.NewWorker(\n\t\tworker.WithClient(\n\t\t\tc,\n\t\t),\n\t)\n\n\tif err != nil {\n\t\tpanic(err)\n\t}\n\n\t// ,\n\n\terr = w.RegisterWorkflow(\n\t\t&worker.WorkflowJob{\n\t\t\tName:        "retry-with-backoff-workflow",\n\t\t\tOn:          worker.NoTrigger(),\n\t\t\tDescription: "Demonstrates retry with exponential backoff.",\n\t\t\tSteps: []*worker.WorkflowStep{\n\t\t\t\tworker.Fn(StepOne).SetName("with-backoff").\n\t\t\t\t\tSetRetries(10).\n\t\t\t\t\t// 👀 Backoff configuration\n\t\t\t\t\t// 👀 Maximum number of seconds to wait between retries\n\t\t\t\t\tSetRetryBackoffFactor(2.0).\n\t\t\t\t\t// 👀 Factor to increase the wait time between retries.\n\t\t\t\t\t// This sequence will be 2s, 4s, 8s, 16s, 32s, 60s... due to the maxSeconds limit\n\t\t\t\t\tSetRetryMaxBackoffSeconds(60),\n\t\t\t},\n\t\t},\n\t)\n\n\t// ...\n\n\tif err != nil {\n\t\tpanic(err)\n\t}\n\n\tinterruptCtx, cancel := cmdutils.InterruptContextFromChan(cmdutils.InterruptChan())\n\tdefer cancel()\n\n\tcleanup, err := w.Start()\n\tif err != nil {\n\t\tpanic(fmt.Errorf("error cleaning up: %w", err))\n\t}\n\n\t<-interruptCtx.Done()\n\n\tif err := cleanup(); err != nil {\n\t\tpanic(fmt.Errorf("error cleaning up: %w", err))\n\t}\n\n\t// ,\n}\n\n// ‼️\n',
      language: 'go',
      source: 'examples/go/z_v0/retries-with-backoff/main.go',
      highlights: {},
    },
  L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL2dvL3pfdjAvc2NoZWR1bGVkL21haW4uZ28_:
    {
      content:
        'package main\n\nimport (\n\t"context"\n\t"fmt"\n\t"time"\n\n\t"github.com/joho/godotenv"\n\n\t"github.com/hatchet-dev/hatchet/pkg/client"\n\t"github.com/hatchet-dev/hatchet/pkg/cmdutils"\n\t"github.com/hatchet-dev/hatchet/pkg/worker"\n)\n\n// ❓ Create\n// ... normal workflow definition\ntype printOutput struct{}\n\nfunc print(ctx context.Context) (result *printOutput, err error) {\n\tfmt.Println("called print:print")\n\n\treturn &printOutput{}, nil\n}\n\n// ,\nfunc main() {\n\t// ... initialize client, worker and workflow\n\terr := godotenv.Load()\n\n\tif err != nil {\n\t\tpanic(err)\n\t}\n\n\tc, err := client.New()\n\n\tif err != nil {\n\t\tpanic(err)\n\t}\n\n\tw, err := worker.NewWorker(\n\t\tworker.WithClient(\n\t\t\tc,\n\t\t),\n\t)\n\n\tif err != nil {\n\t\tpanic(err)\n\t}\n\n\terr = w.RegisterWorkflow(\n\t\t&worker.WorkflowJob{\n\t\t\tOn:          worker.NoTrigger(),\n\t\t\tName:        "schedule-workflow",\n\t\t\tDescription: "Demonstrates a simple scheduled workflow",\n\t\t\tSteps: []*worker.WorkflowStep{\n\t\t\t\tworker.Fn(print),\n\t\t\t},\n\t\t},\n\t)\n\n\tif err != nil {\n\t\tpanic(err)\n\t}\n\n\tinterrupt := cmdutils.InterruptChan()\n\n\tcleanup, err := w.Start()\n\n\tif err != nil {\n\t\tpanic(err)\n\t}\n\n\t// ,\n\n\tgo func() {\n\t\t// 👀 define the scheduled workflow to run in a minute\n\t\tschedule, err := c.Schedule().Create(\n\t\t\tcontext.Background(),\n\t\t\t"schedule-workflow",\n\t\t\t&client.ScheduleOpts{\n\t\t\t\t// 👀 define the time to run the scheduled workflow, in UTC\n\t\t\t\tTriggerAt: time.Now().UTC().Add(time.Minute),\n\t\t\t\tInput: map[string]interface{}{\n\t\t\t\t\t"message": "Hello, world!",\n\t\t\t\t},\n\t\t\t\tAdditionalMetadata: map[string]string{},\n\t\t\t},\n\t\t)\n\n\t\tif err != nil {\n\t\t\tpanic(err)\n\t\t}\n\n\t\tfmt.Println(schedule.TriggerAt, schedule.WorkflowName)\n\t}()\n\n\t// ... wait for interrupt signal\n\n\t<-interrupt\n\n\tif err := cleanup(); err != nil {\n\t\tpanic(fmt.Errorf("error cleaning up: %w", err))\n\t}\n\n\t// ,\n}\n\n// !!\n\nfunc ListScheduledWorkflows() {\n\tc, err := client.New()\n\n\tif err != nil {\n\t\tpanic(err)\n\t}\n\n\t// ❓ List\n\tschedules, err := c.Schedule().List(context.Background())\n\t// !!\n\n\tif err != nil {\n\t\tpanic(err)\n\t}\n\n\tfor _, schedule := range *schedules.Rows {\n\t\tfmt.Println(schedule.TriggerAt, schedule.WorkflowName)\n\t}\n}\n\nfunc DeleteScheduledWorkflow(id string) {\n\tc, err := client.New()\n\n\tif err != nil {\n\t\tpanic(err)\n\t}\n\n\t// ❓ Delete\n\t// 👀 id is the schedule\'s metadata id, can get it via schedule.Metadata.Id\n\terr = c.Schedule().Delete(context.Background(), id)\n\t// !!\n\n\tif err != nil {\n\t\tpanic(err)\n\t}\n}\n',
      language: 'go',
      source: 'examples/go/z_v0/scheduled/main.go',
      highlights: {},
    },
  L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL2dvL3pfdjAvc2ltcGxlL21haW4uZ28_:
    {
      content:
        'package main\n\nimport (\n\t"context"\n\t"fmt"\n\t"log"\n\n\t"github.com/joho/godotenv"\n\n\t"github.com/hatchet-dev/hatchet/pkg/client"\n\t"github.com/hatchet-dev/hatchet/pkg/cmdutils"\n\t"github.com/hatchet-dev/hatchet/pkg/worker"\n)\n\ntype userCreateEvent struct {\n\tUsername string            `json:"username"`\n\tUserID   string            `json:"user_id"`\n\tData     map[string]string `json:"data"`\n}\n\ntype stepOneOutput struct {\n\tMessage string `json:"message"`\n}\n\nfunc main() {\n\terr := godotenv.Load()\n\tif err != nil {\n\t\tpanic(err)\n\t}\n\n\tevents := make(chan string, 50)\n\tinterrupt := cmdutils.InterruptChan()\n\n\tcleanup, err := run(events)\n\tif err != nil {\n\t\tpanic(err)\n\t}\n\n\t<-interrupt\n\n\tif err := cleanup(); err != nil {\n\t\tpanic(fmt.Errorf("error cleaning up: %w", err))\n\t}\n}\n\nfunc run(events chan<- string) (func() error, error) {\n\tc, err := client.New()\n\n\tif err != nil {\n\t\treturn nil, fmt.Errorf("error creating client: %w", err)\n\t}\n\n\tw, err := worker.NewWorker(\n\t\tworker.WithClient(\n\t\t\tc,\n\t\t),\n\t)\n\tif err != nil {\n\t\treturn nil, fmt.Errorf("error creating worker: %w", err)\n\t}\n\n\terr = w.RegisterWorkflow(\n\t\t&worker.WorkflowJob{\n\t\t\tOn:          worker.Events("user:create:simple"),\n\t\t\tName:        "simple",\n\t\t\tDescription: "This runs after an update to the user model.",\n\t\t\tConcurrency: worker.Expression("input.user_id"),\n\t\t\tSteps: []*worker.WorkflowStep{\n\t\t\t\tworker.Fn(func(ctx worker.HatchetContext) (result *stepOneOutput, err error) {\n\t\t\t\t\tinput := &userCreateEvent{}\n\n\t\t\t\t\terr = ctx.WorkflowInput(input)\n\n\t\t\t\t\tif err != nil {\n\t\t\t\t\t\treturn nil, err\n\t\t\t\t\t}\n\n\t\t\t\t\tlog.Printf("step-one")\n\t\t\t\t\tevents <- "step-one"\n\n\t\t\t\t\treturn &stepOneOutput{\n\t\t\t\t\t\tMessage: "Username is: " + input.Username,\n\t\t\t\t\t}, nil\n\t\t\t\t},\n\t\t\t\t).SetName("step-one"),\n\t\t\t\tworker.Fn(func(ctx worker.HatchetContext) (result *stepOneOutput, err error) {\n\t\t\t\t\tinput := &stepOneOutput{}\n\t\t\t\t\terr = ctx.StepOutput("step-one", input)\n\n\t\t\t\t\tif err != nil {\n\t\t\t\t\t\treturn nil, err\n\t\t\t\t\t}\n\n\t\t\t\t\tlog.Printf("step-two")\n\t\t\t\t\tevents <- "step-two"\n\n\t\t\t\t\treturn &stepOneOutput{\n\t\t\t\t\t\tMessage: "Above message is: " + input.Message,\n\t\t\t\t\t}, nil\n\t\t\t\t}).SetName("step-two").AddParents("step-one"),\n\t\t\t},\n\t\t},\n\t)\n\tif err != nil {\n\t\treturn nil, fmt.Errorf("error registering workflow: %w", err)\n\t}\n\n\tgo func() {\n\t\ttestEvent := userCreateEvent{\n\t\t\tUsername: "echo-test",\n\t\t\tUserID:   "1234",\n\t\t\tData: map[string]string{\n\t\t\t\t"test": "test",\n\t\t\t},\n\t\t}\n\n\t\tlog.Printf("pushing event user:create:simple")\n\t\t// push an event\n\t\terr := c.Event().Push(\n\t\t\tcontext.Background(),\n\t\t\t"user:create:simple",\n\t\t\ttestEvent,\n\t\t\tclient.WithEventMetadata(map[string]string{\n\t\t\t\t"hello": "world",\n\t\t\t}),\n\t\t)\n\t\tif err != nil {\n\t\t\tpanic(fmt.Errorf("error pushing event: %w", err))\n\t\t}\n\t}()\n\n\tcleanup, err := w.Start()\n\tif err != nil {\n\t\tpanic(err)\n\t}\n\n\treturn cleanup, nil\n}\n',
      language: 'go',
      source: 'examples/go/z_v0/simple/main.go',
      highlights: {},
    },
  L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL2dvL3pfdjAvc2ltcGxlL21haW5fZTJlX3Rlc3QuZ28_:
    {
      content:
        '//go:build e2e\n\npackage main\n\nimport (\n\t"context"\n\t"testing"\n\t"time"\n\n\t"github.com/stretchr/testify/assert"\n\n\t"github.com/hatchet-dev/hatchet/internal/testutils"\n)\n\nfunc TestSimple(t *testing.T) {\n\ttestutils.Prepare(t)\n\n\tctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)\n\tdefer cancel()\n\n\tevents := make(chan string, 50)\n\n\tcleanup, err := run(events)\n\tif err != nil {\n\t\tt.Fatalf("/run() error = %v", err)\n\t}\n\n\tvar items []string\n\nouter:\n\tfor {\n\t\tselect {\n\t\tcase item := <-events:\n\t\t\titems = append(items, item)\n\t\tcase <-ctx.Done():\n\t\t\tbreak outer\n\t\t}\n\t}\n\n\tassert.Equal(t, []string{\n\t\t"step-one",\n\t\t"step-two",\n\t}, items)\n\n\tif err := cleanup(); err != nil {\n\t\tt.Fatalf("cleanup() error = %v", err)\n\t}\n}\n',
      language: 'go',
      source: 'examples/go/z_v0/simple/main_e2e_test.go',
      highlights: {},
    },
  L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL2dvL3pfdjAvc3RyZWFtLWV2ZW50L21haW4uZ28_:
    {
      content:
        'package main\n\nimport (\n\t"fmt"\n\t"time"\n\n\t"github.com/joho/godotenv"\n\n\t"github.com/hatchet-dev/hatchet/pkg/client"\n\t"github.com/hatchet-dev/hatchet/pkg/cmdutils"\n\t"github.com/hatchet-dev/hatchet/pkg/worker"\n)\n\ntype streamEventInput struct {\n\tIndex int `json:"index"`\n}\n\ntype stepOneOutput struct {\n\tMessage string `json:"message"`\n}\n\nfunc StepOne(ctx worker.HatchetContext) (result *stepOneOutput, err error) {\n\tinput := &streamEventInput{}\n\n\terr = ctx.WorkflowInput(input)\n\n\tif err != nil {\n\t\treturn nil, err\n\t}\n\n\tctx.StreamEvent([]byte(fmt.Sprintf("This is a stream event %d", input.Index)))\n\n\treturn &stepOneOutput{\n\t\tMessage: fmt.Sprintf("This ran at %s", time.Now().String()),\n\t}, nil\n}\n\nfunc main() {\n\terr := godotenv.Load()\n\n\tif err != nil {\n\t\tpanic(err)\n\t}\n\n\tc, err := client.New()\n\n\tif err != nil {\n\t\tpanic(err)\n\t}\n\n\tw, err := worker.NewWorker(\n\t\tworker.WithClient(\n\t\t\tc,\n\t\t),\n\t)\n\n\tif err != nil {\n\t\tpanic(err)\n\t}\n\n\terr = w.On(\n\t\tworker.NoTrigger(),\n\t\t&worker.WorkflowJob{\n\t\t\tName:        "stream-event-workflow",\n\t\t\tDescription: "This sends a stream event.",\n\t\t\tSteps: []*worker.WorkflowStep{\n\t\t\t\tworker.Fn(StepOne).SetName("step-one"),\n\t\t\t},\n\t\t},\n\t)\n\n\tif err != nil {\n\t\tpanic(err)\n\t}\n\n\tinterruptCtx, cancel := cmdutils.InterruptContextFromChan(cmdutils.InterruptChan())\n\tdefer cancel()\n\n\t_, err = w.Start()\n\n\tif err != nil {\n\t\tpanic(fmt.Errorf("error cleaning up: %w", err))\n\t}\n\n\tworkflow, err := c.Admin().RunWorkflow("stream-event-workflow", &streamEventInput{\n\t\tIndex: 0,\n\t})\n\n\tif err != nil {\n\t\tpanic(err)\n\t}\n\n\terr = c.Subscribe().Stream(interruptCtx, workflow.WorkflowRunId(), func(event client.StreamEvent) error {\n\t\tfmt.Println(string(event.Message))\n\n\t\treturn nil\n\t})\n\n\tif err != nil {\n\t\tpanic(err)\n\t}\n}\n',
      language: 'go',
      source: 'examples/go/z_v0/stream-event/main.go',
      highlights: {},
    },
  L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL2dvL3pfdjAvc3RyZWFtLWV2ZW50LWJ5LW1ldGEvbWFpbi5nbw__:
    {
      content:
        'package main\n\nimport (\n\t"fmt"\n\t"time"\n\n\t"github.com/joho/godotenv"\n\t"golang.org/x/exp/rand"\n\n\t"github.com/hatchet-dev/hatchet/pkg/client"\n\t"github.com/hatchet-dev/hatchet/pkg/cmdutils"\n\t"github.com/hatchet-dev/hatchet/pkg/worker"\n)\n\ntype streamEventInput struct {\n\tIndex int `json:"index"`\n}\n\ntype stepOneOutput struct {\n\tMessage string `json:"message"`\n}\n\nfunc StepOne(ctx worker.HatchetContext) (result *stepOneOutput, err error) {\n\tinput := &streamEventInput{}\n\n\terr = ctx.WorkflowInput(input)\n\n\tif err != nil {\n\t\treturn nil, err\n\t}\n\n\tctx.StreamEvent([]byte(fmt.Sprintf("This is a stream event %d", input.Index)))\n\n\treturn &stepOneOutput{\n\t\tMessage: fmt.Sprintf("This ran at %s", time.Now().String()),\n\t}, nil\n}\n\nfunc main() {\n\terr := godotenv.Load()\n\n\tif err != nil {\n\t\tpanic(err)\n\t}\n\n\tc, err := client.New()\n\n\tif err != nil {\n\t\tpanic(err)\n\t}\n\n\tw, err := worker.NewWorker(\n\t\tworker.WithClient(\n\t\t\tc,\n\t\t),\n\t)\n\n\tif err != nil {\n\t\tpanic(err)\n\t}\n\n\terr = w.On(\n\t\tworker.NoTrigger(),\n\t\t&worker.WorkflowJob{\n\t\t\tName:        "stream-event-workflow",\n\t\t\tDescription: "This sends a stream event.",\n\t\t\tSteps: []*worker.WorkflowStep{\n\t\t\t\tworker.Fn(StepOne).SetName("step-one"),\n\t\t\t},\n\t\t},\n\t)\n\n\tif err != nil {\n\t\tpanic(err)\n\t}\n\n\tinterruptCtx, cancel := cmdutils.InterruptContextFromChan(cmdutils.InterruptChan())\n\tdefer cancel()\n\n\t_, err = w.Start()\n\n\tif err != nil {\n\t\tpanic(fmt.Errorf("error cleaning up: %w", err))\n\t}\n\n\t// Generate a random number between 1 and 100\n\tstreamKey := "streamKey"\n\tstreamValue := fmt.Sprintf("stream-event-%d", rand.Intn(100)+1)\n\n\t_, err = c.Admin().RunWorkflow("stream-event-workflow", &streamEventInput{\n\t\tIndex: 0,\n\t},\n\t\tclient.WithRunMetadata(map[string]interface{}{\n\t\t\tstreamKey: streamValue,\n\t\t}),\n\t)\n\n\tif err != nil {\n\t\tpanic(err)\n\t}\n\n\terr = c.Subscribe().StreamByAdditionalMetadata(interruptCtx, streamKey, streamValue, func(event client.StreamEvent) error {\n\t\tfmt.Println(string(event.Message))\n\t\treturn nil\n\t})\n\n\tif err != nil {\n\t\tpanic(err)\n\t}\n}\n',
      language: 'go',
      source: 'examples/go/z_v0/stream-event-by-meta/main.go',
      highlights: {},
    },
  L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL2dvL3pfdjAvdGltZW91dC9tYWluLmdv:
    {
      content:
        'package main\n\nimport (\n\t"fmt"\n\t"time"\n\n\t"github.com/joho/godotenv"\n\n\t"github.com/hatchet-dev/hatchet/pkg/worker"\n)\n\ntype userCreateEvent struct {\n\tUsername string            `json:"username"`\n\tUserID   string            `json:"user_id"`\n\tData     map[string]string `json:"data"`\n}\n\ntype stepOneOutput struct {\n\tMessage string `json:"message"`\n}\n\nfunc main() {\n\terr := godotenv.Load()\n\tif err != nil {\n\t\tpanic(err)\n\t}\n\n\tevents := make(chan string, 50)\n\n\t// ❓ TimeoutStep\n\tcleanup, err := run(events, worker.WorkflowJob{\n\t\tName:        "timeout",\n\t\tDescription: "timeout",\n\t\tSteps: []*worker.WorkflowStep{\n\t\t\tworker.Fn(func(ctx worker.HatchetContext) (result *stepOneOutput, err error) {\n\t\t\t\ttime.Sleep(time.Second * 60)\n\t\t\t\treturn nil, nil\n\t\t\t}).SetName("step-one").SetTimeout("10s"),\n\t\t},\n\t})\n\t// ‼️\n\tif err != nil {\n\t\tpanic(err)\n\t}\n\n\t<-events\n\n\tif err := cleanup(); err != nil {\n\t\tpanic(fmt.Errorf("cleanup() error = %v", err))\n\t}\n}\n',
      language: 'go',
      source: 'examples/go/z_v0/timeout/main.go',
      highlights: {},
    },
  L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL2dvL3pfdjAvdGltZW91dC9tYWluX2UyZV90ZXN0Lmdv:
    {
      content:
        '//go:build e2e\n\npackage main\n\nimport (\n\t"context"\n\t"testing"\n\t"time"\n\n\t"github.com/stretchr/testify/assert"\n\n\t"github.com/hatchet-dev/hatchet/internal/testutils"\n\t"github.com/hatchet-dev/hatchet/pkg/worker"\n)\n\nfunc TestTimeout(t *testing.T) {\n\ttestutils.Prepare(t)\n\n\ttests := []struct {\n\t\tname string\n\t\tjob  func(done func()) worker.WorkflowJob\n\t}{\n\t\t{\n\t\t\tname: "step timeout",\n\t\t\tjob: func(done func()) worker.WorkflowJob {\n\t\t\t\treturn worker.WorkflowJob{\n\t\t\t\t\tName:        "timeout",\n\t\t\t\t\tDescription: "timeout",\n\t\t\t\t\tSteps: []*worker.WorkflowStep{\n\t\t\t\t\t\tworker.Fn(func(ctx worker.HatchetContext) (result *stepOneOutput, err error) {\n\t\t\t\t\t\t\tselect {\n\t\t\t\t\t\t\tcase <-time.After(time.Second * 30):\n\t\t\t\t\t\t\t\treturn &stepOneOutput{\n\t\t\t\t\t\t\t\t\tMessage: "finished",\n\t\t\t\t\t\t\t\t}, nil\n\t\t\t\t\t\t\tcase <-ctx.Done():\n\t\t\t\t\t\t\t\tdone()\n\t\t\t\t\t\t\t\treturn nil, nil\n\t\t\t\t\t\t\t}\n\t\t\t\t\t\t}).SetName("step-one").SetTimeout("10s"),\n\t\t\t\t\t},\n\t\t\t\t}\n\t\t\t},\n\t\t},\n\t}\n\tfor _, tt := range tests {\n\t\tt.Run(tt.name, func(t *testing.T) {\n\t\t\tctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)\n\t\t\tdefer cancel()\n\n\t\t\tevents := make(chan string, 50)\n\n\t\t\tcleanup, err := run(events, tt.job(func() {\n\t\t\t\tevents <- "done"\n\t\t\t}))\n\t\t\tif err != nil {\n\t\t\t\tt.Fatalf("run() error = %s", err)\n\t\t\t}\n\n\t\t\tvar items []string\n\n\t\touter:\n\t\t\tfor {\n\t\t\t\tselect {\n\t\t\t\tcase item := <-events:\n\t\t\t\t\titems = append(items, item)\n\t\t\t\tcase <-ctx.Done():\n\t\t\t\t\tbreak outer\n\t\t\t\t}\n\t\t\t}\n\n\t\t\tassert.Equal(t, []string{\n\t\t\t\t"done", // cancellation signal\n\t\t\t\t"done", // test check\n\t\t\t}, items)\n\n\t\t\tif err := cleanup(); err != nil {\n\t\t\t\tt.Fatalf("cleanup() error = %s", err)\n\t\t\t}\n\t\t})\n\t}\n}\n',
      language: 'go',
      source: 'examples/go/z_v0/timeout/main_e2e_test.go',
      highlights: {},
    },
  L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL2dvL3pfdjAvdGltZW91dC9ydW4uZ28_:
    {
      content:
        'package main\n\nimport (\n\t"context"\n\t"fmt"\n\t"log"\n\t"time"\n\n\t"github.com/hatchet-dev/hatchet/pkg/client"\n\t"github.com/hatchet-dev/hatchet/pkg/worker"\n)\n\nfunc run(done chan<- string, job worker.WorkflowJob) (func() error, error) {\n\tc, err := client.New()\n\tif err != nil {\n\t\treturn nil, fmt.Errorf("error creating client: %w", err)\n\t}\n\n\tw, err := worker.NewWorker(\n\t\tworker.WithClient(\n\t\t\tc,\n\t\t),\n\t)\n\tif err != nil {\n\t\treturn nil, fmt.Errorf("error creating worker: %w", err)\n\t}\n\n\terr = w.On(\n\t\tworker.Events("user:create:timeout"),\n\t\t&job,\n\t)\n\tif err != nil {\n\t\treturn nil, fmt.Errorf("error registering workflow: %w", err)\n\t}\n\n\tgo func() {\n\t\tlog.Printf("pushing event")\n\n\t\ttestEvent := userCreateEvent{\n\t\t\tUsername: "echo-test",\n\t\t\tUserID:   "1234",\n\t\t\tData: map[string]string{\n\t\t\t\t"test": "test",\n\t\t\t},\n\t\t}\n\n\t\t// push an event\n\t\terr := c.Event().Push(\n\t\t\tcontext.Background(),\n\t\t\t"user:create:timeout",\n\t\t\ttestEvent,\n\t\t)\n\t\tif err != nil {\n\t\t\tpanic(fmt.Errorf("error pushing event: %w", err))\n\t\t}\n\n\t\ttime.Sleep(20 * time.Second)\n\n\t\tdone <- "done"\n\t}()\n\n\tcleanup, err := w.Start()\n\tif err != nil {\n\t\treturn nil, fmt.Errorf("error starting worker: %w", err)\n\t}\n\n\treturn cleanup, nil\n}\n',
      language: 'go',
      source: 'examples/go/z_v0/timeout/run.go',
      highlights: {},
    },
  L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL2dvL3pfdjAvd2ViaG9vay9tYWluLmdv:
    {
      content:
        'package main\n\nimport (\n\t"fmt"\n\t"log"\n\n\t"github.com/joho/godotenv"\n\n\t"github.com/hatchet-dev/hatchet/pkg/client"\n\t"github.com/hatchet-dev/hatchet/pkg/worker"\n)\n\ntype userCreateEvent struct {\n\tUsername string            `json:"username"`\n\tUserID   string            `json:"user_id"`\n\tData     map[string]string `json:"data"`\n}\n\ntype output struct {\n\tMessage string `json:"message"`\n}\n\nfunc main() {\n\terr := godotenv.Load()\n\tif err != nil {\n\t\tpanic(err)\n\t}\n\n\tc, err := client.New()\n\tif err != nil {\n\t\tpanic(fmt.Errorf("error creating client: %w", err))\n\t}\n\n\tw, err := worker.NewWorker(\n\t\tworker.WithClient(\n\t\t\tc,\n\t\t),\n\t)\n\tif err != nil {\n\t\tpanic(fmt.Errorf("error creating worker: %w", err))\n\t}\n\n\tworkflow := "webhook"\n\tevent := "user:create:webhook"\n\twf := &worker.WorkflowJob{\n\t\tName:        workflow,\n\t\tDescription: workflow,\n\t\tSteps: []*worker.WorkflowStep{\n\t\t\tworker.Fn(func(ctx worker.HatchetContext) (result *output, err error) {\n\t\t\t\tlog.Printf("step name: %s", ctx.StepName())\n\t\t\t\treturn &output{\n\t\t\t\t\tMessage: "hi from " + ctx.StepName(),\n\t\t\t\t}, nil\n\t\t\t}).SetName("webhook-step-one").SetTimeout("10s"),\n\t\t\tworker.Fn(func(ctx worker.HatchetContext) (result *output, err error) {\n\t\t\t\tlog.Printf("step name: %s", ctx.StepName())\n\t\t\t\treturn &output{\n\t\t\t\t\tMessage: "hi from " + ctx.StepName(),\n\t\t\t\t}, nil\n\t\t\t}).SetName("webhook-step-one").SetTimeout("10s"),\n\t\t},\n\t}\n\n\thandler := w.WebhookHttpHandler(worker.WebhookHandlerOptions{\n\t\tSecret: "secret",\n\t}, wf)\n\tport := "8741"\n\terr = run("webhook-demo", w, port, handler, c, workflow, event)\n\tif err != nil {\n\t\tpanic(err)\n\t}\n}\n',
      language: 'go',
      source: 'examples/go/z_v0/webhook/main.go',
      highlights: {},
    },
  L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL2dvL3pfdjAvd2ViaG9vay9tYWluX2UyZV90ZXN0Lmdv:
    {
      content:
        '//go:build e2e\n\npackage main\n\nimport (\n\t"context"\n\t"fmt"\n\t"net/http"\n\t"testing"\n\t"time"\n\n\t"github.com/stretchr/testify/assert"\n\n\t"github.com/hatchet-dev/hatchet/internal/testutils"\n\t"github.com/hatchet-dev/hatchet/pkg/client"\n\t"github.com/hatchet-dev/hatchet/pkg/worker"\n)\n\nfunc TestWebhook(t *testing.T) {\n\ttestutils.Prepare(t)\n\n\tc, err := client.New()\n\tif err != nil {\n\t\tpanic(fmt.Errorf("error creating client: %w", err))\n\t}\n\n\ttests := []struct {\n\t\tname string\n\t\tjob  func(t *testing.T)\n\t}{\n\t\t{\n\t\t\tname: "simple action",\n\t\t\tjob: func(t *testing.T) {\n\t\t\t\tevents := make(chan string, 10)\n\n\t\t\t\tctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)\n\t\t\t\tdefer cancel()\n\n\t\t\t\tevent := "user:webhook-simple"\n\t\t\t\tworkflow := "simple-webhook"\n\t\t\t\twf := &worker.WorkflowJob{\n\t\t\t\t\tOn:          worker.Event(event),\n\t\t\t\t\tName:        workflow,\n\t\t\t\t\tDescription: workflow,\n\t\t\t\t\tSteps: []*worker.WorkflowStep{\n\t\t\t\t\t\tworker.Fn(func(ctx worker.HatchetContext) (*output, error) {\n\t\t\t\t\t\t\t//verifyStepRuns(client, c.TenantId(), db.JobRunStatusRunning, db.StepRunStatusRunning, nil)\n\n\t\t\t\t\t\t\tevents <- "webhook-step-one"\n\n\t\t\t\t\t\t\treturn &output{\n\t\t\t\t\t\t\t\tMessage: "hi from " + ctx.StepName(),\n\t\t\t\t\t\t\t}, nil\n\t\t\t\t\t\t}).SetName("webhook-step-one").SetTimeout("60s"),\n\t\t\t\t\t\tworker.Fn(func(ctx worker.HatchetContext) (*output, error) {\n\t\t\t\t\t\t\tvar out output\n\t\t\t\t\t\t\tif err := ctx.StepOutput("webhook-step-one", &out); err != nil {\n\t\t\t\t\t\t\t\tpanic(err)\n\t\t\t\t\t\t\t}\n\t\t\t\t\t\t\tif out.Message != "hi from webhook-step-one" {\n\t\t\t\t\t\t\t\tpanic(fmt.Errorf("expected step run output to be valid, got %s", out.Message))\n\t\t\t\t\t\t\t}\n\n\t\t\t\t\t\t\tevents <- "webhook-step-two"\n\n\t\t\t\t\t\t\t//verifyStepRuns(client, c.TenantId(), db.JobRunStatusRunning, db.StepRunStatusRunning, nil)\n\n\t\t\t\t\t\t\treturn &output{\n\t\t\t\t\t\t\t\tMessage: "hi from " + ctx.StepName(),\n\t\t\t\t\t\t\t}, nil\n\t\t\t\t\t\t}).SetName("webhook-step-two").SetTimeout("60s").AddParents("webhook-step-one"),\n\t\t\t\t\t},\n\t\t\t\t}\n\n\t\t\t\tw, err := worker.NewWorker(\n\t\t\t\t\tworker.WithClient(\n\t\t\t\t\t\tc,\n\t\t\t\t\t),\n\t\t\t\t)\n\t\t\t\tif err != nil {\n\t\t\t\t\tpanic(fmt.Errorf("error creating worker: %w", err))\n\t\t\t\t}\n\n\t\t\t\thandler := w.WebhookHttpHandler(worker.WebhookHandlerOptions{\n\t\t\t\t\tSecret: "secret",\n\t\t\t\t}, wf)\n\t\t\t\terr = run("simple action", w, "8742", handler, c, workflow, event)\n\t\t\t\tif err != nil {\n\t\t\t\t\tt.Fatalf("run() error = %s", err)\n\t\t\t\t}\n\n\t\t\t\tvar items []string\n\t\t\touter:\n\t\t\t\tfor {\n\t\t\t\t\tselect {\n\t\t\t\t\tcase item := <-events:\n\t\t\t\t\t\titems = append(items, item)\n\t\t\t\t\tcase <-ctx.Done():\n\t\t\t\t\t\tbreak outer\n\t\t\t\t\t}\n\t\t\t\t}\n\n\t\t\t\tassert.Equal(t, []string{\n\t\t\t\t\t"webhook-step-one",\n\t\t\t\t\t"webhook-step-two",\n\t\t\t\t}, items)\n\t\t\t},\n\t\t},\n\t\t{\n\t\t\tname: "mark action as failed immediately if webhook fails",\n\t\t\tjob: func(t *testing.T) {\n\t\t\t\tworkflow := "simple-webhook-failure"\n\t\t\t\twf := &worker.WorkflowJob{\n\t\t\t\t\tName:        workflow,\n\t\t\t\t\tDescription: workflow,\n\t\t\t\t\tSteps: []*worker.WorkflowStep{\n\t\t\t\t\t\tworker.Fn(func(ctx worker.HatchetContext) (*output, error) {\n\t\t\t\t\t\t\treturn &output{\n\t\t\t\t\t\t\t\tMessage: "hi from " + ctx.StepName(),\n\t\t\t\t\t\t\t}, nil\n\t\t\t\t\t\t}).SetName("webhook-failure-step-one").SetTimeout("60s"),\n\t\t\t\t\t},\n\t\t\t\t}\n\n\t\t\t\tw, err := worker.NewWorker(\n\t\t\t\t\tworker.WithClient(\n\t\t\t\t\t\tc,\n\t\t\t\t\t),\n\t\t\t\t)\n\t\t\t\tif err != nil {\n\t\t\t\t\tpanic(fmt.Errorf("error creating worker: %w", err))\n\t\t\t\t}\n\n\t\t\t\tevent := "user:create-webhook-failure"\n\t\t\t\terr = w.On(worker.Events(event), wf)\n\t\t\t\tif err != nil {\n\t\t\t\t\tpanic(fmt.Errorf("error registering webhook workflow: %w", err))\n\t\t\t\t}\n\t\t\t\thandler := func(w http.ResponseWriter, r *http.Request) {\n\t\t\t\t\tif r.Method == http.MethodPut {\n\t\t\t\t\t\tw.WriteHeader(http.StatusOK)\n\t\t\t\t\t\t_, _ = w.Write([]byte(fmt.Sprintf(`{"actions": ["default:%s"]}`, "webhook-failure-step-one")))\n\t\t\t\t\t\treturn\n\t\t\t\t\t}\n\t\t\t\t\tw.WriteHeader(http.StatusInternalServerError) // simulate a failure\n\t\t\t\t}\n\t\t\t\terr = run("mark action as failed immediately if webhook fails", w, "8743", handler, c, workflow, event)\n\t\t\t\tif err != nil {\n\t\t\t\t\tt.Fatalf("run() error = %s", err)\n\t\t\t\t}\n\t\t\t},\n\t\t},\n\t\t{\n\t\t\tname: "register action",\n\t\t\tjob: func(t *testing.T) {\n\t\t\t\tevents := make(chan string, 10)\n\n\t\t\t\tctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)\n\t\t\t\tdefer cancel()\n\n\t\t\t\tw, err := worker.NewWorker(\n\t\t\t\t\tworker.WithClient(\n\t\t\t\t\t\tc,\n\t\t\t\t\t),\n\t\t\t\t)\n\t\t\t\tif err != nil {\n\t\t\t\t\tpanic(fmt.Errorf("error creating worker: %w", err))\n\t\t\t\t}\n\n\t\t\t\ttestSvc := w.NewService("test")\n\n\t\t\t\terr = testSvc.RegisterAction(func(ctx worker.HatchetContext) (*output, error) {\n\t\t\t\t\ttime.Sleep(5 * time.Second)\n\n\t\t\t\t\tevents <- "wha-webhook-action-1"\n\n\t\t\t\t\treturn &output{\n\t\t\t\t\t\tMessage: "hi from wha-webhook-action-1",\n\t\t\t\t\t}, nil\n\t\t\t\t}, worker.WithActionName("wha-webhook-action-1"))\n\t\t\t\tif err != nil {\n\t\t\t\t\tpanic(err)\n\t\t\t\t}\n\n\t\t\t\tevent := "user:wha-webhook-actions"\n\n\t\t\t\terr = testSvc.On(\n\t\t\t\t\tworker.Event(event),\n\t\t\t\t\ttestSvc.Call("wha-webhook-action-1"),\n\t\t\t\t)\n\n\t\t\t\tworkflow := "wha-webhook-with-actions"\n\t\t\t\twf := &worker.WorkflowJob{\n\t\t\t\t\tOn:          worker.Event(event),\n\t\t\t\t\tName:        workflow,\n\t\t\t\t\tDescription: workflow,\n\t\t\t\t\tSteps: []*worker.WorkflowStep{\n\t\t\t\t\t\tworker.Fn(func(ctx worker.HatchetContext) (*output, error) {\n\t\t\t\t\t\t\t//verifyStepRuns(client, c.TenantId(), db.JobRunStatusRunning, db.StepRunStatusRunning, nil)\n\n\t\t\t\t\t\t\tevents <- "wha-webhook-step-one"\n\n\t\t\t\t\t\t\treturn &output{\n\t\t\t\t\t\t\t\tMessage: "hi from " + ctx.StepName(),\n\t\t\t\t\t\t\t}, nil\n\t\t\t\t\t\t}).SetName("wha-webhook-step-one").SetTimeout("60s"),\n\t\t\t\t\t\tworker.Fn(func(ctx worker.HatchetContext) (*output, error) {\n\t\t\t\t\t\t\tvar out output\n\t\t\t\t\t\t\tif err := ctx.StepOutput("wha-webhook-step-one", &out); err != nil {\n\t\t\t\t\t\t\t\tpanic(err)\n\t\t\t\t\t\t\t}\n\t\t\t\t\t\t\tif out.Message != "hi from wha-webhook-step-one" {\n\t\t\t\t\t\t\t\tpanic(fmt.Errorf("expected step run output to be valid, got %s", out.Message))\n\t\t\t\t\t\t\t}\n\n\t\t\t\t\t\t\tevents <- "wha-webhook-step-two"\n\n\t\t\t\t\t\t\t//verifyStepRuns(client, c.TenantId(), db.JobRunStatusRunning, db.StepRunStatusRunning, nil)\n\n\t\t\t\t\t\t\treturn &output{\n\t\t\t\t\t\t\t\tMessage: "hi from " + ctx.StepName(),\n\t\t\t\t\t\t\t}, nil\n\t\t\t\t\t\t}).SetName("wha-webhook-step-two").SetTimeout("60s").AddParents("wha-webhook-step-one"),\n\t\t\t\t\t},\n\t\t\t\t}\n\n\t\t\t\thandler := w.WebhookHttpHandler(worker.WebhookHandlerOptions{\n\t\t\t\t\tSecret: "secret",\n\t\t\t\t}, wf)\n\t\t\t\terr = run("register action", w, "8744", handler, c, workflow, event)\n\t\t\t\tif err != nil {\n\t\t\t\t\tt.Fatalf("run() error = %s", err)\n\t\t\t\t}\n\n\t\t\t\tvar items []string\n\t\t\touter:\n\t\t\t\tfor {\n\t\t\t\t\tselect {\n\t\t\t\t\tcase item := <-events:\n\t\t\t\t\t\titems = append(items, item)\n\t\t\t\t\tcase <-ctx.Done():\n\t\t\t\t\t\tbreak outer\n\t\t\t\t\t}\n\t\t\t\t}\n\n\t\t\t\tassert.Equal(t, []string{\n\t\t\t\t\t"wha-webhook-step-one",\n\t\t\t\t\t"wha-webhook-step-two",\n\t\t\t\t\t"wha-webhook-action-1",\n\t\t\t\t}, items)\n\t\t\t},\n\t\t},\n\t}\n\tfor _, tt := range tests {\n\t\tt.Run(tt.name, func(t *testing.T) {\n\t\t\ttt.job(t)\n\t\t})\n\t}\n}\n',
      language: 'go',
      source: 'examples/go/z_v0/webhook/main_e2e_test.go',
      highlights: {},
    },
  L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL2dvL3pfdjAvd2ViaG9vay9ydW4uZ28_:
    {
      content:
        'package main\n\nimport (\n\t"context"\n\t"errors"\n\t"fmt"\n\t"log"\n\t"net/http"\n\t"time"\n\n\t"github.com/hatchet-dev/hatchet/pkg/client"\n\t"github.com/hatchet-dev/hatchet/pkg/worker"\n)\n\nfunc run(\n\tname string,\n\tw *worker.Worker,\n\tport string,\n\thandler func(w http.ResponseWriter, r *http.Request), c client.Client, workflow string, event string,\n) error {\n\t// create webserver to handle webhook requests\n\tmux := http.NewServeMux()\n\n\t// Register the HelloHandler to the /hello route\n\tmux.HandleFunc("/webhook", handler)\n\n\t// Create a custom server\n\tserver := &http.Server{\n\t\tAddr:         ":" + port,\n\t\tHandler:      mux,\n\t\tReadTimeout:  10 * time.Second,\n\t\tWriteTimeout: 10 * time.Second,\n\t\tIdleTimeout:  15 * time.Second,\n\t}\n\n\tdefer func(server *http.Server, ctx context.Context) {\n\t\terr := server.Shutdown(ctx)\n\t\tif err != nil {\n\t\t\tpanic(err)\n\t\t}\n\t}(server, context.Background())\n\n\tgo func() {\n\t\tif err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {\n\t\t\tpanic(err)\n\t\t}\n\t}()\n\n\tsecret := "secret"\n\tif err := w.RegisterWebhook(worker.RegisterWebhookWorkerOpts{\n\t\tName:   "test-" + name,\n\t\tURL:    fmt.Sprintf("http://localhost:%s/webhook", port),\n\t\tSecret: &secret,\n\t}); err != nil {\n\t\treturn fmt.Errorf("error setting up webhook: %w", err)\n\t}\n\n\ttime.Sleep(30 * time.Second)\n\n\tlog.Printf("pushing event")\n\n\ttestEvent := userCreateEvent{\n\t\tUsername: "echo-test",\n\t\tUserID:   "1234",\n\t\tData: map[string]string{\n\t\t\t"test": "test",\n\t\t},\n\t}\n\n\t// push an event\n\terr := c.Event().Push(\n\t\tcontext.Background(),\n\t\tevent,\n\t\ttestEvent,\n\t)\n\tif err != nil {\n\t\treturn fmt.Errorf("error pushing event: %w", err)\n\t}\n\n\ttime.Sleep(5 * time.Second)\n\n\treturn nil\n}\n',
      language: 'go',
      source: 'examples/go/z_v0/webhook/run.go',
      highlights: {},
    },
} as const;

// Snippet mapping
const snips = {
  typescript: {
    cancellations: {
      run: {
        '*': ':L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3R5cGVzY3JpcHQvY2FuY2VsbGF0aW9ucy9ydW4udHM_',
        running_a_task_with_results:
          'Running a Task with Results:L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3R5cGVzY3JpcHQvY2FuY2VsbGF0aW9ucy9ydW4udHM_',
      },
      worker: {
        '*': ':L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3R5cGVzY3JpcHQvY2FuY2VsbGF0aW9ucy93b3JrZXIudHM_',
        declaring_a_worker:
          'Declaring a Worker:L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3R5cGVzY3JpcHQvY2FuY2VsbGF0aW9ucy93b3JrZXIudHM_',
      },
      workflow: {
        '*': ':L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3R5cGVzY3JpcHQvY2FuY2VsbGF0aW9ucy93b3JrZmxvdy50cw__',
        declaring_a_task:
          'Declaring a Task:L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3R5cGVzY3JpcHQvY2FuY2VsbGF0aW9ucy93b3JrZmxvdy50cw__',
        abort_signal:
          'Abort Signal:L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3R5cGVzY3JpcHQvY2FuY2VsbGF0aW9ucy93b3JrZmxvdy50cw__',
      },
    },
    child_workflows: {
      run: {
        '*': ':L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3R5cGVzY3JpcHQvY2hpbGRfd29ya2Zsb3dzL3J1bi50cw__',
      },
      worker: {
        '*': ':L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3R5cGVzY3JpcHQvY2hpbGRfd29ya2Zsb3dzL3dvcmtlci50cw__',
      },
      workflow: {
        '*': ':L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3R5cGVzY3JpcHQvY2hpbGRfd29ya2Zsb3dzL3dvcmtmbG93LnRz',
        declaring_a_child:
          'Declaring a Child:L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3R5cGVzY3JpcHQvY2hpbGRfd29ya2Zsb3dzL3dvcmtmbG93LnRz',
        declaring_a_parent:
          'Declaring a Parent:L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3R5cGVzY3JpcHQvY2hpbGRfd29ya2Zsb3dzL3dvcmtmbG93LnRz',
      },
    },
    concurrency_rr: {
      load: {
        '*': ':L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3R5cGVzY3JpcHQvY29uY3VycmVuY3ktcnIvbG9hZC50cw__',
      },
      run: {
        '*': ':L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3R5cGVzY3JpcHQvY29uY3VycmVuY3ktcnIvcnVuLnRz',
      },
      worker: {
        '*': ':L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3R5cGVzY3JpcHQvY29uY3VycmVuY3ktcnIvd29ya2VyLnRz',
      },
      workflow: {
        '*': ':L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3R5cGVzY3JpcHQvY29uY3VycmVuY3ktcnIvd29ya2Zsb3cudHM_',
        concurrency_strategy_with_key:
          'Concurrency Strategy With Key:L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3R5cGVzY3JpcHQvY29uY3VycmVuY3ktcnIvd29ya2Zsb3cudHM_',
        multiple_concurrency_keys:
          'Multiple Concurrency Keys:L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3R5cGVzY3JpcHQvY29uY3VycmVuY3ktcnIvd29ya2Zsb3cudHM_',
      },
    },
    dag: {
      interface_workflow: {
        '*': ':L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3R5cGVzY3JpcHQvZGFnL2ludGVyZmFjZS13b3JrZmxvdy50cw__',
        declaring_a_dag_workflow:
          'Declaring a DAG Workflow:L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3R5cGVzY3JpcHQvZGFnL2ludGVyZmFjZS13b3JrZmxvdy50cw__',
      },
      run: {
        '*': ':L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3R5cGVzY3JpcHQvZGFnL3J1bi50cw__',
      },
      worker: {
        '*': ':L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3R5cGVzY3JpcHQvZGFnL3dvcmtlci50cw__',
      },
      workflow: {
        '*': ':L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3R5cGVzY3JpcHQvZGFnL3dvcmtmbG93LnRz',
        declaring_a_dag_workflow:
          'Declaring a DAG Workflow:L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3R5cGVzY3JpcHQvZGFnL3dvcmtmbG93LnRz',
      },
    },
    dag_match_condition: {
      complex_workflow: {
        '*': ':L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3R5cGVzY3JpcHQvZGFnX21hdGNoX2NvbmRpdGlvbi9jb21wbGV4LXdvcmtmbG93LnRz',
        create_a_workflow:
          'Create a workflow:L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3R5cGVzY3JpcHQvZGFnX21hdGNoX2NvbmRpdGlvbi9jb21wbGV4LXdvcmtmbG93LnRz',
        add_base_task:
          'Add base task:L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3R5cGVzY3JpcHQvZGFnX21hdGNoX2NvbmRpdGlvbi9jb21wbGV4LXdvcmtmbG93LnRz',
        add_wait_for_sleep:
          'Add wait for sleep:L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3R5cGVzY3JpcHQvZGFnX21hdGNoX2NvbmRpdGlvbi9jb21wbGV4LXdvcmtmbG93LnRz',
        add_skip_on_event:
          'Add skip on event:L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3R5cGVzY3JpcHQvZGFnX21hdGNoX2NvbmRpdGlvbi9jb21wbGV4LXdvcmtmbG93LnRz',
        add_branching:
          'Add branching:L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3R5cGVzY3JpcHQvZGFnX21hdGNoX2NvbmRpdGlvbi9jb21wbGV4LXdvcmtmbG93LnRz',
        add_wait_for_event:
          'Add wait for event:L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3R5cGVzY3JpcHQvZGFnX21hdGNoX2NvbmRpdGlvbi9jb21wbGV4LXdvcmtmbG93LnRz',
        add_sum:
          'Add sum:L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3R5cGVzY3JpcHQvZGFnX21hdGNoX2NvbmRpdGlvbi9jb21wbGV4LXdvcmtmbG93LnRz',
      },
      event: {
        '*': ':L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3R5cGVzY3JpcHQvZGFnX21hdGNoX2NvbmRpdGlvbi9ldmVudC50cw__',
      },
      run: {
        '*': ':L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3R5cGVzY3JpcHQvZGFnX21hdGNoX2NvbmRpdGlvbi9ydW4udHM_',
      },
      worker: {
        '*': ':L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3R5cGVzY3JpcHQvZGFnX21hdGNoX2NvbmRpdGlvbi93b3JrZXIudHM_',
      },
      workflow: {
        '*': ':L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3R5cGVzY3JpcHQvZGFnX21hdGNoX2NvbmRpdGlvbi93b3JrZmxvdy50cw__',
      },
    },
    deep: {
      run: {
        '*': ':L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3R5cGVzY3JpcHQvZGVlcC9ydW4udHM_',
      },
      worker: {
        '*': ':L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3R5cGVzY3JpcHQvZGVlcC93b3JrZXIudHM_',
      },
      workflow: {
        '*': ':L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3R5cGVzY3JpcHQvZGVlcC93b3JrZmxvdy50cw__',
      },
    },
    durable_event: {
      event: {
        '*': ':L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3R5cGVzY3JpcHQvZHVyYWJsZS1ldmVudC9ldmVudC50cw__',
      },
      run: {
        '*': ':L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3R5cGVzY3JpcHQvZHVyYWJsZS1ldmVudC9ydW4udHM_',
      },
      worker: {
        '*': ':L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3R5cGVzY3JpcHQvZHVyYWJsZS1ldmVudC93b3JrZXIudHM_',
      },
      workflow: {
        '*': ':L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3R5cGVzY3JpcHQvZHVyYWJsZS1ldmVudC93b3JrZmxvdy50cw__',
        durable_event:
          'Durable Event:L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3R5cGVzY3JpcHQvZHVyYWJsZS1ldmVudC93b3JrZmxvdy50cw__',
        durable_event_with_filter:
          'Durable Event With Filter:L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3R5cGVzY3JpcHQvZHVyYWJsZS1ldmVudC93b3JrZmxvdy50cw__',
      },
    },
    durable_sleep: {
      event: {
        '*': ':L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3R5cGVzY3JpcHQvZHVyYWJsZS1zbGVlcC9ldmVudC50cw__',
      },
      run: {
        '*': ':L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3R5cGVzY3JpcHQvZHVyYWJsZS1zbGVlcC9ydW4udHM_',
      },
      worker: {
        '*': ':L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3R5cGVzY3JpcHQvZHVyYWJsZS1zbGVlcC93b3JrZXIudHM_',
      },
      workflow: {
        '*': ':L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3R5cGVzY3JpcHQvZHVyYWJsZS1zbGVlcC93b3JrZmxvdy50cw__',
        durable_sleep:
          'Durable Sleep:L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3R5cGVzY3JpcHQvZHVyYWJsZS1zbGVlcC93b3JrZmxvdy50cw__',
      },
    },
    hatchet_client: {
      '*': ':L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3R5cGVzY3JpcHQvaGF0Y2hldC1jbGllbnQudHM_',
    },
    inferred_typing: {
      run: {
        '*': ':L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3R5cGVzY3JpcHQvaW5mZXJyZWQtdHlwaW5nL3J1bi50cw__',
      },
      worker: {
        '*': ':L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3R5cGVzY3JpcHQvaW5mZXJyZWQtdHlwaW5nL3dvcmtlci50cw__',
      },
      workflow: {
        '*': ':L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3R5cGVzY3JpcHQvaW5mZXJyZWQtdHlwaW5nL3dvcmtmbG93LnRz',
      },
    },
    landing_page: {
      durable_excution: {
        '*': ':L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3R5cGVzY3JpcHQvbGFuZGluZ19wYWdlL2R1cmFibGUtZXhjdXRpb24udHM_',
        declaring_a_durable_task:
          'Declaring a Durable Task:L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3R5cGVzY3JpcHQvbGFuZGluZ19wYWdlL2R1cmFibGUtZXhjdXRpb24udHM_',
        running_a_task:
          'Running a Task:L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3R5cGVzY3JpcHQvbGFuZGluZ19wYWdlL2R1cmFibGUtZXhjdXRpb24udHM_',
      },
      event_signaling: {
        '*': ':L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3R5cGVzY3JpcHQvbGFuZGluZ19wYWdlL2V2ZW50LXNpZ25hbGluZy50cw__',
        trigger_on_an_event:
          'Trigger on an event:L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3R5cGVzY3JpcHQvbGFuZGluZ19wYWdlL2V2ZW50LXNpZ25hbGluZy50cw__',
      },
      flow_control: {
        '*': ':L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3R5cGVzY3JpcHQvbGFuZGluZ19wYWdlL2Zsb3ctY29udHJvbC50cw__',
        process_what_you_can_handle:
          'Process what you can handle:L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3R5cGVzY3JpcHQvbGFuZGluZ19wYWdlL2Zsb3ctY29udHJvbC50cw__',
      },
      queues: {
        '*': ':L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3R5cGVzY3JpcHQvbGFuZGluZ19wYWdlL3F1ZXVlcy50cw__',
        declaring_a_task:
          'Declaring a Task:L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3R5cGVzY3JpcHQvbGFuZGluZ19wYWdlL3F1ZXVlcy50cw__',
        running_a_task:
          'Running a Task:L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3R5cGVzY3JpcHQvbGFuZGluZ19wYWdlL3F1ZXVlcy50cw__',
      },
      scheduling: {
        '*': ':L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3R5cGVzY3JpcHQvbGFuZGluZ19wYWdlL3NjaGVkdWxpbmcudHM_',
        schedules_and_crons:
          'Schedules and Crons:L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3R5cGVzY3JpcHQvbGFuZGluZ19wYWdlL3NjaGVkdWxpbmcudHM_',
      },
      task_routing: {
        '*': ':L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3R5cGVzY3JpcHQvbGFuZGluZ19wYWdlL3Rhc2stcm91dGluZy50cw__',
        route_tasks_to_workers_with_matching_labels:
          'Route tasks to workers with matching labels:L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3R5cGVzY3JpcHQvbGFuZGluZ19wYWdlL3Rhc2stcm91dGluZy50cw__',
      },
    },
    legacy: {
      run: {
        '*': ':L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3R5cGVzY3JpcHQvbGVnYWN5L3J1bi50cw__',
      },
      worker: {
        '*': ':L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3R5cGVzY3JpcHQvbGVnYWN5L3dvcmtlci50cw__',
      },
      workflow: {
        '*': ':L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3R5cGVzY3JpcHQvbGVnYWN5L3dvcmtmbG93LnRz',
      },
    },
    migration_guides: {
      hatchet_client: {
        '*': ':L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3R5cGVzY3JpcHQvbWlncmF0aW9uLWd1aWRlcy9oYXRjaGV0LWNsaWVudC50cw__',
      },
      mergent: {
        '*': ':L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3R5cGVzY3JpcHQvbWlncmF0aW9uLWd1aWRlcy9tZXJnZW50LnRz',
        before_mergent:
          'Before (Mergent):L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3R5cGVzY3JpcHQvbWlncmF0aW9uLWd1aWRlcy9tZXJnZW50LnRz',
        after_hatchet:
          'After (Hatchet):L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3R5cGVzY3JpcHQvbWlncmF0aW9uLWd1aWRlcy9tZXJnZW50LnRz',
        running_a_task_mergent:
          'Running a task (Mergent):L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3R5cGVzY3JpcHQvbWlncmF0aW9uLWd1aWRlcy9tZXJnZW50LnRz',
        running_a_task_hatchet:
          'Running a task (Hatchet):L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3R5cGVzY3JpcHQvbWlncmF0aW9uLWd1aWRlcy9tZXJnZW50LnRz',
        scheduling_tasks_mergent:
          'Scheduling tasks (Mergent):L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3R5cGVzY3JpcHQvbWlncmF0aW9uLWd1aWRlcy9tZXJnZW50LnRz',
        scheduling_tasks_hatchet:
          'Scheduling tasks (Hatchet):L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3R5cGVzY3JpcHQvbWlncmF0aW9uLWd1aWRlcy9tZXJnZW50LnRz',
      },
    },
    multiple_wf_concurrency: {
      run: {
        '*': ':L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3R5cGVzY3JpcHQvbXVsdGlwbGVfd2ZfY29uY3VycmVuY3kvcnVuLnRz',
      },
      worker: {
        '*': ':L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3R5cGVzY3JpcHQvbXVsdGlwbGVfd2ZfY29uY3VycmVuY3kvd29ya2VyLnRz',
      },
      workflow: {
        '*': ':L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3R5cGVzY3JpcHQvbXVsdGlwbGVfd2ZfY29uY3VycmVuY3kvd29ya2Zsb3cudHM_',
        concurrency_strategy_with_key:
          'Concurrency Strategy With Key:L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3R5cGVzY3JpcHQvbXVsdGlwbGVfd2ZfY29uY3VycmVuY3kvd29ya2Zsb3cudHM_',
      },
    },
    non_retryable: {
      run: {
        '*': ':L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3R5cGVzY3JpcHQvbm9uX3JldHJ5YWJsZS9ydW4udHM_',
      },
      worker: {
        '*': ':L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3R5cGVzY3JpcHQvbm9uX3JldHJ5YWJsZS93b3JrZXIudHM_',
      },
      workflow: {
        '*': ':L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3R5cGVzY3JpcHQvbm9uX3JldHJ5YWJsZS93b3JrZmxvdy50cw__',
        non_retrying_task:
          'Non-retrying task:L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3R5cGVzY3JpcHQvbm9uX3JldHJ5YWJsZS93b3JrZmxvdy50cw__',
      },
    },
    on_cron: {
      worker: {
        '*': ':L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3R5cGVzY3JpcHQvb25fY3Jvbi93b3JrZXIudHM_',
      },
      workflow: {
        '*': ':L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3R5cGVzY3JpcHQvb25fY3Jvbi93b3JrZmxvdy50cw__',
        run_workflow_on_cron:
          'Run Workflow on Cron:L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3R5cGVzY3JpcHQvb25fY3Jvbi93b3JrZmxvdy50cw__',
      },
    },
    on_event: {
      event: {
        '*': ':L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3R5cGVzY3JpcHQvb25fZXZlbnQvZXZlbnQudHM_',
        pushing_an_event:
          'Pushing an Event:L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3R5cGVzY3JpcHQvb25fZXZlbnQvZXZlbnQudHM_',
      },
      worker: {
        '*': ':L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3R5cGVzY3JpcHQvb25fZXZlbnQvd29ya2VyLnRz',
      },
      workflow: {
        '*': ':L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3R5cGVzY3JpcHQvb25fZXZlbnQvd29ya2Zsb3cudHM_',
        run_workflow_on_event:
          'Run workflow on event:L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3R5cGVzY3JpcHQvb25fZXZlbnQvd29ya2Zsb3cudHM_',
      },
    },
    on_event_copy: {
      event: {
        '*': ':L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3R5cGVzY3JpcHQvb25fZXZlbnQgY29weS9ldmVudC50cw__',
        pushing_an_event:
          'Pushing an Event:L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3R5cGVzY3JpcHQvb25fZXZlbnQgY29weS9ldmVudC50cw__',
      },
      worker: {
        '*': ':L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3R5cGVzY3JpcHQvb25fZXZlbnQgY29weS93b3JrZXIudHM_',
      },
      workflow: {
        '*': ':L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3R5cGVzY3JpcHQvb25fZXZlbnQgY29weS93b3JrZmxvdy50cw__',
        run_workflow_on_event:
          'Run workflow on event:L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3R5cGVzY3JpcHQvb25fZXZlbnQgY29weS93b3JrZmxvdy50cw__',
      },
    },
    on_failure: {
      run: {
        '*': ':L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3R5cGVzY3JpcHQvb25fZmFpbHVyZS9ydW4udHM_',
      },
      worker: {
        '*': ':L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3R5cGVzY3JpcHQvb25fZmFpbHVyZS93b3JrZXIudHM_',
      },
      workflow: {
        '*': ':L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3R5cGVzY3JpcHQvb25fZmFpbHVyZS93b3JrZmxvdy50cw__',
        on_failure_task:
          'On Failure Task:L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3R5cGVzY3JpcHQvb25fZmFpbHVyZS93b3JrZmxvdy50cw__',
      },
    },
    on_success: {
      run: {
        '*': ':L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3R5cGVzY3JpcHQvb25fc3VjY2Vzcy9ydW4udHM_',
      },
      worker: {
        '*': ':L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3R5cGVzY3JpcHQvb25fc3VjY2Vzcy93b3JrZXIudHM_',
      },
      workflow: {
        '*': ':L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3R5cGVzY3JpcHQvb25fc3VjY2Vzcy93b3JrZmxvdy50cw__',
        on_success_dag:
          'On Success DAG:L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3R5cGVzY3JpcHQvb25fc3VjY2Vzcy93b3JrZmxvdy50cw__',
      },
    },
    priority: {
      run: {
        '*': ':L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3R5cGVzY3JpcHQvcHJpb3JpdHkvcnVuLnRz',
        run_a_task_with_a_priority:
          'Run a Task with a Priority:L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3R5cGVzY3JpcHQvcHJpb3JpdHkvcnVuLnRz',
        schedule_and_cron:
          'Schedule and cron:L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3R5cGVzY3JpcHQvcHJpb3JpdHkvcnVuLnRz',
      },
      worker: {
        '*': ':L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3R5cGVzY3JpcHQvcHJpb3JpdHkvd29ya2VyLnRz',
      },
      workflow: {
        '*': ':L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3R5cGVzY3JpcHQvcHJpb3JpdHkvd29ya2Zsb3cudHM_',
        simple_task_priority:
          'Simple Task Priority:L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3R5cGVzY3JpcHQvcHJpb3JpdHkvd29ya2Zsb3cudHM_',
        task_priority_in_a_workflow:
          'Task Priority in a Workflow:L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3R5cGVzY3JpcHQvcHJpb3JpdHkvd29ya2Zsb3cudHM_',
      },
    },
    rate_limit: {
      workflow: {
        '*': ':L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3R5cGVzY3JpcHQvcmF0ZV9saW1pdC93b3JrZmxvdy50cw__',
        upsert_rate_limit:
          'Upsert Rate Limit:L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3R5cGVzY3JpcHQvcmF0ZV9saW1pdC93b3JrZmxvdy50cw__',
        static:
          'Static:L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3R5cGVzY3JpcHQvcmF0ZV9saW1pdC93b3JrZmxvdy50cw__',
        dynamic:
          'Dynamic:L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3R5cGVzY3JpcHQvcmF0ZV9saW1pdC93b3JrZmxvdy50cw__',
      },
    },
    retries: {
      run: {
        '*': ':L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3R5cGVzY3JpcHQvcmV0cmllcy9ydW4udHM_',
      },
      worker: {
        '*': ':L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3R5cGVzY3JpcHQvcmV0cmllcy93b3JrZXIudHM_',
      },
      workflow: {
        '*': ':L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3R5cGVzY3JpcHQvcmV0cmllcy93b3JrZmxvdy50cw__',
        simple_step_retries:
          'Simple Step Retries:L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3R5cGVzY3JpcHQvcmV0cmllcy93b3JrZmxvdy50cw__',
        retries_with_count:
          'Retries with Count:L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3R5cGVzY3JpcHQvcmV0cmllcy93b3JrZmxvdy50cw__',
        get_the_current_retry_count:
          'Get the current retry count:L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3R5cGVzY3JpcHQvcmV0cmllcy93b3JrZmxvdy50cw__',
        retries_with_backoff:
          'Retries with Backoff:L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3R5cGVzY3JpcHQvcmV0cmllcy93b3JrZmxvdy50cw__',
      },
    },
    simple: {
      bulk: {
        '*': ':L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3R5cGVzY3JpcHQvc2ltcGxlL2J1bGsudHM_',
        bulk_run_a_task:
          'Bulk Run a Task:L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3R5cGVzY3JpcHQvc2ltcGxlL2J1bGsudHM_',
        bulk_run_tasks_from_within_a_task:
          'Bulk Run Tasks from within a Task:L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3R5cGVzY3JpcHQvc2ltcGxlL2J1bGsudHM_',
      },
      client_run: {
        '*': ':L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3R5cGVzY3JpcHQvc2ltcGxlL2NsaWVudC1ydW4udHM_',
        client_run_methods:
          'Client Run Methods:L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3R5cGVzY3JpcHQvc2ltcGxlL2NsaWVudC1ydW4udHM_',
      },
      cron: {
        '*': ':L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3R5cGVzY3JpcHQvc2ltcGxlL2Nyb24udHM_',
        create:
          'Create:L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3R5cGVzY3JpcHQvc2ltcGxlL2Nyb24udHM_',
        delete:
          'Delete:L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3R5cGVzY3JpcHQvc2ltcGxlL2Nyb24udHM_',
        list: 'List:L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3R5cGVzY3JpcHQvc2ltcGxlL2Nyb24udHM_',
      },
      delay: {
        '*': ':L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3R5cGVzY3JpcHQvc2ltcGxlL2RlbGF5LnRz',
      },
      enqueue: {
        '*': ':L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3R5cGVzY3JpcHQvc2ltcGxlL2VucXVldWUudHM_',
        enqueuing_a_workflow_fire_and_forget:
          'Enqueuing a Workflow (Fire and Forget):L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3R5cGVzY3JpcHQvc2ltcGxlL2VucXVldWUudHM_',
        subscribing_to_results:
          'Subscribing to results:L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3R5cGVzY3JpcHQvc2ltcGxlL2VucXVldWUudHM_',
      },
      run: {
        '*': ':L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3R5cGVzY3JpcHQvc2ltcGxlL3J1bi50cw__',
        running_a_task:
          'Running a Task:L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3R5cGVzY3JpcHQvc2ltcGxlL3J1bi50cw__',
        running_multiple_tasks:
          'Running Multiple Tasks:L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3R5cGVzY3JpcHQvc2ltcGxlL3J1bi50cw__',
        spawning_tasks_from_within_a_task:
          'Spawning Tasks from within a Task:L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3R5cGVzY3JpcHQvc2ltcGxlL3J1bi50cw__',
      },
      schedule: {
        '*': ':L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3R5cGVzY3JpcHQvc2ltcGxlL3NjaGVkdWxlLnRz',
        create_a_scheduled_run:
          'Create a Scheduled Run:L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3R5cGVzY3JpcHQvc2ltcGxlL3NjaGVkdWxlLnRz',
        delete_a_scheduled_run:
          'Delete a Scheduled Run:L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3R5cGVzY3JpcHQvc2ltcGxlL3NjaGVkdWxlLnRz',
        list_scheduled_runs:
          'List Scheduled Runs:L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3R5cGVzY3JpcHQvc2ltcGxlL3NjaGVkdWxlLnRz',
      },
      stub_workflow: {
        '*': ':L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3R5cGVzY3JpcHQvc2ltcGxlL3N0dWItd29ya2Zsb3cudHM_',
        declaring_an_external_workflow_reference:
          'Declaring an External Workflow Reference:L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3R5cGVzY3JpcHQvc2ltcGxlL3N0dWItd29ya2Zsb3cudHM_',
      },
      worker: {
        '*': ':L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3R5cGVzY3JpcHQvc2ltcGxlL3dvcmtlci50cw__',
        declaring_a_worker:
          'Declaring a Worker:L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3R5cGVzY3JpcHQvc2ltcGxlL3dvcmtlci50cw__',
      },
      workflow_with_child: {
        '*': ':L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3R5cGVzY3JpcHQvc2ltcGxlL3dvcmtmbG93LXdpdGgtY2hpbGQudHM_',
        declaring_a_task:
          'Declaring a Task:L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3R5cGVzY3JpcHQvc2ltcGxlL3dvcmtmbG93LXdpdGgtY2hpbGQudHM_',
      },
      workflow: {
        '*': ':L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3R5cGVzY3JpcHQvc2ltcGxlL3dvcmtmbG93LnRz',
        declaring_a_task:
          'Declaring a Task:L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3R5cGVzY3JpcHQvc2ltcGxlL3dvcmtmbG93LnRz',
      },
    },
    sticky: {
      run: {
        '*': ':L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3R5cGVzY3JpcHQvc3RpY2t5L3J1bi50cw__',
      },
      worker: {
        '*': ':L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3R5cGVzY3JpcHQvc3RpY2t5L3dvcmtlci50cw__',
      },
      workflow: {
        '*': ':L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3R5cGVzY3JpcHQvc3RpY2t5L3dvcmtmbG93LnRz',
        sticky_task:
          'Sticky Task:L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3R5cGVzY3JpcHQvc3RpY2t5L3dvcmtmbG93LnRz',
      },
    },
    timeouts: {
      run: {
        '*': ':L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3R5cGVzY3JpcHQvdGltZW91dHMvcnVuLnRz',
        running_a_task_with_results:
          'Running a Task with Results:L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3R5cGVzY3JpcHQvdGltZW91dHMvcnVuLnRz',
      },
      worker: {
        '*': ':L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3R5cGVzY3JpcHQvdGltZW91dHMvd29ya2VyLnRz',
        declaring_a_worker:
          'Declaring a Worker:L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3R5cGVzY3JpcHQvdGltZW91dHMvd29ya2VyLnRz',
      },
      workflow: {
        '*': ':L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3R5cGVzY3JpcHQvdGltZW91dHMvd29ya2Zsb3cudHM_',
        declaring_a_task:
          'Declaring a Task:L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3R5cGVzY3JpcHQvdGltZW91dHMvd29ya2Zsb3cudHM_',
      },
    },
    with_timeouts: {
      workflow: {
        '*': ':L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3R5cGVzY3JpcHQvd2l0aF90aW1lb3V0cy93b3JrZmxvdy50cw__',
        declaring_a_task:
          'Declaring a Task:L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3R5cGVzY3JpcHQvd2l0aF90aW1lb3V0cy93b3JrZmxvdy50cw__',
        execution_timeout:
          'Execution Timeout:L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3R5cGVzY3JpcHQvd2l0aF90aW1lb3V0cy93b3JrZmxvdy50cw__',
        refresh_timeout:
          'Refresh Timeout:L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3R5cGVzY3JpcHQvd2l0aF90aW1lb3V0cy93b3JrZmxvdy50cw__',
      },
    },
  },
  python: {
    __init__: {
      '*': ':L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3B5dGhvbi9fX2luaXRfXy5weQ__',
    },
    affinity_workers: {
      trigger: {
        '*': ':L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3B5dGhvbi9hZmZpbml0eV93b3JrZXJzL3RyaWdnZXIucHk_',
      },
      worker: {
        '*': ':L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3B5dGhvbi9hZmZpbml0eV93b3JrZXJzL3dvcmtlci5weQ__',
        affinityworkflow:
          'AffinityWorkflow:L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3B5dGhvbi9hZmZpbml0eV93b3JrZXJzL3dvcmtlci5weQ__',
        affinitytask:
          'AffinityTask:L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3B5dGhvbi9hZmZpbml0eV93b3JrZXJzL3dvcmtlci5weQ__',
        affinityworker:
          'AffinityWorker:L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3B5dGhvbi9hZmZpbml0eV93b3JrZXJzL3dvcmtlci5weQ__',
      },
    },
    api: {
      api: {
        '*': ':L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3B5dGhvbi9hcGkvYXBpLnB5',
      },
      async_api: {
        '*': ':L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3B5dGhvbi9hcGkvYXN5bmNfYXBpLnB5',
      },
    },
    blocked_async: {
      trigger: {
        '*': ':L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3B5dGhvbi9ibG9ja2VkX2FzeW5jL3RyaWdnZXIucHk_',
      },
      worker: {
        '*': ':L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3B5dGhvbi9ibG9ja2VkX2FzeW5jL3dvcmtlci5weQ__',
      },
    },
    bulk_fanout: {
      bulk_trigger: {
        '*': ':L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3B5dGhvbi9idWxrX2Zhbm91dC9idWxrX3RyaWdnZXIucHk_',
      },
      stream: {
        '*': ':L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3B5dGhvbi9idWxrX2Zhbm91dC9zdHJlYW0ucHk_',
      },
      test_bulk_fanout: {
        '*': ':L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3B5dGhvbi9idWxrX2Zhbm91dC90ZXN0X2J1bGtfZmFub3V0LnB5',
      },
      trigger: {
        '*': ':L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3B5dGhvbi9idWxrX2Zhbm91dC90cmlnZ2VyLnB5',
      },
      worker: {
        '*': ':L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3B5dGhvbi9idWxrX2Zhbm91dC93b3JrZXIucHk_',
        bulkfanoutparent:
          'BulkFanoutParent:L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3B5dGhvbi9idWxrX2Zhbm91dC93b3JrZXIucHk_',
      },
    },
    bulk_operations: {
      cancel: {
        '*': ':L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3B5dGhvbi9idWxrX29wZXJhdGlvbnMvY2FuY2VsLnB5',
        setup:
          'Setup:L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3B5dGhvbi9idWxrX29wZXJhdGlvbnMvY2FuY2VsLnB5',
        list_runs:
          'List runs:L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3B5dGhvbi9idWxrX29wZXJhdGlvbnMvY2FuY2VsLnB5',
        cancel_by_run_ids:
          'Cancel by run ids:L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3B5dGhvbi9idWxrX29wZXJhdGlvbnMvY2FuY2VsLnB5',
        cancel_by_filters:
          'Cancel by filters:L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3B5dGhvbi9idWxrX29wZXJhdGlvbnMvY2FuY2VsLnB5',
      },
      replay: {
        '*': ':L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3B5dGhvbi9idWxrX29wZXJhdGlvbnMvcmVwbGF5LnB5',
        setup:
          'Setup:L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3B5dGhvbi9idWxrX29wZXJhdGlvbnMvcmVwbGF5LnB5',
        list_runs:
          'List runs:L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3B5dGhvbi9idWxrX29wZXJhdGlvbnMvcmVwbGF5LnB5',
        replay_by_run_ids:
          'Replay by run ids:L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3B5dGhvbi9idWxrX29wZXJhdGlvbnMvcmVwbGF5LnB5',
        replay_by_filters:
          'Replay by filters:L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3B5dGhvbi9idWxrX29wZXJhdGlvbnMvcmVwbGF5LnB5',
      },
    },
    cancellation: {
      test_cancellation: {
        '*': ':L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3B5dGhvbi9jYW5jZWxsYXRpb24vdGVzdF9jYW5jZWxsYXRpb24ucHk_',
      },
      trigger: {
        '*': ':L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3B5dGhvbi9jYW5jZWxsYXRpb24vdHJpZ2dlci5weQ__',
      },
      worker: {
        '*': ':L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3B5dGhvbi9jYW5jZWxsYXRpb24vd29ya2VyLnB5',
        self_cancelling_task:
          'Self-cancelling task:L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3B5dGhvbi9jYW5jZWxsYXRpb24vd29ya2VyLnB5',
        checking_exit_flag:
          'Checking exit flag:L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3B5dGhvbi9jYW5jZWxsYXRpb24vd29ya2VyLnB5',
      },
    },
    child: {
      bulk: {
        '*': ':L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3B5dGhvbi9jaGlsZC9idWxrLnB5',
        running_a_task:
          'Running a Task:L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3B5dGhvbi9jaGlsZC9idWxrLnB5',
        bulk_run_a_task:
          'Bulk Run a Task:L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3B5dGhvbi9jaGlsZC9idWxrLnB5',
        running_multiple_tasks:
          'Running Multiple Tasks:L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3B5dGhvbi9jaGlsZC9idWxrLnB5',
      },
      simple_fanout: {
        '*': ':L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3B5dGhvbi9jaGlsZC9zaW1wbGUtZmFub3V0LnB5',
        running_a_task_from_within_a_task:
          'Running a Task from within a Task:L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3B5dGhvbi9jaGlsZC9zaW1wbGUtZmFub3V0LnB5',
      },
      trigger: {
        '*': ':L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3B5dGhvbi9jaGlsZC90cmlnZ2VyLnB5',
        running_a_task:
          'Running a Task:L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3B5dGhvbi9jaGlsZC90cmlnZ2VyLnB5',
        schedule_a_task:
          'Schedule a Task:L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3B5dGhvbi9jaGlsZC90cmlnZ2VyLnB5',
        running_a_task_aio:
          'Running a Task AIO:L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3B5dGhvbi9jaGlsZC90cmlnZ2VyLnB5',
        running_multiple_tasks:
          'Running Multiple Tasks:L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3B5dGhvbi9jaGlsZC90cmlnZ2VyLnB5',
      },
      worker: {
        '*': ':L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3B5dGhvbi9jaGlsZC93b3JrZXIucHk_',
        simple:
          'Simple:L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3B5dGhvbi9jaGlsZC93b3JrZXIucHk_',
      },
    },
    concurrency_limit: {
      test_concurrency_limit: {
        '*': ':L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3B5dGhvbi9jb25jdXJyZW5jeV9saW1pdC90ZXN0X2NvbmN1cnJlbmN5X2xpbWl0LnB5',
      },
      trigger: {
        '*': ':L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3B5dGhvbi9jb25jdXJyZW5jeV9saW1pdC90cmlnZ2VyLnB5',
      },
      worker: {
        '*': ':L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3B5dGhvbi9jb25jdXJyZW5jeV9saW1pdC93b3JrZXIucHk_',
        workflow:
          'Workflow:L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3B5dGhvbi9jb25jdXJyZW5jeV9saW1pdC93b3JrZXIucHk_',
      },
    },
    concurrency_limit_rr: {
      test_concurrency_limit_rr: {
        '*': ':L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3B5dGhvbi9jb25jdXJyZW5jeV9saW1pdF9yci90ZXN0X2NvbmN1cnJlbmN5X2xpbWl0X3JyLnB5',
      },
      trigger: {
        '*': ':L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3B5dGhvbi9jb25jdXJyZW5jeV9saW1pdF9yci90cmlnZ2VyLnB5',
      },
      worker: {
        '*': ':L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3B5dGhvbi9jb25jdXJyZW5jeV9saW1pdF9yci93b3JrZXIucHk_',
        concurrency_strategy_with_key:
          'Concurrency Strategy With Key:L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3B5dGhvbi9jb25jdXJyZW5jeV9saW1pdF9yci93b3JrZXIucHk_',
      },
    },
    concurrency_limit_rr_load: {
      event: {
        '*': ':L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3B5dGhvbi9jb25jdXJyZW5jeV9saW1pdF9ycl9sb2FkL2V2ZW50LnB5',
      },
      worker: {
        '*': ':L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3B5dGhvbi9jb25jdXJyZW5jeV9saW1pdF9ycl9sb2FkL3dvcmtlci5weQ__',
      },
    },
    concurrency_multiple_keys: {
      test_multiple_concurrency_keys: {
        '*': ':L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3B5dGhvbi9jb25jdXJyZW5jeV9tdWx0aXBsZV9rZXlzL3Rlc3RfbXVsdGlwbGVfY29uY3VycmVuY3lfa2V5cy5weQ__',
      },
      worker: {
        '*': ':L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3B5dGhvbi9jb25jdXJyZW5jeV9tdWx0aXBsZV9rZXlzL3dvcmtlci5weQ__',
        concurrency_strategy_with_key:
          'Concurrency Strategy With Key:L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3B5dGhvbi9jb25jdXJyZW5jeV9tdWx0aXBsZV9rZXlzL3dvcmtlci5weQ__',
      },
    },
    concurrency_workflow_level: {
      test_workflow_level_concurrency: {
        '*': ':L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3B5dGhvbi9jb25jdXJyZW5jeV93b3JrZmxvd19sZXZlbC90ZXN0X3dvcmtmbG93X2xldmVsX2NvbmN1cnJlbmN5LnB5',
      },
      worker: {
        '*': ':L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3B5dGhvbi9jb25jdXJyZW5jeV93b3JrZmxvd19sZXZlbC93b3JrZXIucHk_',
        multiple_concurrency_keys:
          'Multiple Concurrency Keys:L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3B5dGhvbi9jb25jdXJyZW5jeV93b3JrZmxvd19sZXZlbC93b3JrZXIucHk_',
      },
    },
    cron: {
      programatic_async: {
        '*': ':L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3B5dGhvbi9jcm9uL3Byb2dyYW1hdGljLWFzeW5jLnB5',
        create:
          'Create:L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3B5dGhvbi9jcm9uL3Byb2dyYW1hdGljLWFzeW5jLnB5',
        list: 'List:L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3B5dGhvbi9jcm9uL3Byb2dyYW1hdGljLWFzeW5jLnB5',
        get: 'Get:L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3B5dGhvbi9jcm9uL3Byb2dyYW1hdGljLWFzeW5jLnB5',
        delete:
          'Delete:L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3B5dGhvbi9jcm9uL3Byb2dyYW1hdGljLWFzeW5jLnB5',
      },
      programatic_sync: {
        '*': ':L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3B5dGhvbi9jcm9uL3Byb2dyYW1hdGljLXN5bmMucHk_',
        create:
          'Create:L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3B5dGhvbi9jcm9uL3Byb2dyYW1hdGljLXN5bmMucHk_',
        list: 'List:L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3B5dGhvbi9jcm9uL3Byb2dyYW1hdGljLXN5bmMucHk_',
        get: 'Get:L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3B5dGhvbi9jcm9uL3Byb2dyYW1hdGljLXN5bmMucHk_',
        delete:
          'Delete:L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3B5dGhvbi9jcm9uL3Byb2dyYW1hdGljLXN5bmMucHk_',
      },
      workflow_definition: {
        '*': ':L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3B5dGhvbi9jcm9uL3dvcmtmbG93LWRlZmluaXRpb24ucHk_',
        workflow_definition_cron_trigger:
          'Workflow Definition Cron Trigger:L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3B5dGhvbi9jcm9uL3dvcmtmbG93LWRlZmluaXRpb24ucHk_',
      },
    },
    dag: {
      test_dag: {
        '*': ':L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3B5dGhvbi9kYWcvdGVzdF9kYWcucHk_',
      },
      trigger: {
        '*': ':L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3B5dGhvbi9kYWcvdHJpZ2dlci5weQ__',
      },
      worker: {
        '*': ':L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3B5dGhvbi9kYWcvd29ya2VyLnB5',
      },
    },
    dedupe: {
      worker: {
        '*': ':L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3B5dGhvbi9kZWR1cGUvd29ya2VyLnB5',
      },
    },
    delayed: {
      test_delayed: {
        '*': ':L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3B5dGhvbi9kZWxheWVkL3Rlc3RfZGVsYXllZC5weQ__',
      },
      trigger: {
        '*': ':L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3B5dGhvbi9kZWxheWVkL3RyaWdnZXIucHk_',
      },
      worker: {
        '*': ':L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3B5dGhvbi9kZWxheWVkL3dvcmtlci5weQ__',
      },
    },
    durable: {
      test_durable: {
        '*': ':L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3B5dGhvbi9kdXJhYmxlL3Rlc3RfZHVyYWJsZS5weQ__',
      },
      trigger: {
        '*': ':L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3B5dGhvbi9kdXJhYmxlL3RyaWdnZXIucHk_',
      },
      worker: {
        '*': ':L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3B5dGhvbi9kdXJhYmxlL3dvcmtlci5weQ__',
        create_a_durable_workflow:
          'Create a durable workflow:L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3B5dGhvbi9kdXJhYmxlL3dvcmtlci5weQ__',
        add_durable_task:
          'Add durable task:L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3B5dGhvbi9kdXJhYmxlL3dvcmtlci5weQ__',
      },
    },
    durable_event: {
      trigger: {
        '*': ':L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3B5dGhvbi9kdXJhYmxlX2V2ZW50L3RyaWdnZXIucHk_',
      },
      worker: {
        '*': ':L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3B5dGhvbi9kdXJhYmxlX2V2ZW50L3dvcmtlci5weQ__',
        durable_event:
          'Durable Event:L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3B5dGhvbi9kdXJhYmxlX2V2ZW50L3dvcmtlci5weQ__',
        durable_event_with_filter:
          'Durable Event With Filter:L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3B5dGhvbi9kdXJhYmxlX2V2ZW50L3dvcmtlci5weQ__',
      },
    },
    durable_sleep: {
      trigger: {
        '*': ':L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3B5dGhvbi9kdXJhYmxlX3NsZWVwL3RyaWdnZXIucHk_',
      },
      worker: {
        '*': ':L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3B5dGhvbi9kdXJhYmxlX3NsZWVwL3dvcmtlci5weQ__',
        durable_sleep:
          'Durable Sleep:L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3B5dGhvbi9kdXJhYmxlX3NsZWVwL3dvcmtlci5weQ__',
      },
    },
    events: {
      event: {
        '*': ':L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3B5dGhvbi9ldmVudHMvZXZlbnQucHk_',
        event_trigger:
          'Event trigger:L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3B5dGhvbi9ldmVudHMvZXZlbnQucHk_',
      },
      test_event: {
        '*': ':L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3B5dGhvbi9ldmVudHMvdGVzdF9ldmVudC5weQ__',
      },
      worker: {
        '*': ':L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3B5dGhvbi9ldmVudHMvd29ya2VyLnB5',
        event_trigger:
          'Event trigger:L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3B5dGhvbi9ldmVudHMvd29ya2VyLnB5',
      },
    },
    fanout: {
      stream: {
        '*': ':L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3B5dGhvbi9mYW5vdXQvc3RyZWFtLnB5',
      },
      sync_stream: {
        '*': ':L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3B5dGhvbi9mYW5vdXQvc3luY19zdHJlYW0ucHk_',
      },
      test_fanout: {
        '*': ':L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3B5dGhvbi9mYW5vdXQvdGVzdF9mYW5vdXQucHk_',
      },
      trigger: {
        '*': ':L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3B5dGhvbi9mYW5vdXQvdHJpZ2dlci5weQ__',
      },
      worker: {
        '*': ':L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3B5dGhvbi9mYW5vdXQvd29ya2VyLnB5',
        fanoutparent:
          'FanoutParent:L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3B5dGhvbi9mYW5vdXQvd29ya2VyLnB5',
        fanoutchild:
          'FanoutChild:L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3B5dGhvbi9mYW5vdXQvd29ya2VyLnB5',
      },
    },
    fanout_sync: {
      test_fanout_sync: {
        '*': ':L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3B5dGhvbi9mYW5vdXRfc3luYy90ZXN0X2Zhbm91dF9zeW5jLnB5',
      },
      trigger: {
        '*': ':L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3B5dGhvbi9mYW5vdXRfc3luYy90cmlnZ2VyLnB5',
      },
      worker: {
        '*': ':L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3B5dGhvbi9mYW5vdXRfc3luYy93b3JrZXIucHk_',
      },
    },
    lifespans: {
      simple: {
        '*': ':L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3B5dGhvbi9saWZlc3BhbnMvc2ltcGxlLnB5',
        lifespan:
          'Lifespan:L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3B5dGhvbi9saWZlc3BhbnMvc2ltcGxlLnB5',
      },
      test_lifespans: {
        '*': ':L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3B5dGhvbi9saWZlc3BhbnMvdGVzdF9saWZlc3BhbnMucHk_',
      },
      trigger: {
        '*': ':L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3B5dGhvbi9saWZlc3BhbnMvdHJpZ2dlci5weQ__',
      },
      worker: {
        '*': ':L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3B5dGhvbi9saWZlc3BhbnMvd29ya2VyLnB5',
        use_the_lifespan_in_a_task:
          'Use the lifespan in a task:L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3B5dGhvbi9saWZlc3BhbnMvd29ya2VyLnB5',
        define_a_lifespan:
          'Define a lifespan:L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3B5dGhvbi9saWZlc3BhbnMvd29ya2VyLnB5',
      },
    },
    logger: {
      client: {
        '*': ':L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3B5dGhvbi9sb2dnZXIvY2xpZW50LnB5',
        rootlogger:
          'RootLogger:L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3B5dGhvbi9sb2dnZXIvY2xpZW50LnB5',
      },
      test_logger: {
        '*': ':L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3B5dGhvbi9sb2dnZXIvdGVzdF9sb2dnZXIucHk_',
      },
      trigger: {
        '*': ':L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3B5dGhvbi9sb2dnZXIvdHJpZ2dlci5weQ__',
      },
      worker: {
        '*': ':L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3B5dGhvbi9sb2dnZXIvd29ya2VyLnB5',
      },
      workflow: {
        '*': ':L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3B5dGhvbi9sb2dnZXIvd29ya2Zsb3cucHk_',
        loggingworkflow:
          'LoggingWorkflow:L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3B5dGhvbi9sb2dnZXIvd29ya2Zsb3cucHk_',
        contextlogger:
          'ContextLogger:L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3B5dGhvbi9sb2dnZXIvd29ya2Zsb3cucHk_',
      },
    },
    manual_slot_release: {
      worker: {
        '*': ':L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3B5dGhvbi9tYW51YWxfc2xvdF9yZWxlYXNlL3dvcmtlci5weQ__',
        slotrelease:
          'SlotRelease:L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3B5dGhvbi9tYW51YWxfc2xvdF9yZWxlYXNlL3dvcmtlci5weQ__',
      },
    },
    migration_guides: {
      __init__: {
        '*': ':L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3B5dGhvbi9taWdyYXRpb25fZ3VpZGVzL19faW5pdF9fLnB5',
      },
      hatchet_client: {
        '*': ':L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3B5dGhvbi9taWdyYXRpb25fZ3VpZGVzL2hhdGNoZXRfY2xpZW50LnB5',
      },
      mergent: {
        '*': ':L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3B5dGhvbi9taWdyYXRpb25fZ3VpZGVzL21lcmdlbnQucHk_',
        before_mergent:
          'Before (Mergent):L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3B5dGhvbi9taWdyYXRpb25fZ3VpZGVzL21lcmdlbnQucHk_',
        after_hatchet:
          'After (Hatchet):L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3B5dGhvbi9taWdyYXRpb25fZ3VpZGVzL21lcmdlbnQucHk_',
        running_a_task_mergent:
          'Running a task (Mergent):L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3B5dGhvbi9taWdyYXRpb25fZ3VpZGVzL21lcmdlbnQucHk_',
        running_a_task_hatchet:
          'Running a task (Hatchet):L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3B5dGhvbi9taWdyYXRpb25fZ3VpZGVzL21lcmdlbnQucHk_',
        scheduling_tasks_mergent:
          'Scheduling tasks (Mergent):L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3B5dGhvbi9taWdyYXRpb25fZ3VpZGVzL21lcmdlbnQucHk_',
        scheduling_tasks_hatchet:
          'Scheduling tasks (Hatchet):L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3B5dGhvbi9taWdyYXRpb25fZ3VpZGVzL21lcmdlbnQucHk_',
      },
    },
    non_retryable: {
      test_no_retry: {
        '*': ':L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3B5dGhvbi9ub25fcmV0cnlhYmxlL3Rlc3Rfbm9fcmV0cnkucHk_',
      },
      trigger: {
        '*': ':L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3B5dGhvbi9ub25fcmV0cnlhYmxlL3RyaWdnZXIucHk_',
      },
      worker: {
        '*': ':L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3B5dGhvbi9ub25fcmV0cnlhYmxlL3dvcmtlci5weQ__',
        non_retryable_task:
          'Non-retryable task:L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3B5dGhvbi9ub25fcmV0cnlhYmxlL3dvcmtlci5weQ__',
      },
    },
    on_failure: {
      test_on_failure: {
        '*': ':L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3B5dGhvbi9vbl9mYWlsdXJlL3Rlc3Rfb25fZmFpbHVyZS5weQ__',
      },
      trigger: {
        '*': ':L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3B5dGhvbi9vbl9mYWlsdXJlL3RyaWdnZXIucHk_',
      },
      worker: {
        '*': ':L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3B5dGhvbi9vbl9mYWlsdXJlL3dvcmtlci5weQ__',
        onfailure_step:
          'OnFailure Step:L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3B5dGhvbi9vbl9mYWlsdXJlL3dvcmtlci5weQ__',
        onfailure_with_details:
          'OnFailure With Details:L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3B5dGhvbi9vbl9mYWlsdXJlL3dvcmtlci5weQ__',
      },
    },
    on_success: {
      trigger: {
        '*': ':L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3B5dGhvbi9vbl9zdWNjZXNzL3RyaWdnZXIucHk_',
      },
      worker: {
        '*': ':L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3B5dGhvbi9vbl9zdWNjZXNzL3dvcmtlci5weQ__',
      },
    },
    opentelemetry_instrumentation: {
      client: {
        '*': ':L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3B5dGhvbi9vcGVudGVsZW1ldHJ5X2luc3RydW1lbnRhdGlvbi9jbGllbnQucHk_',
      },
      tracer: {
        '*': ':L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3B5dGhvbi9vcGVudGVsZW1ldHJ5X2luc3RydW1lbnRhdGlvbi90cmFjZXIucHk_',
      },
      triggers: {
        '*': ':L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3B5dGhvbi9vcGVudGVsZW1ldHJ5X2luc3RydW1lbnRhdGlvbi90cmlnZ2Vycy5weQ__',
      },
      worker: {
        '*': ':L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3B5dGhvbi9vcGVudGVsZW1ldHJ5X2luc3RydW1lbnRhdGlvbi93b3JrZXIucHk_',
      },
    },
    priority: {
      test_priority: {
        '*': ':L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3B5dGhvbi9wcmlvcml0eS90ZXN0X3ByaW9yaXR5LnB5',
      },
      trigger: {
        '*': ':L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3B5dGhvbi9wcmlvcml0eS90cmlnZ2VyLnB5',
        runtime_priority:
          'Runtime priority:L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3B5dGhvbi9wcmlvcml0eS90cmlnZ2VyLnB5',
        scheduled_priority:
          'Scheduled priority:L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3B5dGhvbi9wcmlvcml0eS90cmlnZ2VyLnB5',
        default_priority:
          'Default priority:L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3B5dGhvbi9wcmlvcml0eS90cmlnZ2VyLnB5',
      },
      worker: {
        '*': ':L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3B5dGhvbi9wcmlvcml0eS93b3JrZXIucHk_',
        default_priority:
          'Default priority:L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3B5dGhvbi9wcmlvcml0eS93b3JrZXIucHk_',
      },
    },
    rate_limit: {
      dynamic: {
        '*': ':L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3B5dGhvbi9yYXRlX2xpbWl0L2R5bmFtaWMucHk_',
      },
      test_rate_limit: {
        '*': ':L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3B5dGhvbi9yYXRlX2xpbWl0L3Rlc3RfcmF0ZV9saW1pdC5weQ__',
      },
      trigger: {
        '*': ':L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3B5dGhvbi9yYXRlX2xpbWl0L3RyaWdnZXIucHk_',
      },
      worker: {
        '*': ':L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3B5dGhvbi9yYXRlX2xpbWl0L3dvcmtlci5weQ__',
        workflow:
          'Workflow:L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3B5dGhvbi9yYXRlX2xpbWl0L3dvcmtlci5weQ__',
        static:
          'Static:L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3B5dGhvbi9yYXRlX2xpbWl0L3dvcmtlci5weQ__',
        dynamic:
          'Dynamic:L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3B5dGhvbi9yYXRlX2xpbWl0L3dvcmtlci5weQ__',
      },
    },
    retries: {
      worker: {
        '*': ':L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3B5dGhvbi9yZXRyaWVzL3dvcmtlci5weQ__',
        simple_step_retries:
          'Simple Step Retries:L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3B5dGhvbi9yZXRyaWVzL3dvcmtlci5weQ__',
        retries_with_count:
          'Retries with Count:L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3B5dGhvbi9yZXRyaWVzL3dvcmtlci5weQ__',
        retries_with_backoff:
          'Retries with Backoff:L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3B5dGhvbi9yZXRyaWVzL3dvcmtlci5weQ__',
      },
    },
    scheduled: {
      programatic_async: {
        '*': ':L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3B5dGhvbi9zY2hlZHVsZWQvcHJvZ3JhbWF0aWMtYXN5bmMucHk_',
        create:
          'Create:L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3B5dGhvbi9zY2hlZHVsZWQvcHJvZ3JhbWF0aWMtYXN5bmMucHk_',
        delete:
          'Delete:L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3B5dGhvbi9zY2hlZHVsZWQvcHJvZ3JhbWF0aWMtYXN5bmMucHk_',
        list: 'List:L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3B5dGhvbi9zY2hlZHVsZWQvcHJvZ3JhbWF0aWMtYXN5bmMucHk_',
        get: 'Get:L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3B5dGhvbi9zY2hlZHVsZWQvcHJvZ3JhbWF0aWMtYXN5bmMucHk_',
      },
      programatic_sync: {
        '*': ':L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3B5dGhvbi9zY2hlZHVsZWQvcHJvZ3JhbWF0aWMtc3luYy5weQ__',
        create:
          'Create:L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3B5dGhvbi9zY2hlZHVsZWQvcHJvZ3JhbWF0aWMtc3luYy5weQ__',
        delete:
          'Delete:L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3B5dGhvbi9zY2hlZHVsZWQvcHJvZ3JhbWF0aWMtc3luYy5weQ__',
        list: 'List:L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3B5dGhvbi9zY2hlZHVsZWQvcHJvZ3JhbWF0aWMtc3luYy5weQ__',
        get: 'Get:L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3B5dGhvbi9zY2hlZHVsZWQvcHJvZ3JhbWF0aWMtc3luYy5weQ__',
      },
    },
    simple: {
      trigger: {
        '*': ':L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3B5dGhvbi9zaW1wbGUvdHJpZ2dlci5weQ__',
      },
      worker: {
        '*': ':L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3B5dGhvbi9zaW1wbGUvd29ya2VyLnB5',
        simple:
          'Simple:L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3B5dGhvbi9zaW1wbGUvd29ya2VyLnB5',
      },
    },
    sticky_workers: {
      event: {
        '*': ':L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3B5dGhvbi9zdGlja3lfd29ya2Vycy9ldmVudC5weQ__',
      },
      worker: {
        '*': ':L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3B5dGhvbi9zdGlja3lfd29ya2Vycy93b3JrZXIucHk_',
        stickyworker:
          'StickyWorker:L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3B5dGhvbi9zdGlja3lfd29ya2Vycy93b3JrZXIucHk_',
        stickychild:
          'StickyChild:L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3B5dGhvbi9zdGlja3lfd29ya2Vycy93b3JrZXIucHk_',
      },
    },
    streaming: {
      async_stream: {
        '*': ':L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3B5dGhvbi9zdHJlYW1pbmcvYXN5bmNfc3RyZWFtLnB5',
      },
      sync_stream: {
        '*': ':L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3B5dGhvbi9zdHJlYW1pbmcvc3luY19zdHJlYW0ucHk_',
      },
      worker: {
        '*': ':L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3B5dGhvbi9zdHJlYW1pbmcvd29ya2VyLnB5',
        streaming:
          'Streaming:L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3B5dGhvbi9zdHJlYW1pbmcvd29ya2VyLnB5',
      },
    },
    timeout: {
      test_timeout: {
        '*': ':L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3B5dGhvbi90aW1lb3V0L3Rlc3RfdGltZW91dC5weQ__',
      },
      trigger: {
        '*': ':L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3B5dGhvbi90aW1lb3V0L3RyaWdnZXIucHk_',
      },
      worker: {
        '*': ':L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3B5dGhvbi90aW1lb3V0L3dvcmtlci5weQ__',
        scheduletimeout:
          'ScheduleTimeout:L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3B5dGhvbi90aW1lb3V0L3dvcmtlci5weQ__',
        executiontimeout:
          'ExecutionTimeout:L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3B5dGhvbi90aW1lb3V0L3dvcmtlci5weQ__',
        refreshtimeout:
          'RefreshTimeout:L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3B5dGhvbi90aW1lb3V0L3dvcmtlci5weQ__',
      },
    },
    waits: {
      test_waits: {
        '*': ':L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3B5dGhvbi93YWl0cy90ZXN0X3dhaXRzLnB5',
      },
      trigger: {
        '*': ':L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3B5dGhvbi93YWl0cy90cmlnZ2VyLnB5',
      },
      worker: {
        '*': ':L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3B5dGhvbi93YWl0cy93b3JrZXIucHk_',
        create_a_workflow:
          'Create a workflow:L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3B5dGhvbi93YWl0cy93b3JrZXIucHk_',
        add_base_task:
          'Add base task:L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3B5dGhvbi93YWl0cy93b3JrZXIucHk_',
        add_wait_for_sleep:
          'Add wait for sleep:L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3B5dGhvbi93YWl0cy93b3JrZXIucHk_',
        add_skip_on_event:
          'Add skip on event:L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3B5dGhvbi93YWl0cy93b3JrZXIucHk_',
        add_branching:
          'Add branching:L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3B5dGhvbi93YWl0cy93b3JrZXIucHk_',
        add_wait_for_event:
          'Add wait for event:L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3B5dGhvbi93YWl0cy93b3JrZXIucHk_',
        add_sum:
          'Add sum:L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3B5dGhvbi93YWl0cy93b3JrZXIucHk_',
      },
    },
    worker: {
      '*': ':L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3B5dGhvbi93b3JrZXIucHk_',
    },
    worker_existing_loop: {
      worker: {
        '*': ':L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3B5dGhvbi93b3JrZXJfZXhpc3RpbmdfbG9vcC93b3JrZXIucHk_',
      },
    },
    workflow_registration: {
      worker: {
        '*': ':L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3B5dGhvbi93b3JrZmxvd19yZWdpc3RyYXRpb24vd29ya2VyLnB5',
        workflowregistration:
          'WorkflowRegistration:L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3B5dGhvbi93b3JrZmxvd19yZWdpc3RyYXRpb24vd29ya2VyLnB5',
      },
    },
  },
  go: {
    migration_guides: {
      hatchet_client: {
        '*': ':L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL2dvL21pZ3JhdGlvbi1ndWlkZXMvaGF0Y2hldC1jbGllbnQuZ28_',
      },
      mergent: {
        '*': ':L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL2dvL21pZ3JhdGlvbi1ndWlkZXMvbWVyZ2VudC5nbw__',
        before_mergent:
          'Before (Mergent):L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL2dvL21pZ3JhdGlvbi1ndWlkZXMvbWVyZ2VudC5nbw__',
        after_hatchet:
          'After (Hatchet):L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL2dvL21pZ3JhdGlvbi1ndWlkZXMvbWVyZ2VudC5nbw__',
        running_a_task:
          'Running a task:L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL2dvL21pZ3JhdGlvbi1ndWlkZXMvbWVyZ2VudC5nbw__',
        declaring_a_worker:
          'Declaring a Worker:L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL2dvL21pZ3JhdGlvbi1ndWlkZXMvbWVyZ2VudC5nbw__',
        running_a_task_mergent:
          'Running a task (Mergent):L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL2dvL21pZ3JhdGlvbi1ndWlkZXMvbWVyZ2VudC5nbw__',
        running_a_task_hatchet:
          'Running a task (Hatchet):L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL2dvL21pZ3JhdGlvbi1ndWlkZXMvbWVyZ2VudC5nbw__',
        scheduling_tasks_hatchet:
          'Scheduling tasks (Hatchet):L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL2dvL21pZ3JhdGlvbi1ndWlkZXMvbWVyZ2VudC5nbw__',
      },
    },
    run: {
      all: {
        '*': ':L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL2dvL3J1bi9hbGwuZ28_',
      },
      bulk: {
        '*': ':L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL2dvL3J1bi9idWxrLmdv',
        bulk_run_tasks:
          'Bulk Run Tasks:L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL2dvL3J1bi9idWxrLmdv',
      },
      cron: {
        '*': ':L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL2dvL3J1bi9jcm9uLmdv',
        create:
          'Create:L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL2dvL3J1bi9jcm9uLmdv',
        delete:
          'Delete:L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL2dvL3J1bi9jcm9uLmdv',
        list: 'List:L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL2dvL3J1bi9jcm9uLmdv',
      },
      event: {
        '*': ':L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL2dvL3J1bi9ldmVudC5nbw__',
        pushing_an_event:
          'Pushing an Event:L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL2dvL3J1bi9ldmVudC5nbw__',
      },
      priority: {
        '*': ':L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL2dvL3J1bi9wcmlvcml0eS5nbw__',
        running_a_task_with_priority:
          'Running a Task with Priority:L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL2dvL3J1bi9wcmlvcml0eS5nbw__',
        schedule_and_cron:
          'Schedule and cron:L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL2dvL3J1bi9wcmlvcml0eS5nbw__',
      },
      simple: {
        '*': ':L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL2dvL3J1bi9zaW1wbGUuZ28_',
        running_a_task:
          'Running a Task:L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL2dvL3J1bi9zaW1wbGUuZ28_',
        running_multiple_tasks:
          'Running Multiple Tasks:L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL2dvL3J1bi9zaW1wbGUuZ28_',
        running_a_task_without_waiting:
          'Running a Task Without Waiting:L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL2dvL3J1bi9zaW1wbGUuZ28_',
        subscribing_to_results:
          'Subscribing to results:L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL2dvL3J1bi9zaW1wbGUuZ28_',
      },
    },
    worker: {
      start: {
        '*': ':L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL2dvL3dvcmtlci9zdGFydC5nbw__',
      },
    },
    workflows: {
      cancellations: {
        '*': ':L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL2dvL3dvcmtmbG93cy9jYW5jZWxsYXRpb25zLmdv',
        cancelled_task:
          'Cancelled task:L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL2dvL3dvcmtmbG93cy9jYW5jZWxsYXRpb25zLmdv',
      },
      child_workflows: {
        '*': ':L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL2dvL3dvcmtmbG93cy9jaGlsZC13b3JrZmxvd3MuZ28_',
      },
      complex_conditions: {
        '*': ':L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL2dvL3dvcmtmbG93cy9jb21wbGV4LWNvbmRpdGlvbnMuZ28_',
        create_a_workflow:
          'Create a workflow:L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL2dvL3dvcmtmbG93cy9jb21wbGV4LWNvbmRpdGlvbnMuZ28_',
        add_base_task:
          'Add base task:L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL2dvL3dvcmtmbG93cy9jb21wbGV4LWNvbmRpdGlvbnMuZ28_',
        add_wait_for_sleep:
          'Add wait for sleep:L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL2dvL3dvcmtmbG93cy9jb21wbGV4LWNvbmRpdGlvbnMuZ28_',
        add_skip_on_event:
          'Add skip on event:L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL2dvL3dvcmtmbG93cy9jb21wbGV4LWNvbmRpdGlvbnMuZ28_',
        add_branching:
          'Add branching:L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL2dvL3dvcmtmbG93cy9jb21wbGV4LWNvbmRpdGlvbnMuZ28_',
        add_wait_for_event:
          'Add wait for event:L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL2dvL3dvcmtmbG93cy9jb21wbGV4LWNvbmRpdGlvbnMuZ28_',
        add_sum:
          'Add sum:L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL2dvL3dvcmtmbG93cy9jb21wbGV4LWNvbmRpdGlvbnMuZ28_',
      },
      concurrency_rr: {
        '*': ':L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL2dvL3dvcmtmbG93cy9jb25jdXJyZW5jeS1yci5nbw__',
        concurrency_strategy_with_key:
          'Concurrency Strategy With Key:L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL2dvL3dvcmtmbG93cy9jb25jdXJyZW5jeS1yci5nbw__',
        multiple_concurrency_keys:
          'Multiple Concurrency Keys:L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL2dvL3dvcmtmbG93cy9jb25jdXJyZW5jeS1yci5nbw__',
      },
      dag_with_conditions: {
        '*': ':L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL2dvL3dvcmtmbG93cy9kYWctd2l0aC1jb25kaXRpb25zLmdv',
      },
      dag: {
        '*': ':L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL2dvL3dvcmtmbG93cy9kYWcuZ28_',
        declaring_a_workflow:
          'Declaring a Workflow:L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL2dvL3dvcmtmbG93cy9kYWcuZ28_',
        defining_a_task:
          'Defining a Task:L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL2dvL3dvcmtmbG93cy9kYWcuZ28_',
        adding_a_task_with_a_parent:
          'Adding a Task with a parent:L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL2dvL3dvcmtmbG93cy9kYWcuZ28_',
      },
      durable_event: {
        '*': ':L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL2dvL3dvcmtmbG93cy9kdXJhYmxlLWV2ZW50Lmdv',
        durable_event:
          'Durable Event:L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL2dvL3dvcmtmbG93cy9kdXJhYmxlLWV2ZW50Lmdv',
        durable_event_with_filter:
          'Durable Event With Filter:L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL2dvL3dvcmtmbG93cy9kdXJhYmxlLWV2ZW50Lmdv',
      },
      durable_sleep: {
        '*': ':L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL2dvL3dvcmtmbG93cy9kdXJhYmxlLXNsZWVwLmdv',
        durable_sleep:
          'Durable Sleep:L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL2dvL3dvcmtmbG93cy9kdXJhYmxlLXNsZWVwLmdv',
      },
      non_retryable_error: {
        '*': ':L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL2dvL3dvcmtmbG93cy9ub24tcmV0cnlhYmxlLWVycm9yLmdv',
        non_retryable_error:
          'Non Retryable Error:L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL2dvL3dvcmtmbG93cy9ub24tcmV0cnlhYmxlLWVycm9yLmdv',
      },
      on_cron: {
        '*': ':L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL2dvL3dvcmtmbG93cy9vbi1jcm9uLmdv',
        workflow_definition_cron_trigger:
          'Workflow Definition Cron Trigger:L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL2dvL3dvcmtmbG93cy9vbi1jcm9uLmdv',
      },
      on_event: {
        '*': ':L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL2dvL3dvcmtmbG93cy9vbi1ldmVudC5nbw__',
        run_workflow_on_event:
          'Run workflow on event:L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL2dvL3dvcmtmbG93cy9vbi1ldmVudC5nbw__',
      },
      on_failure: {
        '*': ':L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL2dvL3dvcmtmbG93cy9vbi1mYWlsdXJlLmdv',
      },
      priority: {
        '*': ':L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL2dvL3dvcmtmbG93cy9wcmlvcml0eS5nbw__',
        default_priority:
          'Default priority:L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL2dvL3dvcmtmbG93cy9wcmlvcml0eS5nbw__',
        defining_a_task:
          'Defining a Task:L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL2dvL3dvcmtmbG93cy9wcmlvcml0eS5nbw__',
      },
      ratelimit: {
        '*': ':L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL2dvL3dvcmtmbG93cy9yYXRlbGltaXQuZ28_',
        upsert_rate_limit:
          'Upsert Rate Limit:L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL2dvL3dvcmtmbG93cy9yYXRlbGltaXQuZ28_',
        static_rate_limit:
          'Static Rate Limit:L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL2dvL3dvcmtmbG93cy9yYXRlbGltaXQuZ28_',
        dynamic_rate_limit:
          'Dynamic Rate Limit:L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL2dvL3dvcmtmbG93cy9yYXRlbGltaXQuZ28_',
      },
      retries: {
        '*': ':L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL2dvL3dvcmtmbG93cy9yZXRyaWVzLmdv',
        simple_step_retries:
          'Simple Step Retries:L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL2dvL3dvcmtmbG93cy9yZXRyaWVzLmdv',
        retries_with_count:
          'Retries with Count:L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL2dvL3dvcmtmbG93cy9yZXRyaWVzLmdv',
        retries_with_backoff:
          'Retries with Backoff:L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL2dvL3dvcmtmbG93cy9yZXRyaWVzLmdv',
      },
      simple: {
        '*': ':L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL2dvL3dvcmtmbG93cy9zaW1wbGUuZ28_',
        declaring_a_task:
          'Declaring a Task:L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL2dvL3dvcmtmbG93cy9zaW1wbGUuZ28_',
        running_a_task:
          'Running a Task:L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL2dvL3dvcmtmbG93cy9zaW1wbGUuZ28_',
        declaring_a_worker:
          'Declaring a Worker:L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL2dvL3dvcmtmbG93cy9zaW1wbGUuZ28_',
        spawning_tasks_from_within_a_task:
          'Spawning Tasks from within a Task:L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL2dvL3dvcmtmbG93cy9zaW1wbGUuZ28_',
      },
      sticky: {
        '*': ':L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL2dvL3dvcmtmbG93cy9zdGlja3kuZ28_',
      },
      timeouts: {
        '*': ':L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL2dvL3dvcmtmbG93cy90aW1lb3V0cy5nbw__',
      },
    },
    z_v0: {
      assignment_affinity: {
        main: {
          '*': ':L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL2dvL3pfdjAvYXNzaWdubWVudC1hZmZpbml0eS9tYWluLmdv',
        },
        run: {
          '*': ':L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL2dvL3pfdjAvYXNzaWdubWVudC1hZmZpbml0eS9ydW4uZ28_',
        },
      },
      assignment_sticky: {
        main: {
          '*': ':L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL2dvL3pfdjAvYXNzaWdubWVudC1zdGlja3kvbWFpbi5nbw__',
        },
        run: {
          '*': ':L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL2dvL3pfdjAvYXNzaWdubWVudC1zdGlja3kvcnVuLmdv',
          stickyworker:
            'StickyWorker:L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL2dvL3pfdjAvYXNzaWdubWVudC1zdGlja3kvcnVuLmdv',
          stickychild:
            'StickyChild:L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL2dvL3pfdjAvYXNzaWdubWVudC1zdGlja3kvcnVuLmdv',
        },
      },
      bulk_imports: {
        main: {
          '*': ':L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL2dvL3pfdjAvYnVsa19pbXBvcnRzL21haW4uZ28_',
        },
      },
      bulk_workflows: {
        main: {
          '*': ':L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL2dvL3pfdjAvYnVsa193b3JrZmxvd3MvbWFpbi5nbw__',
        },
        run: {
          '*': ':L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL2dvL3pfdjAvYnVsa193b3JrZmxvd3MvcnVuLmdv',
        },
      },
      cancellation: {
        main: {
          '*': ':L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL2dvL3pfdjAvY2FuY2VsbGF0aW9uL21haW4uZ28_',
        },
        main_e2e_test: {
          '*': ':L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL2dvL3pfdjAvY2FuY2VsbGF0aW9uL21haW5fZTJlX3Rlc3QuZ28_',
        },
        run: {
          '*': ':L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL2dvL3pfdjAvY2FuY2VsbGF0aW9uL3J1bi5nbw__',
        },
      },
      compute: {
        main: {
          '*': ':L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL2dvL3pfdjAvY29tcHV0ZS9tYWluLmdv',
        },
      },
      concurrency: {
        main: {
          '*': ':L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL2dvL3pfdjAvY29uY3VycmVuY3kvbWFpbi5nbw__',
        },
        main_e2e_test: {
          '*': ':L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL2dvL3pfdjAvY29uY3VycmVuY3kvbWFpbl9lMmVfdGVzdC5nbw__',
        },
      },
      cron: {
        main: {
          '*': ':L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL2dvL3pfdjAvY3Jvbi9tYWluLmdv',
          workflow_definition_cron_trigger:
            'Workflow Definition Cron Trigger:L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL2dvL3pfdjAvY3Jvbi9tYWluLmdv',
        },
      },
      cron_programmatic: {
        main: {
          '*': ':L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL2dvL3pfdjAvY3Jvbi1wcm9ncmFtbWF0aWMvbWFpbi5nbw__',
          create:
            'Create:L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL2dvL3pfdjAvY3Jvbi1wcm9ncmFtbWF0aWMvbWFpbi5nbw__',
          list: 'List:L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL2dvL3pfdjAvY3Jvbi1wcm9ncmFtbWF0aWMvbWFpbi5nbw__',
          delete:
            'Delete:L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL2dvL3pfdjAvY3Jvbi1wcm9ncmFtbWF0aWMvbWFpbi5nbw__',
        },
      },
      dag: {
        main: {
          '*': ':L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL2dvL3pfdjAvZGFnL21haW4uZ28_',
        },
      },
      deprecated: {
        requeue: {
          main: {
            '*': ':L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL2dvL3pfdjAvZGVwcmVjYXRlZC9yZXF1ZXVlL21haW4uZ28_',
          },
        },
        schedule_timeout: {
          main: {
            '*': ':L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL2dvL3pfdjAvZGVwcmVjYXRlZC9zY2hlZHVsZS10aW1lb3V0L21haW4uZ28_',
          },
        },
        timeout: {
          main: {
            '*': ':L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL2dvL3pfdjAvZGVwcmVjYXRlZC90aW1lb3V0L21haW4uZ28_',
          },
        },
        yaml: {
          main: {
            '*': ':L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL2dvL3pfdjAvZGVwcmVjYXRlZC95YW1sL21haW4uZ28_',
          },
        },
      },
      errors_test: {
        main: {
          '*': ':L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL2dvL3pfdjAvZXJyb3JzLXRlc3QvbWFpbi5nbw__',
        },
      },
      limit_concurrency: {
        cancel_in_progress: {
          main: {
            '*': ':L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL2dvL3pfdjAvbGltaXQtY29uY3VycmVuY3kvY2FuY2VsLWluLXByb2dyZXNzL21haW4uZ28_',
          },
        },
        group_round_robin: {
          main: {
            '*': ':L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL2dvL3pfdjAvbGltaXQtY29uY3VycmVuY3kvZ3JvdXAtcm91bmQtcm9iaW4vbWFpbi5nbw__',
          },
        },
        group_round_robin_advanced: {
          main: {
            '*': ':L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL2dvL3pfdjAvbGltaXQtY29uY3VycmVuY3kvZ3JvdXAtcm91bmQtcm9iaW4tYWR2YW5jZWQvbWFpbi5nbw__',
          },
          main_e2e_test: {
            '*': ':L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL2dvL3pfdjAvbGltaXQtY29uY3VycmVuY3kvZ3JvdXAtcm91bmQtcm9iaW4tYWR2YW5jZWQvbWFpbl9lMmVfdGVzdC5nbw__',
          },
        },
      },
      loadtest: {
        cli: {
          cli_e2e_test: {
            '*': ':L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL2dvL3pfdjAvbG9hZHRlc3QvY2xpL2NsaV9lMmVfdGVzdC5nbw__',
          },
          do: {
            '*': ':L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL2dvL3pfdjAvbG9hZHRlc3QvY2xpL2RvLmdv',
          },
          emit: {
            '*': ':L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL2dvL3pfdjAvbG9hZHRlc3QvY2xpL2VtaXQuZ28_',
          },
          main: {
            '*': ':L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL2dvL3pfdjAvbG9hZHRlc3QvY2xpL21haW4uZ28_',
          },
          run: {
            '*': ':L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL2dvL3pfdjAvbG9hZHRlc3QvY2xpL3J1bi5nbw__',
          },
        },
        emitter: {
          main: {
            '*': ':L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL2dvL3pfdjAvbG9hZHRlc3QvZW1pdHRlci9tYWluLmdv',
          },
        },
        rampup: {
          do: {
            '*': ':L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL2dvL3pfdjAvbG9hZHRlc3QvcmFtcHVwL2RvLmdv',
          },
          emit: {
            '*': ':L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL2dvL3pfdjAvbG9hZHRlc3QvcmFtcHVwL2VtaXQuZ28_',
          },
          ramp_up_e2e_test: {
            '*': ':L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL2dvL3pfdjAvbG9hZHRlc3QvcmFtcHVwL3JhbXBfdXBfZTJlX3Rlc3QuZ28_',
          },
          run: {
            '*': ':L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL2dvL3pfdjAvbG9hZHRlc3QvcmFtcHVwL3J1bi5nbw__',
          },
        },
        worker: {
          main: {
            '*': ':L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL2dvL3pfdjAvbG9hZHRlc3Qvd29ya2VyL21haW4uZ28_',
          },
        },
      },
      logging: {
        main: {
          '*': ':L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL2dvL3pfdjAvbG9nZ2luZy9tYWluLmdv',
        },
      },
      manual_trigger: {
        trigger: {
          main: {
            '*': ':L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL2dvL3pfdjAvbWFudWFsLXRyaWdnZXIvdHJpZ2dlci9tYWluLmdv',
          },
        },
        worker: {
          main: {
            '*': ':L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL2dvL3pfdjAvbWFudWFsLXRyaWdnZXIvd29ya2VyL21haW4uZ28_',
          },
        },
      },
      middleware: {
        main: {
          '*': ':L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL2dvL3pfdjAvbWlkZGxld2FyZS9tYWluLmdv',
        },
        main_e2e_test: {
          '*': ':L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL2dvL3pfdjAvbWlkZGxld2FyZS9tYWluX2UyZV90ZXN0Lmdv',
        },
        run: {
          '*': ':L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL2dvL3pfdjAvbWlkZGxld2FyZS9ydW4uZ28_',
        },
      },
      namespaced: {
        main: {
          '*': ':L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL2dvL3pfdjAvbmFtZXNwYWNlZC9tYWluLmdv',
        },
      },
      no_tls: {
        main: {
          '*': ':L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL2dvL3pfdjAvbm8tdGxzL21haW4uZ28_',
        },
      },
      on_failure: {
        main: {
          '*': ':L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL2dvL3pfdjAvb24tZmFpbHVyZS9tYWluLmdv',
          onfailure_step:
            'OnFailure Step:L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL2dvL3pfdjAvb24tZmFpbHVyZS9tYWluLmdv',
        },
      },
      procedural: {
        main: {
          '*': ':L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL2dvL3pfdjAvcHJvY2VkdXJhbC9tYWluLmdv',
        },
        main_e2e_test: {
          '*': ':L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL2dvL3pfdjAvcHJvY2VkdXJhbC9tYWluX2UyZV90ZXN0Lmdv',
        },
      },
      rate_limit: {
        main: {
          '*': ':L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL2dvL3pfdjAvcmF0ZS1saW1pdC9tYWluLmdv',
        },
      },
      register_action: {
        main: {
          '*': ':L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL2dvL3pfdjAvcmVnaXN0ZXItYWN0aW9uL21haW4uZ28_',
        },
      },
      retries: {
        main: {
          '*': ':L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL2dvL3pfdjAvcmV0cmllcy9tYWluLmdv',
        },
      },
      retries_with_backoff: {
        main: {
          '*': ':L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL2dvL3pfdjAvcmV0cmllcy13aXRoLWJhY2tvZmYvbWFpbi5nbw__',
          backoff:
            'Backoff:L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL2dvL3pfdjAvcmV0cmllcy13aXRoLWJhY2tvZmYvbWFpbi5nbw__',
        },
      },
      scheduled: {
        main: {
          '*': ':L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL2dvL3pfdjAvc2NoZWR1bGVkL21haW4uZ28_',
          create:
            'Create:L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL2dvL3pfdjAvc2NoZWR1bGVkL21haW4uZ28_',
          list: 'List:L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL2dvL3pfdjAvc2NoZWR1bGVkL21haW4uZ28_',
          delete:
            'Delete:L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL2dvL3pfdjAvc2NoZWR1bGVkL21haW4uZ28_',
        },
      },
      simple: {
        main: {
          '*': ':L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL2dvL3pfdjAvc2ltcGxlL21haW4uZ28_',
        },
        main_e2e_test: {
          '*': ':L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL2dvL3pfdjAvc2ltcGxlL21haW5fZTJlX3Rlc3QuZ28_',
        },
      },
      stream_event: {
        main: {
          '*': ':L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL2dvL3pfdjAvc3RyZWFtLWV2ZW50L21haW4uZ28_',
        },
      },
      stream_event_by_meta: {
        main: {
          '*': ':L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL2dvL3pfdjAvc3RyZWFtLWV2ZW50LWJ5LW1ldGEvbWFpbi5nbw__',
        },
      },
      timeout: {
        main: {
          '*': ':L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL2dvL3pfdjAvdGltZW91dC9tYWluLmdv',
          timeoutstep:
            'TimeoutStep:L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL2dvL3pfdjAvdGltZW91dC9tYWluLmdv',
        },
        main_e2e_test: {
          '*': ':L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL2dvL3pfdjAvdGltZW91dC9tYWluX2UyZV90ZXN0Lmdv',
        },
        run: {
          '*': ':L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL2dvL3pfdjAvdGltZW91dC9ydW4uZ28_',
        },
      },
      webhook: {
        main: {
          '*': ':L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL2dvL3pfdjAvd2ViaG9vay9tYWluLmdv',
        },
        main_e2e_test: {
          '*': ':L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL2dvL3pfdjAvd2ViaG9vay9tYWluX2UyZV90ZXN0Lmdv',
        },
        run: {
          '*': ':L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL2dvL3pfdjAvd2ViaG9vay9ydW4uZ28_',
        },
      },
    },
  },
} as const;

export default snips;
