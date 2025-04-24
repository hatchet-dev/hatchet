// This file is auto-generated. Do not edit directly.

// Types for snippets
type Snippet = {
  content: string;
  language: string;
  source: string;
};

type Snippets = {
  [key: string]: Snippet;
};

// Snippet contents
export const snippets: Snippets = {
  "L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3R5cGVzY3JpcHQvY2FuY2VsbGF0aW9ucy9ydW4udHM_": {
    "content": "/* eslint-disable no-console */\n// ❓ Running a Task with Results\nimport sleep from '@hatchet/util/sleep';\nimport { cancellation } from './workflow';\nimport { hatchet } from '../hatchet-client';\n// ...\nasync function main() {\n  const run = cancellation.runNoWait({});\n  const run1 = cancellation.runNoWait({});\n\n  await sleep(1000);\n\n  await run.cancel();\n\n  const res = await run.output;\n  const res1 = await run1.output;\n\n  console.log('canceled', res);\n  console.log('completed', res1);\n\n  await sleep(1000);\n\n  await run.replay();\n\n  const resReplay = await run.output;\n\n  console.log(resReplay);\n\n  const run2 = cancellation.runNoWait({}, { additionalMetadata: { test: 'abc' } });\n  const run4 = cancellation.runNoWait({}, { additionalMetadata: { test: 'test' } });\n\n  await sleep(1000);\n\n  await hatchet.runs.cancel({\n    filters: {\n      since: new Date(Date.now() - 60 * 60),\n      additionalMetadata: { test: 'test' },\n    },\n  });\n\n  const res3 = await Promise.all([run2.output, run4.output]);\n  console.log(res3);\n  // !!\n}\n\nif (require.main === module) {\n  main()\n    .catch(console.error)\n    .finally(() => process.exit(0));\n}\n",
    "language": "ts",
    "source": "/Users/gabrielruttner/dev/hatchet/examples/typescript/cancellations/run.ts"
  },
  "L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3R5cGVzY3JpcHQvY2FuY2VsbGF0aW9ucy93b3JrZXIudHM_": {
    "content": "// ❓ Declaring a Worker\nimport { hatchet } from '../hatchet-client';\nimport { cancellation } from './workflow';\n\nasync function main() {\n  const worker = await hatchet.worker('cancellation-worker', {\n    // 👀 Declare the workflows that the worker can execute\n    workflows: [cancellation],\n    // 👀 Declare the number of concurrent task runs the worker can accept\n    slots: 100,\n  });\n\n  await worker.start();\n}\n\nif (require.main === module) {\n  main();\n}\n// !!\n",
    "language": "ts",
    "source": "/Users/gabrielruttner/dev/hatchet/examples/typescript/cancellations/worker.ts"
  },
  "L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3R5cGVzY3JpcHQvY2FuY2VsbGF0aW9ucy93b3JrZmxvdy50cw__": {
    "content": "import sleep from '@hatchet/util/sleep';\nimport axios from 'axios';\nimport { hatchet } from '../hatchet-client';\n\n// ❓ Declaring a Task\nexport const cancellation = hatchet.task({\n  name: 'cancellation',\n  fn: async (_, { cancelled }) => {\n    await sleep(10 * 1000);\n\n    if (cancelled) {\n      throw new Error('Task was cancelled');\n    }\n\n    return {\n      Completed: true,\n    };\n  },\n});\n// !!\n\n// ❓ Abort Signal\nexport const abortSignal = hatchet.task({\n  name: 'abort-signal',\n  fn: async (_, { controller }) => {\n    try {\n      const response = await axios.get('https://api.example.com/data', {\n        signal: controller.signal,\n      });\n      // Handle the response\n    } catch (error) {\n      if (axios.isCancel(error)) {\n        // Request was canceled\n        console.log('Request canceled');\n      } else {\n        // Handle other errors\n      }\n    }\n  },\n});\n// !!\n\n// see ./worker.ts and ./run.ts for how to run the workflow\n",
    "language": "ts",
    "source": "/Users/gabrielruttner/dev/hatchet/examples/typescript/cancellations/workflow.ts"
  },
  "L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3R5cGVzY3JpcHQvY2hpbGRfd29ya2Zsb3dzL3J1bi50cw__": {
    "content": "import { parent } from './workflow';\n\nasync function main() {\n  const res = await parent.run({\n    N: 10,\n  });\n\n  // eslint-disable-next-line no-console\n  console.log(res.Result);\n}\n\nif (require.main === module) {\n  main()\n    .then(() => process.exit(0))\n    .catch((error) => {\n      // eslint-disable-next-line no-console\n      console.error('Error:', error);\n      process.exit(1);\n    });\n}\n",
    "language": "ts",
    "source": "/Users/gabrielruttner/dev/hatchet/examples/typescript/child_workflows/run.ts"
  },
  "L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3R5cGVzY3JpcHQvY2hpbGRfd29ya2Zsb3dzL3dvcmtlci50cw__": {
    "content": "import { hatchet } from '../hatchet-client';\nimport { parent, child } from './workflow';\n\nasync function main() {\n  const worker = await hatchet.worker('child-workflow-worker', {\n    workflows: [parent, child],\n    slots: 100,\n  });\n\n  await worker.start();\n}\n\nif (require.main === module) {\n  main();\n}\n",
    "language": "ts",
    "source": "/Users/gabrielruttner/dev/hatchet/examples/typescript/child_workflows/worker.ts"
  },
  "L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3R5cGVzY3JpcHQvY2hpbGRfd29ya2Zsb3dzL3dvcmtmbG93LnRz": {
    "content": "/* eslint-disable no-plusplus */\n// ❓ Declaring a Child\nimport { hatchet } from '../hatchet-client';\n\ntype ChildInput = {\n  N: number;\n};\n\nexport const child = hatchet.task({\n  name: 'child',\n  fn: (input: ChildInput) => {\n    return {\n      Value: input.N,\n    };\n  },\n});\n// !!\n\n// ❓ Declaring a Parent\n\ntype ParentInput = {\n  N: number;\n};\n\nexport const parent = hatchet.task({\n  name: 'parent',\n  fn: async (input: ParentInput, ctx) => {\n    const n = input.N;\n    const promises = [];\n\n    for (let i = 0; i < n; i++) {\n      promises.push(ctx.runChild(child, { N: i }));\n    }\n\n    const childRes = await Promise.all(promises);\n    const sum = childRes.reduce((acc, curr) => acc + curr.Value, 0);\n\n    return {\n      Result: sum,\n    };\n  },\n});\n// !!\n",
    "language": "ts",
    "source": "/Users/gabrielruttner/dev/hatchet/examples/typescript/child_workflows/workflow.ts"
  },
  "L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3R5cGVzY3JpcHQvY29uY3VycmVuY3ktcnIvcnVuLnRz": {
    "content": "import { simpleConcurrency } from './workflow';\n\nasync function main() {\n  const res = await simpleConcurrency.run([\n    {\n      Message: 'Hello World',\n      GroupKey: 'A',\n    },\n    {\n      Message: 'Goodbye Moon',\n      GroupKey: 'A',\n    },\n    {\n      Message: 'Hello World B',\n      GroupKey: 'B',\n    },\n  ]);\n\n  // eslint-disable-next-line no-console\n  console.log(res[0]['to-lower'].TransformedMessage);\n  // eslint-disable-next-line no-console\n  console.log(res[1]['to-lower'].TransformedMessage);\n  // eslint-disable-next-line no-console\n  console.log(res[2]['to-lower'].TransformedMessage);\n}\n\nif (require.main === module) {\n  main().then(() => process.exit(0));\n}\n",
    "language": "ts",
    "source": "/Users/gabrielruttner/dev/hatchet/examples/typescript/concurrency-rr/run.ts"
  },
  "L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3R5cGVzY3JpcHQvY29uY3VycmVuY3ktcnIvd29ya2VyLnRz": {
    "content": "import { hatchet } from '../hatchet-client';\nimport { simpleConcurrency } from './workflow';\n\nasync function main() {\n  const worker = await hatchet.worker('simple-concurrency-worker', {\n    workflows: [simpleConcurrency],\n  });\n\n  await worker.start();\n}\n\nif (require.main === module) {\n  main();\n}\n",
    "language": "ts",
    "source": "/Users/gabrielruttner/dev/hatchet/examples/typescript/concurrency-rr/worker.ts"
  },
  "L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3R5cGVzY3JpcHQvY29uY3VycmVuY3ktcnIvd29ya2Zsb3cudHM_": {
    "content": "import { ConcurrencyLimitStrategy } from '@hatchet/workflow';\nimport { hatchet } from '../hatchet-client';\n\ntype SimpleInput = {\n  Message: string;\n  GroupKey: string;\n};\n\ntype SimpleOutput = {\n  'to-lower': {\n    TransformedMessage: string;\n  };\n};\n\nconst sleep = (ms: number) =>\n  new Promise((resolve) => {\n    setTimeout(resolve, ms);\n  });\n\n// ❓ Concurrency Strategy With Key\nexport const simpleConcurrency = hatchet.workflow<SimpleInput, SimpleOutput>({\n  name: 'simple-concurrency',\n  concurrency: {\n    maxRuns: 1,\n    limitStrategy: ConcurrencyLimitStrategy.GROUP_ROUND_ROBIN,\n    expression: 'input.GroupKey',\n  },\n});\n// !!\n\nsimpleConcurrency.task({\n  name: 'to-lower',\n  fn: async (input) => {\n    await sleep(Math.floor(Math.random() * (1000 - 200 + 1)) + 200);\n    return {\n      TransformedMessage: input.Message.toLowerCase(),\n    };\n  },\n});\n\n// ❓ Multiple Concurrency Keys\nexport const multipleConcurrencyKeys = hatchet.workflow<SimpleInput, SimpleOutput>({\n  name: 'simple-concurrency',\n  concurrency: [\n    {\n      maxRuns: 1,\n      limitStrategy: ConcurrencyLimitStrategy.GROUP_ROUND_ROBIN,\n      expression: 'input.Tier',\n    },\n    {\n      maxRuns: 1,\n      limitStrategy: ConcurrencyLimitStrategy.GROUP_ROUND_ROBIN,\n      expression: 'input.Account',\n    },\n  ],\n});\n// !!\n\nmultipleConcurrencyKeys.task({\n  name: 'to-lower',\n  fn: async (input) => {\n    await sleep(Math.floor(Math.random() * (1000 - 200 + 1)) + 200);\n    return {\n      TransformedMessage: input.Message.toLowerCase(),\n    };\n  },\n});\n",
    "language": "ts",
    "source": "/Users/gabrielruttner/dev/hatchet/examples/typescript/concurrency-rr/workflow.ts"
  },
  "L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3R5cGVzY3JpcHQvZGFnL3J1bi50cw__": {
    "content": "import { dag } from './workflow';\n\nasync function main() {\n  const res = await dag.run({\n    Message: 'hello world',\n  });\n\n  // eslint-disable-next-line no-console\n  console.log(res.reverse.Transformed);\n}\n\nif (require.main === module) {\n  main();\n}\n",
    "language": "ts",
    "source": "/Users/gabrielruttner/dev/hatchet/examples/typescript/dag/run.ts"
  },
  "L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3R5cGVzY3JpcHQvZGFnL3dvcmtlci50cw__": {
    "content": "import { hatchet } from '../hatchet-client';\nimport { dag } from './workflow';\n\nasync function main() {\n  const worker = await hatchet.worker('dag-worker', {\n    workflows: [dag],\n  });\n\n  await worker.start();\n}\n\nif (require.main === module) {\n  main();\n}\n",
    "language": "ts",
    "source": "/Users/gabrielruttner/dev/hatchet/examples/typescript/dag/worker.ts"
  },
  "L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3R5cGVzY3JpcHQvZGFnL3dvcmtmbG93LnRz": {
    "content": "import { hatchet } from '../hatchet-client';\n\ntype DagInput = {\n  Message: string;\n};\n\ntype DagOutput = {\n  reverse: {\n    Original: string;\n    Transformed: string;\n  };\n};\n\n// ❓ Declaring a DAG Workflow\n// First, we declare the workflow\nexport const dag = hatchet.workflow<DagInput, DagOutput>({\n  name: 'simple',\n});\n\n// Next, we declare the tasks bound to the workflow\nconst toLower = dag.task({\n  name: 'to-lower',\n  fn: (input) => {\n    return {\n      TransformedMessage: input.Message.toLowerCase(),\n    };\n  },\n});\n\n// Next, we declare the tasks bound to the workflow\ndag.task({\n  name: 'reverse',\n  parents: [toLower],\n  fn: async (input, ctx) => {\n    const lower = await ctx.parentOutput(toLower);\n    return {\n      Original: input.Message,\n      Transformed: lower.TransformedMessage.split('').reverse().join(''),\n    };\n  },\n});\n// !!\n",
    "language": "ts",
    "source": "/Users/gabrielruttner/dev/hatchet/examples/typescript/dag/workflow.ts"
  },
  "L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3R5cGVzY3JpcHQvZGFnX21hdGNoX2NvbmRpdGlvbi9ydW4udHM_": {
    "content": "/* eslint-disable no-console */\nimport { dagWithConditions } from './workflow';\n\nasync function main() {\n  const res = await dagWithConditions.run({});\n\n  console.log(res['first-task'].Completed);\n  console.log(res['second-task'].Completed);\n}\n\nif (require.main === module) {\n  main()\n    .catch(console.error)\n    .finally(() => process.exit(0));\n}\n",
    "language": "ts",
    "source": "/Users/gabrielruttner/dev/hatchet/examples/typescript/dag_match_condition/run.ts"
  },
  "L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3R5cGVzY3JpcHQvZGFnX21hdGNoX2NvbmRpdGlvbi93b3JrZXIudHM_": {
    "content": "import { hatchet } from '../hatchet-client';\nimport { dagWithConditions } from './workflow';\n\nasync function main() {\n  const worker = await hatchet.worker('dag-worker', {\n    workflows: [dagWithConditions],\n  });\n\n  await worker.start();\n}\n\nif (require.main === module) {\n  main();\n}\n",
    "language": "ts",
    "source": "/Users/gabrielruttner/dev/hatchet/examples/typescript/dag_match_condition/worker.ts"
  },
  "L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3R5cGVzY3JpcHQvZGFnX21hdGNoX2NvbmRpdGlvbi93b3JrZmxvdy50cw__": {
    "content": "import sleep from '@hatchet/util/sleep';\nimport { Or } from '@hatchet-dev/typescript-sdk/conditions';\nimport { hatchet } from '../hatchet-client';\n\ntype DagInput = {};\n\ntype DagOutput = {\n  'first-task': {\n    Completed: boolean;\n  };\n  'second-task': {\n    Completed: boolean;\n  };\n};\n\nexport const dagWithConditions = hatchet.workflow<DagInput, DagOutput>({\n  name: 'simple',\n});\n\nconst firstTask = dagWithConditions.task({\n  name: 'first-task',\n  fn: async () => {\n    await sleep(2000);\n    return {\n      Completed: true,\n    };\n  },\n});\n\ndagWithConditions.task({\n  name: 'second-task',\n  parents: [firstTask],\n  waitFor: Or({ eventKey: 'user:event' }, { sleepFor: '10s' }),\n  fn: async (_, ctx) => {\n    console.log('triggered by condition', ctx.triggers());\n\n    return {\n      Completed: true,\n    };\n  },\n});\n",
    "language": "ts",
    "source": "/Users/gabrielruttner/dev/hatchet/examples/typescript/dag_match_condition/workflow.ts"
  },
  "L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3R5cGVzY3JpcHQvZGVlcC9ydW4udHM_": {
    "content": "import { parent } from './workflow';\n\nasync function main() {\n  const res = await parent.run({\n    Message: 'hello',\n    N: 5,\n  });\n\n  // eslint-disable-next-line no-console\n  console.log(res.parent.Sum);\n}\n\nif (require.main === module) {\n  main()\n    // eslint-disable-next-line no-console\n    .catch(console.error)\n    .finally(() => process.exit(0));\n}\n",
    "language": "ts",
    "source": "/Users/gabrielruttner/dev/hatchet/examples/typescript/deep/run.ts"
  },
  "L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3R5cGVzY3JpcHQvZGVlcC93b3JrZXIudHM_": {
    "content": "import { hatchet } from '../hatchet-client';\nimport { parent, child1, child2, child3, child4, child5 } from './workflow';\n\nasync function main() {\n  const worker = await hatchet.worker('simple-worker', {\n    workflows: [parent, child1, child2, child3, child4, child5],\n    slots: 5000,\n  });\n\n  await worker.start();\n}\n\nif (require.main === module) {\n  main();\n}\n",
    "language": "ts",
    "source": "/Users/gabrielruttner/dev/hatchet/examples/typescript/deep/worker.ts"
  },
  "L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3R5cGVzY3JpcHQvZGVlcC93b3JrZmxvdy50cw__": {
    "content": "import sleep from '@hatchet/util/sleep';\nimport { hatchet } from '../hatchet-client';\n\ntype SimpleInput = {\n  Message: string;\n  N: number;\n};\n\ntype Output = {\n  transformer: {\n    Sum: number;\n  };\n};\n\nexport const child1 = hatchet.workflow<SimpleInput, Output>({\n  name: 'child1',\n});\n\nchild1.task({\n  name: 'transformer',\n  fn: () => {\n    sleep(15);\n    return {\n      Sum: 1,\n    };\n  },\n});\n\nexport const child2 = hatchet.workflow<SimpleInput, Output>({\n  name: 'child2',\n});\n\nchild2.task({\n  name: 'transformer',\n  fn: async (input, ctx) => {\n    const count = input.N;\n    const promises = Array(count)\n      .fill(null)\n      .map(() => ({ workflow: child1, input }));\n\n    const results = await ctx.bulkRunChildren(promises);\n\n    sleep(15);\n    return {\n      Sum: results.reduce((acc, r) => acc + r.transformer.Sum, 0),\n    };\n  },\n});\n\nexport const child3 = hatchet.workflow<SimpleInput, Output>({\n  name: 'child3',\n});\n\nchild3.task({\n  name: 'transformer',\n  fn: async (input, ctx) => {\n    const count = input.N;\n    const promises = Array(count)\n      .fill(null)\n      .map(() => ({ workflow: child2, input }));\n\n    const results = await ctx.bulkRunChildren(promises);\n\n    return {\n      Sum: results.reduce((acc, r) => acc + r.transformer.Sum, 0),\n    };\n  },\n});\n\nexport const child4 = hatchet.workflow<SimpleInput, Output>({\n  name: 'child4',\n});\n\nchild4.task({\n  name: 'transformer',\n  fn: async (input, ctx) => {\n    const count = input.N;\n    const promises = Array(count)\n      .fill(null)\n      .map(() => ({ workflow: child3, input }));\n\n    const results = await ctx.bulkRunChildren(promises);\n\n    return {\n      Sum: results.reduce((acc, r) => acc + r.transformer.Sum, 0),\n    };\n  },\n});\n\nexport const child5 = hatchet.workflow<SimpleInput, Output>({\n  name: 'child5',\n});\n\nchild5.task({\n  name: 'transformer',\n  fn: async (input, ctx) => {\n    const count = input.N;\n    const promises = Array(count)\n      .fill(null)\n      .map(() => ({ workflow: child4, input }));\n\n    const results = await ctx.bulkRunChildren(promises);\n\n    return {\n      Sum: results.reduce((acc, r) => acc + r.transformer.Sum, 0),\n    };\n  },\n});\n\nexport const parent = hatchet.workflow<SimpleInput, { parent: Output['transformer'] }>({\n  name: 'parent',\n});\n\nparent.task({\n  name: 'parent',\n  fn: async (input, ctx) => {\n    const count = input.N; // Random number between 2-4\n    const promises = Array(count)\n      .fill(null)\n      .map(() => ({ workflow: child5, input }));\n\n    const results = await ctx.bulkRunChildren(promises);\n\n    return {\n      Sum: results.reduce((acc, r) => acc + r.transformer.Sum, 0),\n    };\n  },\n});\n",
    "language": "ts",
    "source": "/Users/gabrielruttner/dev/hatchet/examples/typescript/deep/workflow.ts"
  },
  "L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3R5cGVzY3JpcHQvZHVyYWJsZS1ldmVudC9ydW4udHM_": {
    "content": "import { durableEvent } from './workflow';\n\nasync function main() {\n  const timeStart = Date.now();\n  const res = await durableEvent.run({});\n  const timeEnd = Date.now();\n  // eslint-disable-next-line no-console\n  console.log(`Time taken: ${timeEnd - timeStart}ms`);\n}\n\nif (require.main === module) {\n  main()\n    .then(() => process.exit(0))\n    .catch((error) => {\n      // eslint-disable-next-line no-console\n      console.error('Error:', error);\n      process.exit(1);\n    });\n}\n",
    "language": "ts",
    "source": "/Users/gabrielruttner/dev/hatchet/examples/typescript/durable-event/run.ts"
  },
  "L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3R5cGVzY3JpcHQvZHVyYWJsZS1ldmVudC93b3JrZXIudHM_": {
    "content": "import { hatchet } from '../hatchet-client';\nimport { durableEvent } from './workflow';\n\nasync function main() {\n  const worker = await hatchet.worker('durable-event-worker', {\n    workflows: [durableEvent],\n  });\n\n  await worker.start();\n}\n\nif (require.main === module) {\n  main();\n}\n",
    "language": "ts",
    "source": "/Users/gabrielruttner/dev/hatchet/examples/typescript/durable-event/worker.ts"
  },
  "L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3R5cGVzY3JpcHQvZHVyYWJsZS1ldmVudC93b3JrZmxvdy50cw__": {
    "content": "// import sleep from '@hatchet/util/sleep';\nimport { hatchet } from '../hatchet-client';\n\n// ❓ Durable Event\nexport const durableEvent = hatchet.durableTask({\n  name: 'durable-event',\n  executionTimeout: '10m',\n  fn: async (_, ctx) => {\n    const res = ctx.waitFor({\n      eventKey: 'user:update',\n    });\n\n    console.log('res', res);\n\n    return {\n      Value: 'done',\n    };\n  },\n});\n// !!\n\nexport const durableEventWithFilter = hatchet.durableTask({\n  name: 'durable-event-with-filter',\n  executionTimeout: '10m',\n  fn: async (_, ctx) => {\n    // ❓ Durable Event With Filter\n    const res = ctx.waitFor({\n      eventKey: 'user:update',\n      expression: \"input.userId == '1234'\",\n    });\n    // !!\n\n    console.log('res', res);\n\n    return {\n      Value: 'done',\n    };\n  },\n});\n// !!\n",
    "language": "ts",
    "source": "/Users/gabrielruttner/dev/hatchet/examples/typescript/durable-event/workflow.ts"
  },
  "L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3R5cGVzY3JpcHQvZHVyYWJsZS1zbGVlcC9ydW4udHM_": {
    "content": "import { durableSleep } from './workflow';\n\nasync function main() {\n  const timeStart = Date.now();\n  const res = await durableSleep.run({});\n  const timeEnd = Date.now();\n  // eslint-disable-next-line no-console\n  console.log(`Time taken: ${timeEnd - timeStart}ms`);\n}\n\nif (require.main === module) {\n  main()\n    .then(() => process.exit(0))\n    .catch((error) => {\n      // eslint-disable-next-line no-console\n      console.error('Error:', error);\n      process.exit(1);\n    });\n}\n",
    "language": "ts",
    "source": "/Users/gabrielruttner/dev/hatchet/examples/typescript/durable-sleep/run.ts"
  },
  "L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3R5cGVzY3JpcHQvZHVyYWJsZS1zbGVlcC93b3JrZXIudHM_": {
    "content": "import { hatchet } from '../hatchet-client';\nimport { durableSleep } from './workflow';\n\nasync function main() {\n  const worker = await hatchet.worker('sleep-worker', {\n    workflows: [durableSleep],\n  });\n\n  await worker.start();\n}\n\nif (require.main === module) {\n  main();\n}\n",
    "language": "ts",
    "source": "/Users/gabrielruttner/dev/hatchet/examples/typescript/durable-sleep/worker.ts"
  },
  "L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3R5cGVzY3JpcHQvZHVyYWJsZS1zbGVlcC93b3JrZmxvdy50cw__": {
    "content": "// import sleep from '@hatchet/util/sleep';\nimport { hatchet } from '../hatchet-client';\n\nexport const durableSleep = hatchet.workflow({\n  name: 'durable-sleep',\n});\n\n// ❓ Durable Sleep\ndurableSleep.durableTask({\n  name: 'durable-sleep',\n  executionTimeout: '10m',\n  fn: async (_, ctx) => {\n    console.log('sleeping for 5s');\n    const sleepRes = await ctx.sleepFor('5s');\n    console.log('done sleeping for 5s', sleepRes);\n\n    return {\n      Value: 'done',\n    };\n  },\n});\n// !!\n",
    "language": "ts",
    "source": "/Users/gabrielruttner/dev/hatchet/examples/typescript/durable-sleep/workflow.ts"
  },
  "L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3R5cGVzY3JpcHQvaW5mZXJyZWQtdHlwaW5nL3J1bi50cw__": {
    "content": "/* eslint-disable no-console */\nimport { crazyWorkflow, declaredType, inferredType, inferredTypeDurable } from './workflow';\n\nasync function main() {\n  const declaredTypeRun = declaredType.run({\n    Message: 'hello',\n  });\n\n  const inferredTypeRun = inferredType.run({\n    Message: 'hello',\n  });\n\n  const crazyWorkflowRun = crazyWorkflow.run({\n    Message: 'hello',\n  });\n\n  const inferredTypeDurableRun = inferredTypeDurable.run({\n    Message: 'Durable Task',\n  });\n\n  const [declaredTypeResult, inferredTypeResult, inferredTypeDurableResult, crazyWorkflowResult] =\n    await Promise.all([declaredTypeRun, inferredTypeRun, inferredTypeDurableRun, crazyWorkflowRun]);\n\n  console.log('declaredTypeResult', declaredTypeResult);\n  console.log('inferredTypeResult', inferredTypeResult);\n  console.log('inferredTypeDurableResult', inferredTypeDurableResult);\n  console.log('crazyWorkflowResult', crazyWorkflowResult);\n  console.log('declaredTypeResult.TransformedMessage', declaredTypeResult.TransformedMessage);\n  console.log('inferredTypeResult.TransformedMessage', inferredTypeResult.TransformedMessage);\n  console.log(\n    'inferredTypeDurableResult.TransformedMessage',\n    inferredTypeDurableResult.TransformedMessage\n  );\n  console.log('crazyWorkflowResult.TransformedMessage', crazyWorkflowResult.TransformedMessage);\n}\n\nif (require.main === module) {\n  main();\n}\n",
    "language": "ts",
    "source": "/Users/gabrielruttner/dev/hatchet/examples/typescript/inferred-typing/run.ts"
  },
  "L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3R5cGVzY3JpcHQvaW5mZXJyZWQtdHlwaW5nL3dvcmtlci50cw__": {
    "content": "import { hatchet } from '../hatchet-client';\nimport { declaredType, inferredType, inferredTypeDurable, crazyWorkflow } from './workflow';\n\nasync function main() {\n  const worker = await hatchet.worker('simple-worker', {\n    workflows: [declaredType, inferredType, inferredTypeDurable, crazyWorkflow],\n  });\n\n  await worker.start();\n}\n\nif (require.main === module) {\n  main();\n}\n",
    "language": "ts",
    "source": "/Users/gabrielruttner/dev/hatchet/examples/typescript/inferred-typing/worker.ts"
  },
  "L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3R5cGVzY3JpcHQvaW5mZXJyZWQtdHlwaW5nL3dvcmtmbG93LnRz": {
    "content": "import { hatchet } from '../hatchet-client';\n\ntype SimpleInput = {\n  Message: string;\n};\n\ntype SimpleOutput = {\n  TransformedMessage: string;\n};\n\nexport const declaredType = hatchet.task<SimpleInput, SimpleOutput>({\n  name: 'declared-type',\n  fn: (input) => {\n    return {\n      TransformedMessage: input.Message.toLowerCase(),\n    };\n  },\n});\n\nexport const inferredType = hatchet.task({\n  name: 'inferred-type',\n  fn: (input: SimpleInput) => {\n    return {\n      TransformedMessage: input.Message.toUpperCase(),\n    };\n  },\n});\n\nexport const inferredTypeDurable = hatchet.durableTask({\n  name: 'inferred-type-durable',\n  fn: async (input: SimpleInput, ctx) => {\n    // await ctx.sleepFor('5s');\n\n    return {\n      TransformedMessage: input.Message.toUpperCase(),\n    };\n  },\n});\n\nexport const crazyWorkflow = hatchet.workflow<any, any>({\n  name: 'crazy-workflow',\n});\n\nconst step1 = crazyWorkflow.task(declaredType);\n// crazyWorkflow.task(inferredTypeDurable);\n\ncrazyWorkflow.task({\n  parents: [step1],\n  ...inferredType.taskDef,\n});\n",
    "language": "ts",
    "source": "/Users/gabrielruttner/dev/hatchet/examples/typescript/inferred-typing/workflow.ts"
  },
  "L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3R5cGVzY3JpcHQvbGVnYWN5L3J1bi50cw__": {
    "content": "import { hatchet } from '../hatchet-client';\nimport { simple } from './workflow';\n\nasync function main() {\n  const res = await hatchet.run<{ Message: string }, { step2: string }>(simple, {\n    Message: 'hello',\n  });\n\n  // eslint-disable-next-line no-console\n  console.log(res.step2);\n}\n\nif (require.main === module) {\n  main();\n}\n",
    "language": "ts",
    "source": "/Users/gabrielruttner/dev/hatchet/examples/typescript/legacy/run.ts"
  },
  "L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3R5cGVzY3JpcHQvbGVnYWN5L3dvcmtlci50cw__": {
    "content": "import { hatchet } from '../hatchet-client';\nimport { simple } from './workflow';\n\nasync function main() {\n  const worker = await hatchet.worker('legacy-worker', {\n    workflows: [simple],\n  });\n\n  await worker.start();\n}\n\nif (require.main === module) {\n  main();\n}\n",
    "language": "ts",
    "source": "/Users/gabrielruttner/dev/hatchet/examples/typescript/legacy/worker.ts"
  },
  "L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3R5cGVzY3JpcHQvbGVnYWN5L3dvcmtmbG93LnRz": {
    "content": "import { Workflow } from '@hatchet/workflow';\n\nexport const simple: Workflow = {\n  id: 'legacy-workflow',\n  description: 'test',\n  on: {\n    event: 'user:create',\n  },\n  steps: [\n    {\n      name: 'step1',\n      run: async (ctx) => {\n        const input = ctx.workflowInput();\n\n        return { step1: `original input: ${input.Message}` };\n      },\n    },\n    {\n      name: 'step2',\n      parents: ['step1'],\n      run: (ctx) => {\n        const step1Output = ctx.stepOutput('step1');\n\n        return { step2: `step1 output: ${step1Output.step1}` };\n      },\n    },\n  ],\n};\n",
    "language": "ts",
    "source": "/Users/gabrielruttner/dev/hatchet/examples/typescript/legacy/workflow.ts"
  },
  "L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3R5cGVzY3JpcHQvbXVsdGlwbGVfd2ZfY29uY3VycmVuY3kvcnVuLnRz": {
    "content": "import { multiConcurrency } from './workflow';\n\nasync function main() {\n  const res = await multiConcurrency.run([\n    {\n      Message: 'Hello World',\n      GroupKey: 'A',\n    },\n    {\n      Message: 'Goodbye Moon',\n      GroupKey: 'A',\n    },\n    {\n      Message: 'Hello World B',\n      GroupKey: 'B',\n    },\n  ]);\n\n  // eslint-disable-next-line no-console\n  console.log(res[0]['to-lower'].TransformedMessage);\n  // eslint-disable-next-line no-console\n  console.log(res[1]['to-lower'].TransformedMessage);\n  // eslint-disable-next-line no-console\n  console.log(res[2]['to-lower'].TransformedMessage);\n}\n\nif (require.main === module) {\n  main().then(() => process.exit(0));\n}\n",
    "language": "ts",
    "source": "/Users/gabrielruttner/dev/hatchet/examples/typescript/multiple_wf_concurrency/run.ts"
  },
  "L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3R5cGVzY3JpcHQvbXVsdGlwbGVfd2ZfY29uY3VycmVuY3kvd29ya2VyLnRz": {
    "content": "import { hatchet } from '../hatchet-client';\nimport { multiConcurrency } from './workflow';\n\nasync function main() {\n  const worker = await hatchet.worker('simple-concurrency-worker', {\n    workflows: [multiConcurrency],\n  });\n\n  await worker.start();\n}\n\nif (require.main === module) {\n  main();\n}\n",
    "language": "ts",
    "source": "/Users/gabrielruttner/dev/hatchet/examples/typescript/multiple_wf_concurrency/worker.ts"
  },
  "L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3R5cGVzY3JpcHQvbXVsdGlwbGVfd2ZfY29uY3VycmVuY3kvd29ya2Zsb3cudHM_": {
    "content": "import { ConcurrencyLimitStrategy } from '@hatchet/workflow';\nimport { hatchet } from '../hatchet-client';\n\ntype SimpleInput = {\n  Message: string;\n  GroupKey: string;\n};\n\ntype SimpleOutput = {\n  'to-lower': {\n    TransformedMessage: string;\n  };\n};\n\nconst sleep = (ms: number) =>\n  new Promise((resolve) => {\n    setTimeout(resolve, ms);\n  });\n\n// ❓ Concurrency Strategy With Key\nexport const multiConcurrency = hatchet.workflow<SimpleInput, SimpleOutput>({\n  name: 'simple-concurrency',\n  concurrency: [\n    {\n      maxRuns: 1,\n      limitStrategy: ConcurrencyLimitStrategy.GROUP_ROUND_ROBIN,\n      expression: 'input.GroupKey',\n    },\n    {\n      maxRuns: 1,\n      limitStrategy: ConcurrencyLimitStrategy.GROUP_ROUND_ROBIN,\n      expression: 'input.UserId',\n    },\n  ],\n});\n// !!\n\nmultiConcurrency.task({\n  name: 'to-lower',\n  fn: async (input) => {\n    await sleep(Math.floor(Math.random() * (1000 - 200 + 1)) + 200);\n    return {\n      TransformedMessage: input.Message.toLowerCase(),\n    };\n  },\n});\n",
    "language": "ts",
    "source": "/Users/gabrielruttner/dev/hatchet/examples/typescript/multiple_wf_concurrency/workflow.ts"
  },
  "L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3R5cGVzY3JpcHQvbm9uX3JldHJ5YWJsZS9ydW4udHM_": {
    "content": "import { nonRetryableWorkflow } from './workflow';\n\nasync function main() {\n  const res = await nonRetryableWorkflow.runNoWait({});\n\n  // eslint-disable-next-line no-console\n  console.log(res);\n}\n\nif (require.main === module) {\n  main();\n}\n",
    "language": "ts",
    "source": "/Users/gabrielruttner/dev/hatchet/examples/typescript/non_retryable/run.ts"
  },
  "L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3R5cGVzY3JpcHQvbm9uX3JldHJ5YWJsZS93b3JrZXIudHM_": {
    "content": "import { hatchet } from '../hatchet-client';\nimport { nonRetryableWorkflow } from './workflow';\n\nasync function main() {\n  const worker = await hatchet.worker('no-retry-worker', {\n    workflows: [nonRetryableWorkflow],\n  });\n\n  await worker.start();\n}\n\nif (require.main === module) {\n  main();\n}\n",
    "language": "ts",
    "source": "/Users/gabrielruttner/dev/hatchet/examples/typescript/non_retryable/worker.ts"
  },
  "L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3R5cGVzY3JpcHQvbm9uX3JldHJ5YWJsZS93b3JrZmxvdy50cw__": {
    "content": "import { NonRetryableError } from '@hatchet-dev/typescript-sdk/task';\nimport { hatchet } from '../hatchet-client';\n\nexport const nonRetryableWorkflow = hatchet.workflow({\n  name: 'no-retry-workflow',\n});\n\n// ❓ Non-retrying task\nconst shouldNotRetry = nonRetryableWorkflow.task({\n  name: 'should-not-retry',\n  fn: () => {\n    throw new NonRetryableError('This task should not retry');\n  },\n  retries: 1,\n});\n// !!\n\n// Create a task that should retry\nconst shouldRetryWrongErrorType = nonRetryableWorkflow.task({\n  name: 'should-retry-wrong-error-type',\n  fn: () => {\n    throw new Error('This task should not retry');\n  },\n  retries: 1,\n});\n\nconst shouldNotRetrySuccessfulTask = nonRetryableWorkflow.task({\n  name: 'should-not-retry-successful-task',\n  fn: () => {},\n});\n",
    "language": "ts",
    "source": "/Users/gabrielruttner/dev/hatchet/examples/typescript/non_retryable/workflow.ts"
  },
  "L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3R5cGVzY3JpcHQvb25fY3Jvbi93b3JrZXIudHM_": {
    "content": "import { hatchet } from '../hatchet-client';\nimport { onCron } from './workflow';\n\nasync function main() {\n  const worker = await hatchet.worker('on-cron-worker', {\n    workflows: [onCron],\n  });\n\n  await worker.start();\n}\n\nif (require.main === module) {\n  main();\n}\n",
    "language": "ts",
    "source": "/Users/gabrielruttner/dev/hatchet/examples/typescript/on_cron/worker.ts"
  },
  "L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3R5cGVzY3JpcHQvb25fY3Jvbi93b3JrZmxvdy50cw__": {
    "content": "import { hatchet } from '../hatchet-client';\n\nexport type Input = {\n  Message: string;\n};\n\ntype OnCronOutput = {\n  job: {\n    TransformedMessage: string;\n  };\n};\n\n// ❓ Run Workflow on Cron\nexport const onCron = hatchet.workflow<Input, OnCronOutput>({\n  name: 'on-cron-workflow',\n  on: {\n    // 👀 add a cron expression to run the workflow every 15 minutes\n    cron: '*/15 * * * *',\n  },\n});\n// !!\n\nonCron.task({\n  name: 'job',\n  fn: (input) => {\n    return {\n      TransformedMessage: input.Message.toLowerCase(),\n    };\n  },\n});\n",
    "language": "ts",
    "source": "/Users/gabrielruttner/dev/hatchet/examples/typescript/on_cron/workflow.ts"
  },
  "L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3R5cGVzY3JpcHQvb25fZXZlbnQvd29ya2VyLnRz": {
    "content": "import { hatchet } from '../hatchet-client';\nimport { lower, upper } from './workflow';\n\nasync function main() {\n  const worker = await hatchet.worker('on-event-worker', {\n    workflows: [lower, upper],\n  });\n\n  await worker.start();\n}\n\nif (require.main === module) {\n  main();\n}\n",
    "language": "ts",
    "source": "/Users/gabrielruttner/dev/hatchet/examples/typescript/on_event/worker.ts"
  },
  "L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3R5cGVzY3JpcHQvb25fZXZlbnQvd29ya2Zsb3cudHM_": {
    "content": "import { hatchet } from '../hatchet-client';\n\nexport type Input = {\n  Message: string;\n};\n\nexport const SIMPLE_EVENT = 'simple-event:create';\n\ntype LowerOutput = {\n  lower: {\n    TransformedMessage: string;\n  };\n};\n\n// ❓ Run workflow on event\nexport const lower = hatchet.workflow<Input, LowerOutput>({\n  name: 'lower',\n  // 👀 Declare the event that will trigger the workflow\n  onEvents: ['simple-event:create'],\n});\n// !!\n\nlower.task({\n  name: 'lower',\n  fn: (input) => {\n    return {\n      TransformedMessage: input.Message.toLowerCase(),\n    };\n  },\n});\n\ntype UpperOutput = {\n  upper: {\n    TransformedMessage: string;\n  };\n};\n\nexport const upper = hatchet.workflow<Input, UpperOutput>({\n  name: 'upper',\n  on: {\n    event: SIMPLE_EVENT,\n  },\n});\n\nupper.task({\n  name: 'upper',\n  fn: (input) => {\n    return {\n      TransformedMessage: input.Message.toUpperCase(),\n    };\n  },\n});\n",
    "language": "ts",
    "source": "/Users/gabrielruttner/dev/hatchet/examples/typescript/on_event/workflow.ts"
  },
  "L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3R5cGVzY3JpcHQvb25fZXZlbnQgY29weS93b3JrZXIudHM_": {
    "content": "import { hatchet } from '../hatchet-client';\nimport { lower, upper } from './workflow';\n\nasync function main() {\n  const worker = await hatchet.worker('on-event-worker', {\n    workflows: [lower, upper],\n  });\n\n  await worker.start();\n}\n\nif (require.main === module) {\n  main();\n}\n",
    "language": "ts",
    "source": "/Users/gabrielruttner/dev/hatchet/examples/typescript/on_event copy/worker.ts"
  },
  "L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3R5cGVzY3JpcHQvb25fZXZlbnQgY29weS93b3JrZmxvdy50cw__": {
    "content": "import { hatchet } from '../hatchet-client';\n\nexport type Input = {\n  Message: string;\n};\n\ntype LowerOutput = {\n  lower: {\n    TransformedMessage: string;\n  };\n};\n\n// ❓ Run workflow on event\nexport const lower = hatchet.workflow<Input, LowerOutput>({\n  name: 'lower',\n  on: {\n    // 👀 Declare the event that will trigger the workflow\n    event: 'simple-event:create',\n  },\n});\n// !!\n\nlower.task({\n  name: 'lower',\n  fn: (input) => {\n    return {\n      TransformedMessage: input.Message.toLowerCase(),\n    };\n  },\n});\n\ntype UpperOutput = {\n  upper: {\n    TransformedMessage: string;\n  };\n};\n\nexport const upper = hatchet.workflow<Input, UpperOutput>({\n  name: 'upper',\n  on: {\n    event: 'simple-event:create',\n  },\n});\n\nupper.task({\n  name: 'upper',\n  fn: (input) => {\n    return {\n      TransformedMessage: input.Message.toUpperCase(),\n    };\n  },\n});\n",
    "language": "ts",
    "source": "/Users/gabrielruttner/dev/hatchet/examples/typescript/on_event copy/workflow.ts"
  },
  "L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3R5cGVzY3JpcHQvb25fZmFpbHVyZS9ydW4udHM_": {
    "content": "/* eslint-disable no-console */\nimport { failureWorkflow } from './workflow';\n\nasync function main() {\n  try {\n    const res = await failureWorkflow.run({});\n    console.log(res);\n  } catch (e) {\n    console.log('error', e);\n  }\n}\n\nif (require.main === module) {\n  main()\n    .catch(console.error)\n    .finally(() => process.exit(0));\n}\n",
    "language": "ts",
    "source": "/Users/gabrielruttner/dev/hatchet/examples/typescript/on_failure/run.ts"
  },
  "L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3R5cGVzY3JpcHQvb25fZmFpbHVyZS93b3JrZXIudHM_": {
    "content": "import { hatchet } from '../hatchet-client';\nimport { failureWorkflow } from './workflow';\n\nasync function main() {\n  const worker = await hatchet.worker('always-fail-worker', {\n    workflows: [failureWorkflow],\n  });\n\n  await worker.start();\n}\n\nif (require.main === module) {\n  main();\n}\n",
    "language": "ts",
    "source": "/Users/gabrielruttner/dev/hatchet/examples/typescript/on_failure/worker.ts"
  },
  "L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3R5cGVzY3JpcHQvb25fZmFpbHVyZS93b3JrZmxvdy50cw__": {
    "content": "/* eslint-disable no-console */\nimport { hatchet } from '../hatchet-client';\n\n// ❓ On Failure Task\nexport const failureWorkflow = hatchet.workflow({\n  name: 'always-fail',\n});\n\nfailureWorkflow.task({\n  name: 'always-fail',\n  fn: async () => {\n    throw new Error('intentional failure');\n  },\n});\n\nfailureWorkflow.onFailure({\n  name: 'on-failure',\n  fn: async (input, ctx) => {\n    console.log('onFailure for run:', ctx.workflowRunId());\n    return {\n      'on-failure': 'success',\n    };\n  },\n});\n// !!\n",
    "language": "ts",
    "source": "/Users/gabrielruttner/dev/hatchet/examples/typescript/on_failure/workflow.ts"
  },
  "L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3R5cGVzY3JpcHQvb25fc3VjY2Vzcy9ydW4udHM_": {
    "content": "/* eslint-disable no-console */\nimport { onSuccessDag } from './workflow';\n\nasync function main() {\n  try {\n    const res2 = await onSuccessDag.run({});\n    console.log(res2);\n  } catch (e) {\n    console.log('error', e);\n  }\n}\n\nif (require.main === module) {\n  main()\n    .catch(console.error)\n    .finally(() => process.exit(0));\n}\n",
    "language": "ts",
    "source": "/Users/gabrielruttner/dev/hatchet/examples/typescript/on_success/run.ts"
  },
  "L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3R5cGVzY3JpcHQvb25fc3VjY2Vzcy93b3JrZXIudHM_": {
    "content": "import { hatchet } from '../hatchet-client';\nimport { onSuccessDag } from './workflow';\n\nasync function main() {\n  const worker = await hatchet.worker('always-succeed-worker', {\n    workflows: [onSuccessDag],\n  });\n\n  await worker.start();\n}\n\nif (require.main === module) {\n  main();\n}\n",
    "language": "ts",
    "source": "/Users/gabrielruttner/dev/hatchet/examples/typescript/on_success/worker.ts"
  },
  "L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3R5cGVzY3JpcHQvb25fc3VjY2Vzcy93b3JrZmxvdy50cw__": {
    "content": "/* eslint-disable no-console */\nimport { hatchet } from '../hatchet-client';\n\n// ❓ On Success DAG\nexport const onSuccessDag = hatchet.workflow({\n  name: 'on-success-dag',\n});\n\nonSuccessDag.task({\n  name: 'always-succeed',\n  fn: async () => {\n    return {\n      'always-succeed': 'success',\n    };\n  },\n});\nonSuccessDag.task({\n  name: 'always-succeed2',\n  fn: async () => {\n    return {\n      'always-succeed': 'success',\n    };\n  },\n});\n\n// 👀 onSuccess handler will run if all tasks in the workflow succeed\nonSuccessDag.onSuccess({\n  fn: (_, ctx) => {\n    console.log('onSuccess for run:', ctx.workflowRunId());\n    return {\n      'on-success': 'success',\n    };\n  },\n});\n// !!\n",
    "language": "ts",
    "source": "/Users/gabrielruttner/dev/hatchet/examples/typescript/on_success/workflow.ts"
  },
  "L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3R5cGVzY3JpcHQvcHJpb3JpdHkvcnVuLnRz": {
    "content": "import { Priority } from '@hatchet-dev/typescript-sdk';\nimport { priority } from './workflow';\n\n/* eslint-disable no-console */\nasync function main() {\n  try {\n    console.log('running priority workflow');\n\n    // ❓ Run a Task with a Priority\n    const run = priority.run(new Date(Date.now() + 60 * 60 * 1000), { priority: Priority.HIGH });\n    // !!\n\n    // ❓ Schedule and cron\n    const scheduled = priority.schedule(\n      new Date(Date.now() + 60 * 60 * 1000),\n      {},\n      { priority: Priority.HIGH }\n    );\n    const delayed = priority.delay(60 * 60 * 1000, {}, { priority: Priority.HIGH });\n    const cron = priority.cron(\n      `daily-cron-${Math.random()}`,\n      '0 0 * * *',\n      {},\n      { priority: Priority.HIGH }\n    );\n    // !!\n\n    const [scheduledResult, delayedResult] = await Promise.all([scheduled, delayed]);\n    console.log('scheduledResult', scheduledResult);\n    console.log('delayedResult', delayedResult);\n    // !!\n  } catch (e) {\n    console.log('error', e);\n  }\n}\n\nif (require.main === module) {\n  main()\n    .catch(console.error)\n    .finally(() => process.exit(0));\n}\n",
    "language": "ts",
    "source": "/Users/gabrielruttner/dev/hatchet/examples/typescript/priority/run.ts"
  },
  "L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3R5cGVzY3JpcHQvcHJpb3JpdHkvd29ya2VyLnRz": {
    "content": "import { hatchet } from '../hatchet-client';\nimport { priorityTasks } from './workflow';\n\nasync function main() {\n  const worker = await hatchet.worker('priority-worker', {\n    workflows: [...priorityTasks],\n  });\n\n  await worker.start();\n}\n\nif (require.main === module) {\n  main();\n}\n",
    "language": "ts",
    "source": "/Users/gabrielruttner/dev/hatchet/examples/typescript/priority/worker.ts"
  },
  "L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3R5cGVzY3JpcHQvcHJpb3JpdHkvd29ya2Zsb3cudHM_": {
    "content": "/* eslint-disable no-console */\nimport { Priority } from '@hatchet-dev/typescript-sdk';\nimport { hatchet } from '../hatchet-client';\n\n// ❓ Simple Task Priority\nexport const priority = hatchet.task({\n  name: 'priority',\n  defaultPriority: Priority.MEDIUM,\n  fn: async (_, ctx) => {\n    return {\n      priority: ctx.priority(),\n    };\n  },\n});\n// !!\n\n// ❓ Task Priority in a Workflow\nexport const priorityWf = hatchet.workflow({\n  name: 'priorityWf',\n  defaultPriority: Priority.LOW,\n});\n// !!\n\npriorityWf.task({\n  name: 'child-medium',\n  fn: async (_, ctx) => {\n    return {\n      priority: ctx.priority(),\n    };\n  },\n});\n\npriorityWf.task({\n  name: 'child-high',\n  // will inherit the default priority from the workflow\n  fn: async (_, ctx) => {\n    return {\n      priority: ctx.priority(),\n    };\n  },\n});\n\nexport const priorityTasks = [priority, priorityWf];\n",
    "language": "ts",
    "source": "/Users/gabrielruttner/dev/hatchet/examples/typescript/priority/workflow.ts"
  },
  "L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3R5cGVzY3JpcHQvcmF0ZV9saW1pdC93b3JrZmxvdy50cw__": {
    "content": "import { RateLimitDuration } from '@hatchet/protoc/v1/workflows';\nimport { hatchet } from '../hatchet-client';\n\n// ❓ Upsert Rate Limit\nhatchet.ratelimits.upsert({\n  key: 'api-service-rate-limit',\n  limit: 10,\n  duration: RateLimitDuration.SECOND,\n});\n// !!\n\n// ❓ Static\nconst RATE_LIMIT_KEY = 'api-service-rate-limit';\n\nconst task1 = hatchet.task({\n  name: 'task1',\n  rateLimits: [\n    {\n      staticKey: RATE_LIMIT_KEY,\n      units: 1,\n    },\n  ],\n  fn: (input) => {\n    console.log('executed task1');\n  },\n});\n\n// !!\n\n// ❓ Dynamic\nconst task2 = hatchet.task({\n  name: 'task2',\n  fn: (input: { userId: string }) => {\n    console.log('executed task2 for user: ', input.userId);\n  },\n  rateLimits: [\n    {\n      dynamicKey: 'input.userId',\n      units: 1,\n      limit: 10,\n      duration: RateLimitDuration.MINUTE,\n    },\n  ],\n});\n// !!\n",
    "language": "ts",
    "source": "/Users/gabrielruttner/dev/hatchet/examples/typescript/rate_limit/workflow.ts"
  },
  "L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3R5cGVzY3JpcHQvcmV0cmllcy9ydW4udHM_": {
    "content": "/* eslint-disable no-console */\nimport { retries } from './workflow';\n\nasync function main() {\n  try {\n    const res = await retries.run({});\n    console.log(res);\n  } catch (e) {\n    console.log('error', e);\n  }\n}\n\nif (require.main === module) {\n  main()\n    .catch(console.error)\n    .finally(() => process.exit(0));\n}\n",
    "language": "ts",
    "source": "/Users/gabrielruttner/dev/hatchet/examples/typescript/retries/run.ts"
  },
  "L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3R5cGVzY3JpcHQvcmV0cmllcy93b3JrZXIudHM_": {
    "content": "import { hatchet } from '../hatchet-client';\nimport { retries } from './workflow';\n\nasync function main() {\n  const worker = await hatchet.worker('always-fail-worker', {\n    workflows: [retries],\n  });\n\n  await worker.start();\n}\n\nif (require.main === module) {\n  main();\n}\n",
    "language": "ts",
    "source": "/Users/gabrielruttner/dev/hatchet/examples/typescript/retries/worker.ts"
  },
  "L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3R5cGVzY3JpcHQvcmV0cmllcy93b3JrZmxvdy50cw__": {
    "content": "/* eslint-disable no-console */\nimport { hatchet } from '../hatchet-client';\n\n// ❓ Simple Step Retries\nexport const retries = hatchet.task({\n  name: 'retries',\n  retries: 3,\n  fn: async (_, ctx) => {\n    throw new Error('intentional failure');\n  },\n});\n// !!\n\n// ❓ Retries with Count\nexport const retriesWithCount = hatchet.task({\n  name: 'retriesWithCount',\n  retries: 3,\n  fn: async (_, ctx) => {\n    // ❓ Get the current retry count\n    const retryCount = ctx.retryCount();\n\n    console.log(`Retry count: ${retryCount}`);\n\n    if (retryCount < 2) {\n      throw new Error('intentional failure');\n    }\n\n    return {\n      message: 'success',\n    };\n  },\n});\n// !!\n\n// ❓ Retries with Backoff\nexport const withBackoff = hatchet.task({\n  name: 'withBackoff',\n  retries: 10,\n  backoff: {\n    // 👀 Maximum number of seconds to wait between retries\n    maxSeconds: 10,\n    // 👀 Factor to increase the wait time between retries.\n    // This sequence will be 2s, 4s, 8s, 10s, 10s, 10s... due to the maxSeconds limit\n    factor: 2,\n  },\n  fn: async () => {\n    throw new Error('intentional failure');\n  },\n});\n// !!\n",
    "language": "ts",
    "source": "/Users/gabrielruttner/dev/hatchet/examples/typescript/retries/workflow.ts"
  },
  "L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3R5cGVzY3JpcHQvc2ltcGxlL3J1bi50cw__": {
    "content": "/* eslint-disable no-console */\nimport { hatchet } from '../hatchet-client';\nimport { simple } from './workflow';\n\nasync function main() {\n  // ❓ Running a Task\n  const res = await simple.run({\n    Message: 'HeLlO WoRlD',\n  });\n\n  // 👀 Access the results of the Task\n  console.log(res.TransformedMessage);\n  // !!\n}\n\nexport async function extra() {\n  // ❓ Running Multiple Tasks\n  const res1 = simple.run({\n    Message: 'HeLlO WoRlD',\n  });\n\n  const res2 = simple.run({\n    Message: 'Hello MoOn',\n  });\n\n  const results = await Promise.all([res1, res2]);\n\n  console.log(results[0].TransformedMessage);\n  console.log(results[1].TransformedMessage);\n  // !!\n\n  // ❓ Spawning Tasks from within a Task\n  const parent = hatchet.task({\n    name: 'parent',\n    fn: async (input, ctx) => {\n      // Simply call ctx.runChild with the task you want to run\n      const child = await ctx.runChild(simple, {\n        Message: 'HeLlO WoRlD',\n      });\n\n      return {\n        result: child.TransformedMessage,\n      };\n    },\n  });\n  // !!\n}\n\nif (require.main === module) {\n  main();\n}\n",
    "language": "ts",
    "source": "/Users/gabrielruttner/dev/hatchet/examples/typescript/simple/run.ts"
  },
  "L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3R5cGVzY3JpcHQvc2ltcGxlL3dvcmtlci50cw__": {
    "content": "// ❓ Declaring a Worker\nimport { hatchet } from '../hatchet-client';\nimport { simple } from './workflow';\nimport { parent, child } from './workflow-with-child';\n\nasync function main() {\n  const worker = await hatchet.worker('simple-worker', {\n    // 👀 Declare the workflows that the worker can execute\n    workflows: [simple, parent, child],\n    // 👀 Declare the number of concurrent task runs the worker can accept\n    slots: 100,\n  });\n\n  await worker.start();\n}\n\nif (require.main === module) {\n  main();\n}\n// !!\n",
    "language": "ts",
    "source": "/Users/gabrielruttner/dev/hatchet/examples/typescript/simple/worker.ts"
  },
  "L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3R5cGVzY3JpcHQvc2ltcGxlL3dvcmtmbG93LnRz": {
    "content": "// ❓ Declaring a Task\nimport { hatchet } from '../hatchet-client';\n\n// (optional) Define the input type for the workflow\nexport type SimpleInput = {\n  Message: string;\n};\n\nexport const simple = hatchet.task({\n  name: 'simple',\n  fn: (input: SimpleInput) => {\n    return {\n      TransformedMessage: input.Message.toLowerCase(),\n    };\n  },\n});\n\n// !!\n\n// see ./worker.ts and ./run.ts for how to run the workflow\n",
    "language": "ts",
    "source": "/Users/gabrielruttner/dev/hatchet/examples/typescript/simple/workflow.ts"
  },
  "L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3R5cGVzY3JpcHQvc3RpY2t5L3J1bi50cw__": {
    "content": "/* eslint-disable no-console */\nimport { retries } from '../retries/workflow';\n\nasync function main() {\n  try {\n    const res = await retries.run({});\n    console.log(res);\n  } catch (e) {\n    console.log('error', e);\n  }\n}\n\nif (require.main === module) {\n  main()\n    .catch(console.error)\n    .finally(() => process.exit(0));\n}\n",
    "language": "ts",
    "source": "/Users/gabrielruttner/dev/hatchet/examples/typescript/sticky/run.ts"
  },
  "L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3R5cGVzY3JpcHQvc3RpY2t5L3dvcmtlci50cw__": {
    "content": "import { hatchet } from '../hatchet-client';\nimport { retries } from '../retries/workflow';\n\nasync function main() {\n  const worker = await hatchet.worker('always-fail-worker', {\n    workflows: [retries],\n  });\n\n  await worker.start();\n}\n\nif (require.main === module) {\n  main();\n}\n",
    "language": "ts",
    "source": "/Users/gabrielruttner/dev/hatchet/examples/typescript/sticky/worker.ts"
  },
  "L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3R5cGVzY3JpcHQvc3RpY2t5L3dvcmtmbG93LnRz": {
    "content": "/* eslint-disable no-console */\nimport { StickyStrategy } from '@hatchet/protoc/workflows';\nimport { hatchet } from '../hatchet-client';\nimport { child } from '../child_workflows/workflow';\n\n// ❓ Sticky Task\nexport const sticky = hatchet.task({\n  name: 'sticky',\n  retries: 3,\n  sticky: StickyStrategy.SOFT,\n  fn: async (_, ctx) => {\n    // specify a child workflow to run on the same worker\n    const result = await ctx.runChild(\n      child,\n      {\n        N: 1,\n      },\n      { sticky: true }\n    );\n\n    return {\n      result,\n    };\n  },\n});\n// !!\n",
    "language": "ts",
    "source": "/Users/gabrielruttner/dev/hatchet/examples/typescript/sticky/workflow.ts"
  },
  "L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3R5cGVzY3JpcHQvdGltZW91dHMvcnVuLnRz": {
    "content": "/* eslint-disable no-console */\n// ❓ Running a Task with Results\nimport { cancellation } from './workflow';\n// ...\nasync function main() {\n  // 👀 Run the workflow with results\n  const res = await cancellation.run({});\n\n  // 👀 Access the results of the workflow\n  console.log(res.Completed);\n  // !!\n}\n\nif (require.main === module) {\n  main()\n    .catch(console.error)\n    .finally(() => process.exit(0));\n}\n",
    "language": "ts",
    "source": "/Users/gabrielruttner/dev/hatchet/examples/typescript/timeouts/run.ts"
  },
  "L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3R5cGVzY3JpcHQvdGltZW91dHMvd29ya2VyLnRz": {
    "content": "// ❓ Declaring a Worker\nimport { hatchet } from '../hatchet-client';\nimport { cancellation } from './workflow';\n\nasync function main() {\n  const worker = await hatchet.worker('cancellation-worker', {\n    // 👀 Declare the workflows that the worker can execute\n    workflows: [cancellation],\n    // 👀 Declare the number of concurrent task runs the worker can accept\n    slots: 100,\n  });\n\n  await worker.start();\n}\n\nif (require.main === module) {\n  main();\n}\n// !!\n",
    "language": "ts",
    "source": "/Users/gabrielruttner/dev/hatchet/examples/typescript/timeouts/worker.ts"
  },
  "L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3R5cGVzY3JpcHQvdGltZW91dHMvd29ya2Zsb3cudHM_": {
    "content": "// ❓ Declaring a Task\nimport sleep from '@hatchet/util/sleep';\nimport { hatchet } from '../hatchet-client';\n\n// (optional) Define the input type for the workflow\nexport const cancellation = hatchet.task({\n  name: 'cancellation',\n  executionTimeout: '3s',\n  fn: async (_, { cancelled }) => {\n    await sleep(10 * 1000);\n\n    if (cancelled) {\n      throw new Error('Task was cancelled');\n    }\n\n    return {\n      Completed: true,\n    };\n  },\n});\n// !!\n\n// see ./worker.ts and ./run.ts for how to run the workflow\n",
    "language": "ts",
    "source": "/Users/gabrielruttner/dev/hatchet/examples/typescript/timeouts/workflow.ts"
  },
  "L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3R5cGVzY3JpcHQvd2l0aF90aW1lb3V0cy93b3JrZmxvdy50cw__": {
    "content": "// ❓ Declaring a Task\nimport sleep from '@hatchet/util/sleep';\nimport { hatchet } from '../hatchet-client';\n\n// (optional) Define the input type for the workflow\nexport type SimpleInput = {\n  Message: string;\n};\n\n// ❓ Execution Timeout\nexport const withTimeouts = hatchet.task({\n  name: 'with-timeouts',\n  // time the task can wait in the queue before it is cancelled\n  scheduleTimeout: '10s',\n  // time the task can run before it is cancelled\n  executionTimeout: '10s',\n  fn: async (input: SimpleInput, ctx) => {\n    // wait 15 seconds\n    await sleep(15000);\n\n    // get the abort controller\n    const { controller } = ctx;\n\n    // if the abort controller is aborted, throw an error\n    if (controller.signal.aborted) {\n      throw new Error('cancelled');\n    }\n\n    return {\n      TransformedMessage: input.Message.toLowerCase(),\n    };\n  },\n});\n// !!\n\n// ❓ Refresh Timeout\nexport const refreshTimeout = hatchet.task({\n  name: 'refresh-timeout',\n  executionTimeout: '10s',\n  scheduleTimeout: '10s',\n  fn: async (input: SimpleInput, ctx) => {\n    // adds 15 seconds to the execution timeout\n    ctx.refreshTimeout('15s');\n    await sleep(15000);\n\n    // get the abort controller\n    const { controller } = ctx;\n\n    // now this condition will not be met\n    // if the abort controller is aborted, throw an error\n    if (controller.signal.aborted) {\n      throw new Error('cancelled');\n    }\n\n    return {\n      TransformedMessage: input.Message.toLowerCase(),\n    };\n  },\n});\n// !!\n",
    "language": "ts",
    "source": "/Users/gabrielruttner/dev/hatchet/examples/typescript/with_timeouts/workflow.ts"
  },
  "L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3B5dGhvbi9hZmZpbml0eV93b3JrZXJzL3dvcmtlci5weQ__": {
    "content": "from hatchet_sdk import Context, EmptyModel, Hatchet, WorkerLabelComparator\nfrom hatchet_sdk.labels import DesiredWorkerLabel\n\nhatchet = Hatchet(debug=True)\n\n\n# ❓ AffinityWorkflow\n\naffinity_worker_workflow = hatchet.workflow(name=\"AffinityWorkflow\")\n\n\n@affinity_worker_workflow.task(\n    desired_worker_labels={\n        \"model\": DesiredWorkerLabel(value=\"fancy-ai-model-v2\", weight=10),\n        \"memory\": DesiredWorkerLabel(\n            value=256,\n            required=True,\n            comparator=WorkerLabelComparator.LESS_THAN,\n        ),\n    },\n)\n\n# ‼️\n\n\n# ❓ AffinityTask\nasync def step(input: EmptyModel, ctx: Context) -> dict[str, str | None]:\n    if ctx.worker.labels().get(\"model\") != \"fancy-ai-model-v2\":\n        ctx.worker.upsert_labels({\"model\": \"unset\"})\n        # DO WORK TO EVICT OLD MODEL / LOAD NEW MODEL\n        ctx.worker.upsert_labels({\"model\": \"fancy-ai-model-v2\"})\n\n    return {\"worker\": ctx.worker.id()}\n\n\n# ‼️\n\n\ndef main() -> None:\n\n    # ❓ AffinityWorker\n    worker = hatchet.worker(\n        \"affinity-worker\",\n        slots=10,\n        labels={\n            \"model\": \"fancy-ai-model-v2\",\n            \"memory\": 512,\n        },\n        workflows=[affinity_worker_workflow],\n    )\n    worker.start()\n\n\n# ‼️\n\nif __name__ == \"__main__\":\n    main()\n",
    "language": "py",
    "source": "/Users/gabrielruttner/dev/hatchet/examples/python/affinity_workers/worker.py"
  },
  "L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3B5dGhvbi9ibG9ja2VkX2FzeW5jL3dvcmtlci5weQ__": {
    "content": "import hashlib\nimport time\nfrom datetime import timedelta\n\nfrom hatchet_sdk import Context, EmptyModel, Hatchet\n\nhatchet = Hatchet(debug=True)\n\n# WARNING: this is an example of what NOT to do\n# This workflow is intentionally blocking the main thread\n# and will block the worker from processing other workflows\n#\n# You do not want to run long sync functions in an async def function\n\nblocked_worker_workflow = hatchet.workflow(name=\"Blocked\")\n\n\n@blocked_worker_workflow.task(execution_timeout=timedelta(seconds=11), retries=3)\nasync def step1(input: EmptyModel, ctx: Context) -> dict[str, str | int | float]:\n    print(\"Executing step1\")\n\n    # CPU-bound task: Calculate a large number of SHA-256 hashes\n    start_time = time.time()\n    iterations = 10_000_000\n    for i in range(iterations):\n        hashlib.sha256(f\"data{i}\".encode()).hexdigest()\n\n    end_time = time.time()\n    execution_time = end_time - start_time\n\n    print(f\"Completed {iterations} hash calculations in {execution_time:.2f} seconds\")\n\n    return {\n        \"step1\": \"step1\",\n        \"iterations\": iterations,\n        \"execution_time\": execution_time,\n    }\n\n\ndef main() -> None:\n    worker = hatchet.worker(\n        \"blocked-worker\", slots=3, workflows=[blocked_worker_workflow]\n    )\n    worker.start()\n\n\nif __name__ == \"__main__\":\n    main()\n",
    "language": "py",
    "source": "/Users/gabrielruttner/dev/hatchet/examples/python/blocked_async/worker.py"
  },
  "L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3B5dGhvbi9idWxrX2Zhbm91dC93b3JrZXIucHk_": {
    "content": "from datetime import timedelta\nfrom typing import Any\n\nfrom pydantic import BaseModel\n\nfrom hatchet_sdk import Context, Hatchet\nfrom hatchet_sdk.clients.admin import TriggerWorkflowOptions\n\nhatchet = Hatchet(debug=True)\n\n\nclass ParentInput(BaseModel):\n    n: int = 100\n\n\nclass ChildInput(BaseModel):\n    a: str\n\n\nbulk_parent_wf = hatchet.workflow(name=\"BulkFanoutParent\", input_validator=ParentInput)\nbulk_child_wf = hatchet.workflow(name=\"BulkFanoutChild\", input_validator=ChildInput)\n\n\n# ❓ BulkFanoutParent\n@bulk_parent_wf.task(execution_timeout=timedelta(minutes=5))\nasync def spawn(input: ParentInput, ctx: Context) -> dict[str, list[dict[str, Any]]]:\n    # 👀 Create each workflow run to spawn\n    child_workflow_runs = [\n        bulk_child_wf.create_bulk_run_item(\n            input=ChildInput(a=str(i)),\n            key=f\"child{i}\",\n            options=TriggerWorkflowOptions(additional_metadata={\"hello\": \"earth\"}),\n        )\n        for i in range(input.n)\n    ]\n\n    # 👀 Run workflows in bulk to improve performance\n    spawn_results = await bulk_child_wf.aio_run_many(child_workflow_runs)\n\n    return {\"results\": spawn_results}\n\n\n# ‼️\n\n\n@bulk_child_wf.task()\ndef process(input: ChildInput, ctx: Context) -> dict[str, str]:\n    print(f\"child process {input.a}\")\n    return {\"status\": \"success \" + input.a}\n\n\n@bulk_child_wf.task()\ndef process2(input: ChildInput, ctx: Context) -> dict[str, str]:\n    print(\"child process2\")\n    return {\"status2\": \"success\"}\n\n\ndef main() -> None:\n    worker = hatchet.worker(\n        \"fanout-worker\", slots=40, workflows=[bulk_parent_wf, bulk_child_wf]\n    )\n    worker.start()\n\n\nif __name__ == \"__main__\":\n    main()\n",
    "language": "py",
    "source": "/Users/gabrielruttner/dev/hatchet/examples/python/bulk_fanout/worker.py"
  },
  "L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3B5dGhvbi9jYW5jZWxsYXRpb24vd29ya2VyLnB5": {
    "content": "import asyncio\nimport time\n\nfrom hatchet_sdk import Context, EmptyModel, Hatchet\n\nhatchet = Hatchet(debug=True)\n\ncancellation_workflow = hatchet.workflow(name=\"CancelWorkflow\")\n\n\n# ❓ Self-cancelling task\n@cancellation_workflow.task()\nasync def self_cancel(input: EmptyModel, ctx: Context) -> dict[str, str]:\n    await asyncio.sleep(2)\n\n    ## Cancel the task\n    await ctx.aio_cancel()\n\n    await asyncio.sleep(10)\n\n    return {\"error\": \"Task should have been cancelled\"}\n\n\n# !!\n\n\n# ❓ Checking exit flag\n@cancellation_workflow.task()\ndef check_flag(input: EmptyModel, ctx: Context) -> dict[str, str]:\n    for i in range(3):\n        time.sleep(1)\n\n        # Note: Checking the status of the exit flag is mostly useful for cancelling\n        # sync tasks without needing to forcibly kill the thread they're running on.\n        if ctx.exit_flag:\n            print(\"Task has been cancelled\")\n            raise ValueError(\"Task has been cancelled\")\n\n    return {\"error\": \"Task should have been cancelled\"}\n\n\n# !!\n\n\ndef main() -> None:\n    worker = hatchet.worker(\"cancellation-worker\", workflows=[cancellation_workflow])\n    worker.start()\n\n\nif __name__ == \"__main__\":\n    main()\n",
    "language": "py",
    "source": "/Users/gabrielruttner/dev/hatchet/examples/python/cancellation/worker.py"
  },
  "L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3B5dGhvbi9jaGlsZC93b3JrZXIucHk_": {
    "content": "# ❓ Simple\n\nfrom pydantic import BaseModel\n\nfrom hatchet_sdk import Context, Hatchet\n\nhatchet = Hatchet(debug=True)\n\n\nclass SimpleInput(BaseModel):\n    message: str\n\n\nclass SimpleOutput(BaseModel):\n    transformed_message: str\n\n\nchild_task = hatchet.workflow(name=\"SimpleWorkflow\", input_validator=SimpleInput)\n\n\n@child_task.task(name=\"step1\")\ndef step1(input: SimpleInput, ctx: Context) -> SimpleOutput:\n    print(\"executed step1: \", input.message)\n    return SimpleOutput(transformed_message=input.message.upper())\n\n\n# ‼️\n\n\ndef main() -> None:\n    worker = hatchet.worker(\"test-worker\", slots=1, workflows=[child_task])\n    worker.start()\n\n\nif __name__ == \"__main__\":\n    main()\n",
    "language": "py",
    "source": "/Users/gabrielruttner/dev/hatchet/examples/python/child/worker.py"
  },
  "L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3B5dGhvbi9jb25jdXJyZW5jeV9saW1pdC93b3JrZXIucHk_": {
    "content": "import time\nfrom typing import Any\n\nfrom pydantic import BaseModel\n\nfrom hatchet_sdk import (\n    ConcurrencyExpression,\n    ConcurrencyLimitStrategy,\n    Context,\n    Hatchet,\n)\n\nhatchet = Hatchet(debug=True)\n\n\n# ❓ Workflow\nclass WorkflowInput(BaseModel):\n    run: int\n    group_key: str\n\n\nconcurrency_limit_workflow = hatchet.workflow(\n    name=\"ConcurrencyDemoWorkflow\",\n    concurrency=ConcurrencyExpression(\n        expression=\"input.group_key\",\n        max_runs=5,\n        limit_strategy=ConcurrencyLimitStrategy.CANCEL_IN_PROGRESS,\n    ),\n    input_validator=WorkflowInput,\n)\n\n# ‼️\n\n\n@concurrency_limit_workflow.task()\ndef step1(input: WorkflowInput, ctx: Context) -> dict[str, Any]:\n    time.sleep(3)\n    print(\"executed step1\")\n    return {\"run\": input.run}\n\n\ndef main() -> None:\n    worker = hatchet.worker(\n        \"concurrency-demo-worker\", slots=10, workflows=[concurrency_limit_workflow]\n    )\n\n    worker.start()\n\n\nif __name__ == \"__main__\":\n    main()\n",
    "language": "py",
    "source": "/Users/gabrielruttner/dev/hatchet/examples/python/concurrency_limit/worker.py"
  },
  "L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3B5dGhvbi9jb25jdXJyZW5jeV9saW1pdF9yci93b3JrZXIucHk_": {
    "content": "import time\n\nfrom pydantic import BaseModel\n\nfrom hatchet_sdk import (\n    ConcurrencyExpression,\n    ConcurrencyLimitStrategy,\n    Context,\n    Hatchet,\n)\n\nhatchet = Hatchet(debug=True)\n\n\n# ❓ Concurrency Strategy With Key\nclass WorkflowInput(BaseModel):\n    group: str\n\n\nconcurrency_limit_rr_workflow = hatchet.workflow(\n    name=\"ConcurrencyDemoWorkflowRR\",\n    concurrency=ConcurrencyExpression(\n        expression=\"input.group\",\n        max_runs=1,\n        limit_strategy=ConcurrencyLimitStrategy.GROUP_ROUND_ROBIN,\n    ),\n    input_validator=WorkflowInput,\n)\n# ‼️\n\n\n@concurrency_limit_rr_workflow.task()\ndef step1(input: WorkflowInput, ctx: Context) -> None:\n    print(\"starting step1\")\n    time.sleep(2)\n    print(\"finished step1\")\n    pass\n\n\ndef main() -> None:\n    worker = hatchet.worker(\n        \"concurrency-demo-worker-rr\",\n        slots=10,\n        workflows=[concurrency_limit_rr_workflow],\n    )\n\n    worker.start()\n\n\nif __name__ == \"__main__\":\n    main()\n",
    "language": "py",
    "source": "/Users/gabrielruttner/dev/hatchet/examples/python/concurrency_limit_rr/worker.py"
  },
  "L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3B5dGhvbi9jb25jdXJyZW5jeV9saW1pdF9ycl9sb2FkL3dvcmtlci5weQ__": {
    "content": "import random\nimport time\n\nfrom pydantic import BaseModel\n\nfrom hatchet_sdk import (\n    ConcurrencyExpression,\n    ConcurrencyLimitStrategy,\n    Context,\n    Hatchet,\n)\n\nhatchet = Hatchet(debug=True)\n\n\nclass LoadRRInput(BaseModel):\n    group: str\n\n\nload_rr_workflow = hatchet.workflow(\n    name=\"LoadRoundRobin\",\n    on_events=[\"concurrency-test\"],\n    concurrency=ConcurrencyExpression(\n        expression=\"input.group\",\n        max_runs=1,\n        limit_strategy=ConcurrencyLimitStrategy.GROUP_ROUND_ROBIN,\n    ),\n    input_validator=LoadRRInput,\n)\n\n\n@load_rr_workflow.on_failure_task()\ndef on_failure(input: LoadRRInput, context: Context) -> dict[str, str]:\n    print(\"on_failure\")\n    return {\"on_failure\": \"on_failure\"}\n\n\n@load_rr_workflow.task()\ndef step1(input: LoadRRInput, context: Context) -> dict[str, str]:\n    print(\"starting step1\")\n    time.sleep(random.randint(2, 20))\n    print(\"finished step1\")\n    return {\"step1\": \"step1\"}\n\n\n@load_rr_workflow.task(\n    retries=3,\n    backoff_factor=5,\n    backoff_max_seconds=60,\n)\ndef step2(sinput: LoadRRInput, context: Context) -> dict[str, str]:\n    print(\"starting step2\")\n    if random.random() < 0.5:  # 1% chance of failure\n        raise Exception(\"Random failure in step2\")\n    time.sleep(2)\n    print(\"finished step2\")\n    return {\"step2\": \"step2\"}\n\n\n@load_rr_workflow.task()\ndef step3(input: LoadRRInput, context: Context) -> dict[str, str]:\n    print(\"starting step3\")\n    time.sleep(0.2)\n    print(\"finished step3\")\n    return {\"step3\": \"step3\"}\n\n\ndef main() -> None:\n    worker = hatchet.worker(\n        \"concurrency-demo-worker-rr\", slots=50, workflows=[load_rr_workflow]\n    )\n\n    worker.start()\n\n\nif __name__ == \"__main__\":\n    main()\n",
    "language": "py",
    "source": "/Users/gabrielruttner/dev/hatchet/examples/python/concurrency_limit_rr_load/worker.py"
  },
  "L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3B5dGhvbi9jb25jdXJyZW5jeV9tdWx0aXBsZV9rZXlzL3dvcmtlci5weQ__": {
    "content": "import asyncio\n\nfrom pydantic import BaseModel\n\nfrom hatchet_sdk import (\n    ConcurrencyExpression,\n    ConcurrencyLimitStrategy,\n    Context,\n    Hatchet,\n)\n\nhatchet = Hatchet(debug=True)\n\nSLEEP_TIME = 2\nDIGIT_MAX_RUNS = 8\nNAME_MAX_RUNS = 3\n\n\n# ❓ Concurrency Strategy With Key\nclass WorkflowInput(BaseModel):\n    name: str\n    digit: str\n\n\nconcurrency_multiple_keys_workflow = hatchet.workflow(\n    name=\"ConcurrencyWorkflowManyKeys\",\n    input_validator=WorkflowInput,\n)\n# ‼️\n\n\n@concurrency_multiple_keys_workflow.task(\n    concurrency=[\n        ConcurrencyExpression(\n            expression=\"input.digit\",\n            max_runs=DIGIT_MAX_RUNS,\n            limit_strategy=ConcurrencyLimitStrategy.GROUP_ROUND_ROBIN,\n        ),\n        ConcurrencyExpression(\n            expression=\"input.name\",\n            max_runs=NAME_MAX_RUNS,\n            limit_strategy=ConcurrencyLimitStrategy.GROUP_ROUND_ROBIN,\n        ),\n    ]\n)\nasync def concurrency_task(input: WorkflowInput, ctx: Context) -> None:\n    await asyncio.sleep(SLEEP_TIME)\n\n\ndef main() -> None:\n    worker = hatchet.worker(\n        \"concurrency-worker-multiple-keys\",\n        slots=10,\n        workflows=[concurrency_multiple_keys_workflow],\n    )\n\n    worker.start()\n\n\nif __name__ == \"__main__\":\n    main()\n",
    "language": "py",
    "source": "/Users/gabrielruttner/dev/hatchet/examples/python/concurrency_multiple_keys/worker.py"
  },
  "L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3B5dGhvbi9jb25jdXJyZW5jeV93b3JrZmxvd19sZXZlbC93b3JrZXIucHk_": {
    "content": "import asyncio\n\nfrom pydantic import BaseModel\n\nfrom hatchet_sdk import (\n    ConcurrencyExpression,\n    ConcurrencyLimitStrategy,\n    Context,\n    Hatchet,\n)\n\nhatchet = Hatchet(debug=True)\n\nSLEEP_TIME = 2\nDIGIT_MAX_RUNS = 8\nNAME_MAX_RUNS = 3\n\n\n# ❓ Multiple Concurrency Keys\nclass WorkflowInput(BaseModel):\n    name: str\n    digit: str\n\n\nconcurrency_workflow_level_workflow = hatchet.workflow(\n    name=\"ConcurrencyWorkflowManyKeys\",\n    input_validator=WorkflowInput,\n    concurrency=[\n        ConcurrencyExpression(\n            expression=\"input.digit\",\n            max_runs=DIGIT_MAX_RUNS,\n            limit_strategy=ConcurrencyLimitStrategy.GROUP_ROUND_ROBIN,\n        ),\n        ConcurrencyExpression(\n            expression=\"input.name\",\n            max_runs=NAME_MAX_RUNS,\n            limit_strategy=ConcurrencyLimitStrategy.GROUP_ROUND_ROBIN,\n        ),\n    ],\n)\n# ‼️\n\n\n@concurrency_workflow_level_workflow.task()\nasync def task_1(input: WorkflowInput, ctx: Context) -> None:\n    await asyncio.sleep(SLEEP_TIME)\n\n\n@concurrency_workflow_level_workflow.task()\nasync def task_2(input: WorkflowInput, ctx: Context) -> None:\n    await asyncio.sleep(SLEEP_TIME)\n\n\ndef main() -> None:\n    worker = hatchet.worker(\n        \"concurrency-worker-workflow-level\",\n        slots=10,\n        workflows=[concurrency_workflow_level_workflow],\n    )\n\n    worker.start()\n\n\nif __name__ == \"__main__\":\n    main()\n",
    "language": "py",
    "source": "/Users/gabrielruttner/dev/hatchet/examples/python/concurrency_workflow_level/worker.py"
  },
  "L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3B5dGhvbi9kYWcvd29ya2VyLnB5": {
    "content": "import random\nimport time\nfrom datetime import timedelta\n\nfrom pydantic import BaseModel\n\nfrom hatchet_sdk import Context, EmptyModel, Hatchet\n\n\nclass StepOutput(BaseModel):\n    random_number: int\n\n\nclass RandomSum(BaseModel):\n    sum: int\n\n\nhatchet = Hatchet(debug=True)\n\ndag_workflow = hatchet.workflow(name=\"DAGWorkflow\")\n\n\n@dag_workflow.task(execution_timeout=timedelta(seconds=5))\ndef step1(input: EmptyModel, ctx: Context) -> StepOutput:\n    return StepOutput(random_number=random.randint(1, 100))\n\n\n@dag_workflow.task(execution_timeout=timedelta(seconds=5))\nasync def step2(input: EmptyModel, ctx: Context) -> StepOutput:\n    return StepOutput(random_number=random.randint(1, 100))\n\n\n@dag_workflow.task(parents=[step1, step2])\nasync def step3(input: EmptyModel, ctx: Context) -> RandomSum:\n    one = ctx.task_output(step1).random_number\n    two = (await ctx.task_output(step2)).random_number\n\n    return RandomSum(sum=one + two)\n\n\n@dag_workflow.task(parents=[step1, step3])\nasync def step4(input: EmptyModel, ctx: Context) -> dict[str, str]:\n    print(\n        \"executed step4\",\n        time.strftime(\"%H:%M:%S\", time.localtime()),\n        input,\n        ctx.task_output(step1),\n        await ctx.task_output(step3),\n    )\n    return {\n        \"step4\": \"step4\",\n    }\n\n\ndef main() -> None:\n    worker = hatchet.worker(\"dag-worker\", workflows=[dag_workflow])\n\n    worker.start()\n\n\nif __name__ == \"__main__\":\n    main()\n",
    "language": "py",
    "source": "/Users/gabrielruttner/dev/hatchet/examples/python/dag/worker.py"
  },
  "L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3B5dGhvbi9kZWR1cGUvd29ya2VyLnB5": {
    "content": "import asyncio\nfrom datetime import timedelta\nfrom typing import Any\n\nfrom hatchet_sdk import Context, EmptyModel, Hatchet, TriggerWorkflowOptions\nfrom hatchet_sdk.clients.admin import DedupeViolationErr\n\nhatchet = Hatchet(debug=True)\n\ndedupe_parent_wf = hatchet.workflow(name=\"DedupeParent\")\ndedupe_child_wf = hatchet.workflow(name=\"DedupeChild\")\n\n\n@dedupe_parent_wf.task(execution_timeout=timedelta(minutes=1))\nasync def spawn(input: EmptyModel, ctx: Context) -> dict[str, list[Any]]:\n    print(\"spawning child\")\n\n    results = []\n\n    for i in range(2):\n        try:\n            results.append(\n                (\n                    dedupe_child_wf.aio_run(\n                        options=TriggerWorkflowOptions(\n                            additional_metadata={\"dedupe\": \"test\"}, key=f\"child{i}\"\n                        ),\n                    )\n                )\n            )\n        except DedupeViolationErr as e:\n            print(f\"dedupe violation {e}\")\n            continue\n\n    result = await asyncio.gather(*results)\n    print(f\"results {result}\")\n\n    return {\"results\": result}\n\n\n@dedupe_child_wf.task()\nasync def process(input: EmptyModel, ctx: Context) -> dict[str, str]:\n    await asyncio.sleep(3)\n\n    print(\"child process\")\n    return {\"status\": \"success\"}\n\n\n@dedupe_child_wf.task()\nasync def process2(input: EmptyModel, ctx: Context) -> dict[str, str]:\n    print(\"child process2\")\n    return {\"status2\": \"success\"}\n\n\ndef main() -> None:\n    worker = hatchet.worker(\n        \"fanout-worker\", slots=100, workflows=[dedupe_parent_wf, dedupe_child_wf]\n    )\n    worker.start()\n\n\nif __name__ == \"__main__\":\n    main()\n",
    "language": "py",
    "source": "/Users/gabrielruttner/dev/hatchet/examples/python/dedupe/worker.py"
  },
  "L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3B5dGhvbi9kZWxheWVkL3dvcmtlci5weQ__": {
    "content": "from datetime import datetime, timedelta\n\nfrom pydantic import BaseModel\n\nfrom hatchet_sdk import Context, Hatchet\n\nhatchet = Hatchet(debug=True)\n\n\nclass PrinterInput(BaseModel):\n    message: str\n\n\nprint_schedule_wf = hatchet.workflow(\n    name=\"PrintScheduleWorkflow\",\n    input_validator=PrinterInput,\n)\nprint_printer_wf = hatchet.workflow(\n    name=\"PrintPrinterWorkflow\", input_validator=PrinterInput\n)\n\n\n@print_schedule_wf.task()\ndef schedule(input: PrinterInput, ctx: Context) -> None:\n    now = datetime.now()\n    print(f\"the time is \\t {now.strftime('%H:%M:%S')}\")\n    future_time = now + timedelta(seconds=15)\n    print(f\"scheduling for \\t {future_time.strftime('%H:%M:%S')}\")\n\n    print_printer_wf.schedule(future_time, input=input)\n\n\n@print_schedule_wf.task()\ndef step1(input: PrinterInput, ctx: Context) -> None:\n    now = datetime.now()\n    print(f\"printed at \\t {now.strftime('%H:%M:%S')}\")\n    print(f\"message \\t {input.message}\")\n\n\ndef main() -> None:\n    worker = hatchet.worker(\n        \"delayed-worker\", slots=4, workflows=[print_schedule_wf, print_printer_wf]\n    )\n\n    worker.start()\n\n\nif __name__ == \"__main__\":\n    main()\n",
    "language": "py",
    "source": "/Users/gabrielruttner/dev/hatchet/examples/python/delayed/worker.py"
  },
  "L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3B5dGhvbi9kdXJhYmxlL3dvcmtlci5weQ__": {
    "content": "from datetime import timedelta\n\nfrom hatchet_sdk import Context, DurableContext, EmptyModel, Hatchet, UserEventCondition\n\nhatchet = Hatchet(debug=True)\n\n# ❓ Create a durable workflow\ndurable_workflow = hatchet.workflow(name=\"DurableWorkflow\")\n# !!\n\n\nephemeral_workflow = hatchet.workflow(name=\"EphemeralWorkflow\")\n\n\n# ❓ Add durable task\nEVENT_KEY = \"durable-example:event\"\nSLEEP_TIME = 5\n\n\n@durable_workflow.task()\nasync def ephemeral_task(input: EmptyModel, ctx: Context) -> None:\n    print(\"Running non-durable task\")\n\n\n@durable_workflow.durable_task()\nasync def durable_task(input: EmptyModel, ctx: DurableContext) -> dict[str, str]:\n    print(\"Waiting for sleep\")\n    await ctx.aio_sleep_for(duration=timedelta(seconds=SLEEP_TIME))\n    print(\"Sleep finished\")\n\n    print(\"Waiting for event\")\n    await ctx.aio_wait_for(\n        \"event\",\n        UserEventCondition(event_key=EVENT_KEY, expression=\"true\"),\n    )\n    print(\"Event received\")\n\n    return {\n        \"status\": \"success\",\n    }\n\n\n# !!\n\n\n@ephemeral_workflow.task()\ndef ephemeral_task_2(input: EmptyModel, ctx: Context) -> None:\n    print(\"Running non-durable task\")\n\n\ndef main() -> None:\n    worker = hatchet.worker(\n        \"durable-worker\", workflows=[durable_workflow, ephemeral_workflow]\n    )\n    worker.start()\n\n\nif __name__ == \"__main__\":\n    main()\n",
    "language": "py",
    "source": "/Users/gabrielruttner/dev/hatchet/examples/python/durable/worker.py"
  },
  "L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3B5dGhvbi9kdXJhYmxlX2V2ZW50L3dvcmtlci5weQ__": {
    "content": "from hatchet_sdk import DurableContext, EmptyModel, Hatchet, UserEventCondition\n\nhatchet = Hatchet(debug=True)\n\nEVENT_KEY = \"user:update\"\n\n\n# ❓ Durable Event\n@hatchet.durable_task(name=\"DurableEventTask\")\nasync def durable_event_task(input: EmptyModel, ctx: DurableContext) -> None:\n    res = await ctx.aio_wait_for(\n        \"event\",\n        UserEventCondition(event_key=\"user:update\"),\n    )\n\n    print(\"got event\", res)\n\n\n# !!\n\n\n@hatchet.durable_task(name=\"DurableEventWithFilterTask\")\nasync def durable_event_task_with_filter(\n    input: EmptyModel, ctx: DurableContext\n) -> None:\n    # ❓ Durable Event With Filter\n    res = await ctx.aio_wait_for(\n        \"event\",\n        UserEventCondition(\n            event_key=\"user:update\", expression=\"input.user_id == '1234'\"\n        ),\n    )\n    # !!\n\n    print(\"got event\", res)\n\n\ndef main() -> None:\n    worker = hatchet.worker(\n        \"durable-event-worker\",\n        workflows=[durable_event_task, durable_event_task_with_filter],\n    )\n    worker.start()\n\n\nif __name__ == \"__main__\":\n    main()\n",
    "language": "py",
    "source": "/Users/gabrielruttner/dev/hatchet/examples/python/durable_event/worker.py"
  },
  "L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3B5dGhvbi9kdXJhYmxlX3NsZWVwL3dvcmtlci5weQ__": {
    "content": "from datetime import timedelta\n\nfrom hatchet_sdk import DurableContext, EmptyModel, Hatchet\n\nhatchet = Hatchet(debug=True)\n\n\n# ❓ Durable Sleep\n@hatchet.durable_task(name=\"DurableSleepTask\")\nasync def durable_sleep_task(input: EmptyModel, ctx: DurableContext) -> None:\n    res = await ctx.aio_sleep_for(timedelta(seconds=5))\n\n    print(\"got result\", res)\n\n\n# !!\n\n\ndef main() -> None:\n    worker = hatchet.worker(\"durable-sleep-worker\", workflows=[durable_sleep_task])\n    worker.start()\n\n\nif __name__ == \"__main__\":\n    main()\n",
    "language": "py",
    "source": "/Users/gabrielruttner/dev/hatchet/examples/python/durable_sleep/worker.py"
  },
  "L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3B5dGhvbi9ldmVudHMvd29ya2VyLnB5": {
    "content": "from hatchet_sdk import Context, EmptyModel, Hatchet\n\nhatchet = Hatchet()\n\n# ❓ Event trigger\nevent_workflow = hatchet.workflow(name=\"EventWorkflow\", on_events=[\"user:create\"])\n# ‼️\n\n\n@event_workflow.task()\ndef task(input: EmptyModel, ctx: Context) -> None:\n    print(\"event received\")\n",
    "language": "py",
    "source": "/Users/gabrielruttner/dev/hatchet/examples/python/events/worker.py"
  },
  "L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3B5dGhvbi9mYW5vdXQvd29ya2VyLnB5": {
    "content": "from datetime import timedelta\nfrom typing import Any\n\nfrom pydantic import BaseModel\n\nfrom hatchet_sdk import Context, Hatchet, TriggerWorkflowOptions\n\nhatchet = Hatchet(debug=True)\n\n\n# ❓ FanoutParent\nclass ParentInput(BaseModel):\n    n: int = 100\n\n\nclass ChildInput(BaseModel):\n    a: str\n\n\nparent_wf = hatchet.workflow(name=\"FanoutParent\", input_validator=ParentInput)\nchild_wf = hatchet.workflow(name=\"FanoutChild\", input_validator=ChildInput)\n\n\n@parent_wf.task(execution_timeout=timedelta(minutes=5))\nasync def spawn(input: ParentInput, ctx: Context) -> dict[str, Any]:\n    print(\"spawning child\")\n\n    result = await child_wf.aio_run_many(\n        [\n            child_wf.create_bulk_run_item(\n                input=ChildInput(a=str(i)),\n                options=TriggerWorkflowOptions(\n                    additional_metadata={\"hello\": \"earth\"}, key=f\"child{i}\"\n                ),\n            )\n            for i in range(input.n)\n        ]\n    )\n\n    print(f\"results {result}\")\n\n    return {\"results\": result}\n\n\n# ‼️\n\n\n# ❓ FanoutChild\n@child_wf.task()\ndef process(input: ChildInput, ctx: Context) -> dict[str, str]:\n    print(f\"child process {input.a}\")\n    return {\"status\": input.a}\n\n\n@child_wf.task(parents=[process])\ndef process2(input: ChildInput, ctx: Context) -> dict[str, str]:\n    process_output = ctx.task_output(process)\n    a = process_output[\"status\"]\n\n    return {\"status2\": a + \"2\"}\n\n\n# ‼️\n\nchild_wf.create_bulk_run_item()\n\n\ndef main() -> None:\n    worker = hatchet.worker(\"fanout-worker\", slots=40, workflows=[parent_wf, child_wf])\n    worker.start()\n\n\nif __name__ == \"__main__\":\n    main()\n",
    "language": "py",
    "source": "/Users/gabrielruttner/dev/hatchet/examples/python/fanout/worker.py"
  },
  "L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3B5dGhvbi9mYW5vdXRfc3luYy93b3JrZXIucHk_": {
    "content": "from datetime import timedelta\nfrom typing import Any\n\nfrom pydantic import BaseModel\n\nfrom hatchet_sdk import Context, Hatchet, TriggerWorkflowOptions\n\nhatchet = Hatchet(debug=True)\n\n\nclass ParentInput(BaseModel):\n    n: int = 5\n\n\nclass ChildInput(BaseModel):\n    a: str\n\n\nsync_fanout_parent = hatchet.workflow(\n    name=\"SyncFanoutParent\", input_validator=ParentInput\n)\nsync_fanout_child = hatchet.workflow(name=\"SyncFanoutChild\", input_validator=ChildInput)\n\n\n@sync_fanout_parent.task(execution_timeout=timedelta(minutes=5))\ndef spawn(input: ParentInput, ctx: Context) -> dict[str, list[dict[str, Any]]]:\n    print(\"spawning child\")\n\n    results = sync_fanout_child.run_many(\n        [\n            sync_fanout_child.create_bulk_run_item(\n                input=ChildInput(a=str(i)),\n                key=f\"child{i}\",\n                options=TriggerWorkflowOptions(additional_metadata={\"hello\": \"earth\"}),\n            )\n            for i in range(input.n)\n        ],\n    )\n\n    print(f\"results {results}\")\n\n    return {\"results\": results}\n\n\n@sync_fanout_child.task()\ndef process(input: ChildInput, ctx: Context) -> dict[str, str]:\n    return {\"status\": \"success \" + input.a}\n\n\ndef main() -> None:\n    worker = hatchet.worker(\n        \"sync-fanout-worker\",\n        slots=40,\n        workflows=[sync_fanout_parent, sync_fanout_child],\n    )\n    worker.start()\n\n\nif __name__ == \"__main__\":\n    main()\n",
    "language": "py",
    "source": "/Users/gabrielruttner/dev/hatchet/examples/python/fanout_sync/worker.py"
  },
  "L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3B5dGhvbi9saWZlc3BhbnMvd29ya2VyLnB5": {
    "content": "from typing import AsyncGenerator, cast\nfrom uuid import UUID\n\nfrom psycopg_pool import ConnectionPool\nfrom pydantic import BaseModel, ConfigDict\n\nfrom hatchet_sdk import Context, EmptyModel, Hatchet\n\nhatchet = Hatchet(debug=True)\n\n\n# ❓ Use the lifespan in a task\nclass TaskOutput(BaseModel):\n    num_rows: int\n    external_ids: list[UUID]\n\n\nlifespan_workflow = hatchet.workflow(name=\"LifespanWorkflow\")\n\n\n@lifespan_workflow.task()\ndef sync_lifespan_task(input: EmptyModel, ctx: Context) -> TaskOutput:\n    pool = cast(Lifespan, ctx.lifespan).pool\n\n    with pool.connection() as conn:\n        query = conn.execute(\"SELECT * FROM v1_lookup_table_olap LIMIT 5;\")\n        rows = query.fetchall()\n\n        for row in rows:\n            print(row)\n\n        print(\"executed sync task with lifespan\", ctx.lifespan)\n\n        return TaskOutput(\n            num_rows=len(rows),\n            external_ids=[cast(UUID, row[0]) for row in rows],\n        )\n\n\n# !!\n\n\n@lifespan_workflow.task()\nasync def async_lifespan_task(input: EmptyModel, ctx: Context) -> TaskOutput:\n    pool = cast(Lifespan, ctx.lifespan).pool\n\n    with pool.connection() as conn:\n        query = conn.execute(\"SELECT * FROM v1_lookup_table_olap LIMIT 5;\")\n        rows = query.fetchall()\n\n        for row in rows:\n            print(row)\n\n        print(\"executed async task with lifespan\", ctx.lifespan)\n\n        return TaskOutput(\n            num_rows=len(rows),\n            external_ids=[cast(UUID, row[0]) for row in rows],\n        )\n\n\n# ❓ Define a lifespan\nclass Lifespan(BaseModel):\n    model_config = ConfigDict(arbitrary_types_allowed=True)\n\n    foo: str\n    pool: ConnectionPool\n\n\nasync def lifespan() -> AsyncGenerator[Lifespan, None]:\n    print(\"Running lifespan!\")\n    with ConnectionPool(\"postgres://hatchet:hatchet@localhost:5431/hatchet\") as pool:\n        yield Lifespan(\n            foo=\"bar\",\n            pool=pool,\n        )\n\n    print(\"Cleaning up lifespan!\")\n\n\nworker = hatchet.worker(\n    \"test-worker\", slots=1, workflows=[lifespan_workflow], lifespan=lifespan\n)\n# !!\n\n\ndef main() -> None:\n    worker.start()\n\n\nif __name__ == \"__main__\":\n    main()\n",
    "language": "py",
    "source": "/Users/gabrielruttner/dev/hatchet/examples/python/lifespans/worker.py"
  },
  "L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3B5dGhvbi9sb2dnZXIvd29ya2VyLnB5": {
    "content": "from examples.logger.client import hatchet\nfrom examples.logger.workflow import logging_workflow\n\n\ndef main() -> None:\n    worker = hatchet.worker(\"logger-worker\", slots=5, workflows=[logging_workflow])\n\n    worker.start()\n\n\nif __name__ == \"__main__\":\n    main()\n",
    "language": "py",
    "source": "/Users/gabrielruttner/dev/hatchet/examples/python/logger/worker.py"
  },
  "L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3B5dGhvbi9sb2dnZXIvd29ya2Zsb3cucHk_": {
    "content": "# ❓ LoggingWorkflow\n\nimport logging\nimport time\n\nfrom examples.logger.client import hatchet\nfrom hatchet_sdk import Context, EmptyModel\n\nlogger = logging.getLogger(__name__)\n\nlogging_workflow = hatchet.workflow(\n    name=\"LoggingWorkflow\",\n)\n\n\n@logging_workflow.task()\ndef root_logger(input: EmptyModel, ctx: Context) -> dict[str, str]:\n    for i in range(12):\n        logger.info(\"executed step1 - {}\".format(i))\n        logger.info({\"step1\": \"step1\"})\n\n        time.sleep(0.1)\n\n    return {\"status\": \"success\"}\n\n\n# ‼️\n\n# ❓ ContextLogger\n\n\n@logging_workflow.task()\ndef context_logger(input: EmptyModel, ctx: Context) -> dict[str, str]:\n    for i in range(12):\n        ctx.log(\"executed step1 - {}\".format(i))\n        ctx.log({\"step1\": \"step1\"})\n\n        time.sleep(0.1)\n\n    return {\"status\": \"success\"}\n\n\n# ‼️\n",
    "language": "py",
    "source": "/Users/gabrielruttner/dev/hatchet/examples/python/logger/workflow.py"
  },
  "L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3B5dGhvbi9tYW51YWxfc2xvdF9yZWxlYXNlL3dvcmtlci5weQ__": {
    "content": "import time\n\nfrom hatchet_sdk import Context, EmptyModel, Hatchet\n\nhatchet = Hatchet()\n\n# ❓ SlotRelease\n\nslot_release_workflow = hatchet.workflow(name=\"SlotReleaseWorkflow\")\n\n\n@slot_release_workflow.task()\ndef step1(input: EmptyModel, ctx: Context) -> dict[str, str]:\n    print(\"RESOURCE INTENSIVE PROCESS\")\n    time.sleep(10)\n\n    # 👀 Release the slot after the resource-intensive process, so that other steps can run\n    ctx.release_slot()\n\n    print(\"NON RESOURCE INTENSIVE PROCESS\")\n    return {\"status\": \"success\"}\n\n\n# ‼️\n",
    "language": "py",
    "source": "/Users/gabrielruttner/dev/hatchet/examples/python/manual_slot_release/worker.py"
  },
  "L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3B5dGhvbi9ub25fcmV0cnlhYmxlL3dvcmtlci5weQ__": {
    "content": "from hatchet_sdk import Context, EmptyModel, Hatchet\nfrom hatchet_sdk.exceptions import NonRetryableException\n\nhatchet = Hatchet(debug=True)\n\nnon_retryable_workflow = hatchet.workflow(name=\"NonRetryableWorkflow\")\n\n\n# ❓ Non-retryable task\n@non_retryable_workflow.task(retries=1)\ndef should_not_retry(input: EmptyModel, ctx: Context) -> None:\n    raise NonRetryableException(\"This task should not retry\")\n\n\n# !!\n\n\n@non_retryable_workflow.task(retries=1)\ndef should_retry_wrong_exception_type(input: EmptyModel, ctx: Context) -> None:\n    raise TypeError(\"This task should retry because it's not a NonRetryableException\")\n\n\n@non_retryable_workflow.task(retries=1)\ndef should_not_retry_successful_task(input: EmptyModel, ctx: Context) -> None:\n    pass\n\n\ndef main() -> None:\n    worker = hatchet.worker(\"non-retry-worker\", workflows=[non_retryable_workflow])\n\n    worker.start()\n\n\nif __name__ == \"__main__\":\n    main()\n",
    "language": "py",
    "source": "/Users/gabrielruttner/dev/hatchet/examples/python/non_retryable/worker.py"
  },
  "L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3B5dGhvbi9vbl9mYWlsdXJlL3dvcmtlci5weQ__": {
    "content": "import json\nfrom datetime import timedelta\n\nfrom hatchet_sdk import Context, EmptyModel, Hatchet\n\nhatchet = Hatchet(debug=True)\n\nERROR_TEXT = \"step1 failed\"\n\n# ❓ OnFailure Step\n# This workflow will fail because the step will throw an error\n# we define an onFailure step to handle this case\n\non_failure_wf = hatchet.workflow(name=\"OnFailureWorkflow\")\n\n\n@on_failure_wf.task(execution_timeout=timedelta(seconds=1))\ndef step1(input: EmptyModel, ctx: Context) -> None:\n    # 👀 this step will always raise an exception\n    raise Exception(ERROR_TEXT)\n\n\n# 👀 After the workflow fails, this special step will run\n@on_failure_wf.on_failure_task()\ndef on_failure(input: EmptyModel, ctx: Context) -> dict[str, str]:\n    # 👀 we can do things like perform cleanup logic\n    # or notify a user here\n\n    # 👀 Fetch the errors from upstream step runs from the context\n    print(ctx.task_run_errors)\n\n    return {\"status\": \"success\"}\n\n\n# ‼️\n\n\n# ❓ OnFailure With Details\n# We can access the failure details in the onFailure step\n# via the context method\n\non_failure_wf_with_details = hatchet.workflow(name=\"OnFailureWorkflowWithDetails\")\n\n\n# ... defined as above\n@on_failure_wf_with_details.task(execution_timeout=timedelta(seconds=1))\ndef details_step1(input: EmptyModel, ctx: Context) -> None:\n    raise Exception(ERROR_TEXT)\n\n\n# 👀 After the workflow fails, this special step will run\n@on_failure_wf_with_details.on_failure_task()\ndef details_on_failure(input: EmptyModel, ctx: Context) -> dict[str, str]:\n    error = ctx.fetch_task_run_error(details_step1)\n\n    # 👀 we can access the failure details here\n    print(json.dumps(error, indent=2))\n\n    if error and error.startswith(ERROR_TEXT):\n        return {\"status\": \"success\"}\n\n    raise Exception(\"unexpected failure\")\n\n\n# ‼️\n\n\ndef main() -> None:\n    worker = hatchet.worker(\n        \"on-failure-worker\",\n        slots=4,\n        workflows=[on_failure_wf, on_failure_wf_with_details],\n    )\n    worker.start()\n\n\nif __name__ == \"__main__\":\n    main()\n",
    "language": "py",
    "source": "/Users/gabrielruttner/dev/hatchet/examples/python/on_failure/worker.py"
  },
  "L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3B5dGhvbi9vbl9zdWNjZXNzL3dvcmtlci5weQ__": {
    "content": "from hatchet_sdk import Context, EmptyModel, Hatchet\n\nhatchet = Hatchet(debug=True)\n\non_success_workflow = hatchet.workflow(name=\"OnSuccessWorkflow\")\n\n\n@on_success_workflow.task()\ndef first_task(input: EmptyModel, ctx: Context) -> None:\n    print(\"First task completed successfully\")\n\n\n@on_success_workflow.task(parents=[first_task])\ndef second_task(input: EmptyModel, ctx: Context) -> None:\n    print(\"Second task completed successfully\")\n\n\n@on_success_workflow.task(parents=[first_task, second_task])\ndef third_task(input: EmptyModel, ctx: Context) -> None:\n    print(\"Third task completed successfully\")\n\n\n@on_success_workflow.task()\ndef fourth_task(input: EmptyModel, ctx: Context) -> None:\n    print(\"Fourth task completed successfully\")\n\n\n@on_success_workflow.on_success_task()\ndef on_success_task(input: EmptyModel, ctx: Context) -> None:\n    print(\"On success task completed successfully\")\n\n\ndef main() -> None:\n    worker = hatchet.worker(\"on-success-worker\", workflows=[on_success_workflow])\n    worker.start()\n\n\nif __name__ == \"__main__\":\n    main()\n",
    "language": "py",
    "source": "/Users/gabrielruttner/dev/hatchet/examples/python/on_success/worker.py"
  },
  "L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3B5dGhvbi9vcGVudGVsZW1ldHJ5X2luc3RydW1lbnRhdGlvbi93b3JrZXIucHk_": {
    "content": "from examples.opentelemetry_instrumentation.client import hatchet\nfrom examples.opentelemetry_instrumentation.tracer import trace_provider\nfrom hatchet_sdk import Context, EmptyModel\nfrom hatchet_sdk.opentelemetry.instrumentor import HatchetInstrumentor\n\nHatchetInstrumentor(\n    tracer_provider=trace_provider,\n).instrument()\n\notel_workflow = hatchet.workflow(\n    name=\"OTelWorkflow\",\n)\n\n\n@otel_workflow.task()\ndef your_spans_are_children_of_hatchet_span(\n    input: EmptyModel, ctx: Context\n) -> dict[str, str]:\n    with trace_provider.get_tracer(__name__).start_as_current_span(\"step1\"):\n        print(\"executed step\")\n        return {\n            \"foo\": \"bar\",\n        }\n\n\n@otel_workflow.task()\ndef your_spans_are_still_children_of_hatchet_span(\n    input: EmptyModel, ctx: Context\n) -> None:\n    with trace_provider.get_tracer(__name__).start_as_current_span(\"step2\"):\n        raise Exception(\"Manually instrumented step failed failed\")\n\n\n@otel_workflow.task()\ndef this_step_is_still_instrumented(input: EmptyModel, ctx: Context) -> dict[str, str]:\n    print(\"executed still-instrumented step\")\n    return {\n        \"still\": \"instrumented\",\n    }\n\n\n@otel_workflow.task()\ndef this_step_is_also_still_instrumented(input: EmptyModel, ctx: Context) -> None:\n    raise Exception(\"Still-instrumented step failed\")\n\n\ndef main() -> None:\n    worker = hatchet.worker(\"otel-example-worker\", slots=1, workflows=[otel_workflow])\n    worker.start()\n\n\nif __name__ == \"__main__\":\n    main()\n",
    "language": "py",
    "source": "/Users/gabrielruttner/dev/hatchet/examples/python/opentelemetry_instrumentation/worker.py"
  },
  "L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3B5dGhvbi9wcmlvcml0eS93b3JrZXIucHk_": {
    "content": "import time\n\nfrom hatchet_sdk import (\n    ConcurrencyExpression,\n    ConcurrencyLimitStrategy,\n    Context,\n    EmptyModel,\n    Hatchet,\n)\n\nhatchet = Hatchet(debug=True)\n\n# ❓ Default priority\nDEFAULT_PRIORITY = 1\nSLEEP_TIME = 0.25\n\npriority_workflow = hatchet.workflow(\n    name=\"PriorityWorkflow\",\n    default_priority=DEFAULT_PRIORITY,\n    concurrency=ConcurrencyExpression(\n        max_runs=1,\n        expression=\"'true'\",\n        limit_strategy=ConcurrencyLimitStrategy.GROUP_ROUND_ROBIN,\n    ),\n)\n# !!\n\n\n@priority_workflow.task()\ndef priority_task(input: EmptyModel, ctx: Context) -> None:\n    print(\"Priority:\", ctx.priority)\n    time.sleep(SLEEP_TIME)\n\n\ndef main() -> None:\n    worker = hatchet.worker(\n        \"priority-worker\",\n        slots=1,\n        workflows=[priority_workflow],\n    )\n\n    worker.start()\n\n\nif __name__ == \"__main__\":\n    main()\n",
    "language": "py",
    "source": "/Users/gabrielruttner/dev/hatchet/examples/python/priority/worker.py"
  },
  "L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3B5dGhvbi9yYXRlX2xpbWl0L3dvcmtlci5weQ__": {
    "content": "from pydantic import BaseModel\n\nfrom hatchet_sdk import Context, Hatchet\nfrom hatchet_sdk.rate_limit import RateLimit, RateLimitDuration\n\nhatchet = Hatchet(debug=True)\n\n\n# ❓ Workflow\nclass RateLimitInput(BaseModel):\n    user_id: str\n\n\nrate_limit_workflow = hatchet.workflow(\n    name=\"RateLimitWorkflow\", input_validator=RateLimitInput\n)\n\n# !!\n\n\n# ❓ Static\nRATE_LIMIT_KEY = \"test-limit\"\n\n\n@rate_limit_workflow.task(rate_limits=[RateLimit(static_key=RATE_LIMIT_KEY, units=1)])\ndef step_1(input: RateLimitInput, ctx: Context) -> None:\n    print(\"executed step_1\")\n\n\n# !!\n\n# ❓ Dynamic\n\n\n@rate_limit_workflow.task(\n    rate_limits=[\n        RateLimit(\n            dynamic_key=\"input.user_id\",\n            units=1,\n            limit=10,\n            duration=RateLimitDuration.MINUTE,\n        )\n    ]\n)\ndef step_2(input: RateLimitInput, ctx: Context) -> None:\n    print(\"executed step_2\")\n\n\n# !!\n\n\ndef main() -> None:\n    hatchet.rate_limits.put(RATE_LIMIT_KEY, 2, RateLimitDuration.SECOND)\n\n    worker = hatchet.worker(\n        \"rate-limit-worker\", slots=10, workflows=[rate_limit_workflow]\n    )\n\n    worker.start()\n\n\nif __name__ == \"__main__\":\n    main()\n",
    "language": "py",
    "source": "/Users/gabrielruttner/dev/hatchet/examples/python/rate_limit/worker.py"
  },
  "L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3B5dGhvbi9yZXRyaWVzL3dvcmtlci5weQ__": {
    "content": "from hatchet_sdk import Context, EmptyModel, Hatchet\n\nhatchet = Hatchet(debug=True)\n\nsimple_workflow = hatchet.workflow(name=\"SimpleRetryWorkflow\")\nbackoff_workflow = hatchet.workflow(name=\"BackoffWorkflow\")\n\n\n# ❓ Simple Step Retries\n@simple_workflow.task(retries=3)\ndef always_fail(input: EmptyModel, ctx: Context) -> dict[str, str]:\n    raise Exception(\"simple task failed\")\n\n\n# ‼️\n\n\n# ❓ Retries with Count\n@simple_workflow.task(retries=3)\ndef fail_twice(input: EmptyModel, ctx: Context) -> dict[str, str]:\n    if ctx.retry_count < 2:\n        raise Exception(\"simple task failed\")\n\n    return {\"status\": \"success\"}\n\n\n# ‼️\n\n\n# ❓ Retries with Backoff\n@backoff_workflow.task(\n    retries=10,\n    # 👀 Maximum number of seconds to wait between retries\n    backoff_max_seconds=10,\n    # 👀 Factor to increase the wait time between retries.\n    # This sequence will be 2s, 4s, 8s, 10s, 10s, 10s... due to the maxSeconds limit\n    backoff_factor=2.0,\n)\ndef backoff_task(input: EmptyModel, ctx: Context) -> dict[str, str]:\n    if ctx.retry_count < 3:\n        raise Exception(\"backoff task failed\")\n\n    return {\"status\": \"success\"}\n\n\n# ‼️\n\n\ndef main() -> None:\n    worker = hatchet.worker(\"backoff-worker\", slots=4, workflows=[backoff_workflow])\n    worker.start()\n\n\nif __name__ == \"__main__\":\n    main()\n",
    "language": "py",
    "source": "/Users/gabrielruttner/dev/hatchet/examples/python/retries/worker.py"
  },
  "L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3B5dGhvbi9zaW1wbGUvd29ya2VyLnB5": {
    "content": "# ❓ Simple\n\nfrom hatchet_sdk import Context, EmptyModel, Hatchet\n\nhatchet = Hatchet(debug=True)\n\n\n@hatchet.task(name=\"SimpleWorkflow\")\ndef step1(input: EmptyModel, ctx: Context) -> None:\n    print(\"executed step1\")\n\n\ndef main() -> None:\n    worker = hatchet.worker(\"test-worker\", slots=1, workflows=[step1])\n    worker.start()\n\n\n# ‼️\n\nif __name__ == \"__main__\":\n    main()\n",
    "language": "py",
    "source": "/Users/gabrielruttner/dev/hatchet/examples/python/simple/worker.py"
  },
  "L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3B5dGhvbi9zdGlja3lfd29ya2Vycy93b3JrZXIucHk_": {
    "content": "from hatchet_sdk import (\n    Context,\n    EmptyModel,\n    Hatchet,\n    StickyStrategy,\n    TriggerWorkflowOptions,\n)\n\nhatchet = Hatchet(debug=True)\n\n# ❓ StickyWorker\n\n\nsticky_workflow = hatchet.workflow(\n    name=\"StickyWorkflow\",\n    # 👀 Specify a sticky strategy when declaring the workflow\n    sticky=StickyStrategy.SOFT,\n)\n\n\n@sticky_workflow.task()\ndef step1a(input: EmptyModel, ctx: Context) -> dict[str, str | None]:\n    return {\"worker\": ctx.worker.id()}\n\n\n@sticky_workflow.task()\ndef step1b(input: EmptyModel, ctx: Context) -> dict[str, str | None]:\n    return {\"worker\": ctx.worker.id()}\n\n\n# ‼️\n\n# ❓ StickyChild\n\nsticky_child_workflow = hatchet.workflow(\n    name=\"StickyChildWorkflow\", sticky=StickyStrategy.SOFT\n)\n\n\n@sticky_workflow.task(parents=[step1a, step1b])\nasync def step2(input: EmptyModel, ctx: Context) -> dict[str, str | None]:\n    ref = await sticky_child_workflow.aio_run_no_wait(\n        options=TriggerWorkflowOptions(sticky=True)\n    )\n\n    await ref.aio_result()\n\n    return {\"worker\": ctx.worker.id()}\n\n\n@sticky_child_workflow.task()\ndef child(input: EmptyModel, ctx: Context) -> dict[str, str | None]:\n    return {\"worker\": ctx.worker.id()}\n\n\n# ‼️\n\n\ndef main() -> None:\n    worker = hatchet.worker(\n        \"sticky-worker\", slots=10, workflows=[sticky_workflow, sticky_child_workflow]\n    )\n    worker.start()\n\n\nif __name__ == \"__main__\":\n    main()\n",
    "language": "py",
    "source": "/Users/gabrielruttner/dev/hatchet/examples/python/sticky_workers/worker.py"
  },
  "L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3B5dGhvbi9zdHJlYW1pbmcvd29ya2VyLnB5": {
    "content": "import asyncio\n\nfrom hatchet_sdk import Context, EmptyModel, Hatchet\n\nhatchet = Hatchet(debug=True)\n\n# ❓ Streaming\n\nstreaming_workflow = hatchet.workflow(name=\"StreamingWorkflow\")\n\n\n@streaming_workflow.task()\nasync def step1(input: EmptyModel, ctx: Context) -> None:\n    for i in range(10):\n        await asyncio.sleep(1)\n        ctx.put_stream(f\"Processing {i}\")\n\n\ndef main() -> None:\n    worker = hatchet.worker(\"test-worker\", workflows=[streaming_workflow])\n    worker.start()\n\n\n# ‼️\n\nif __name__ == \"__main__\":\n    main()\n",
    "language": "py",
    "source": "/Users/gabrielruttner/dev/hatchet/examples/python/streaming/worker.py"
  },
  "L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3B5dGhvbi90aW1lb3V0L3dvcmtlci5weQ__": {
    "content": "import time\nfrom datetime import timedelta\n\nfrom hatchet_sdk import Context, EmptyModel, Hatchet, TaskDefaults\n\nhatchet = Hatchet(debug=True)\n\n# ❓ ScheduleTimeout\ntimeout_wf = hatchet.workflow(\n    name=\"TimeoutWorkflow\",\n    task_defaults=TaskDefaults(execution_timeout=timedelta(minutes=2)),\n)\n# ‼️\n\n\n# ❓ ExecutionTimeout\n# 👀 Specify an execution timeout on a task\n@timeout_wf.task(\n    execution_timeout=timedelta(seconds=4), schedule_timeout=timedelta(minutes=10)\n)\ndef timeout_task(input: EmptyModel, ctx: Context) -> dict[str, str]:\n    time.sleep(5)\n    return {\"status\": \"success\"}\n\n\n# ‼️\n\nrefresh_timeout_wf = hatchet.workflow(name=\"RefreshTimeoutWorkflow\")\n\n\n# ❓ RefreshTimeout\n@refresh_timeout_wf.task(execution_timeout=timedelta(seconds=4))\ndef refresh_task(input: EmptyModel, ctx: Context) -> dict[str, str]:\n\n    ctx.refresh_timeout(timedelta(seconds=10))\n    time.sleep(5)\n\n    return {\"status\": \"success\"}\n\n\n# ‼️\n\n\ndef main() -> None:\n    worker = hatchet.worker(\n        \"timeout-worker\", slots=4, workflows=[timeout_wf, refresh_timeout_wf]\n    )\n\n    worker.start()\n\n\nif __name__ == \"__main__\":\n    main()\n",
    "language": "py",
    "source": "/Users/gabrielruttner/dev/hatchet/examples/python/timeout/worker.py"
  },
  "L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3B5dGhvbi93YWl0cy93b3JrZXIucHk_": {
    "content": "# ❓ Create a workflow\n\nimport random\nfrom datetime import timedelta\n\nfrom pydantic import BaseModel\n\nfrom hatchet_sdk import (\n    Context,\n    EmptyModel,\n    Hatchet,\n    ParentCondition,\n    SleepCondition,\n    UserEventCondition,\n    or_,\n)\n\nhatchet = Hatchet(debug=True)\n\n\nclass StepOutput(BaseModel):\n    random_number: int\n\n\nclass RandomSum(BaseModel):\n    sum: int\n\n\ntask_condition_workflow = hatchet.workflow(name=\"TaskConditionWorkflow\")\n\n# !!\n\n\n# ❓ Add base task\n@task_condition_workflow.task()\ndef start(input: EmptyModel, ctx: Context) -> StepOutput:\n    return StepOutput(random_number=random.randint(1, 100))\n\n\n# !!\n\n\n# ❓ Add wait for sleep\n@task_condition_workflow.task(\n    parents=[start], wait_for=[SleepCondition(timedelta(seconds=10))]\n)\ndef wait_for_sleep(input: EmptyModel, ctx: Context) -> StepOutput:\n    return StepOutput(random_number=random.randint(1, 100))\n\n\n# !!\n\n\n# ❓ Add skip on event\n@task_condition_workflow.task(\n    parents=[start],\n    wait_for=[SleepCondition(timedelta(seconds=30))],\n    skip_if=[UserEventCondition(event_key=\"skip_on_event:skip\")],\n)\ndef skip_on_event(input: EmptyModel, ctx: Context) -> StepOutput:\n    return StepOutput(random_number=random.randint(1, 100))\n\n\n# !!\n\n\n# ❓ Add branching\n@task_condition_workflow.task(\n    parents=[wait_for_sleep],\n    skip_if=[\n        ParentCondition(\n            parent=wait_for_sleep,\n            expression=\"output.random_number > 50\",\n        )\n    ],\n)\ndef left_branch(input: EmptyModel, ctx: Context) -> StepOutput:\n    return StepOutput(random_number=random.randint(1, 100))\n\n\n@task_condition_workflow.task(\n    parents=[wait_for_sleep],\n    skip_if=[\n        ParentCondition(\n            parent=wait_for_sleep,\n            expression=\"output.random_number <= 50\",\n        )\n    ],\n)\ndef right_branch(input: EmptyModel, ctx: Context) -> StepOutput:\n    return StepOutput(random_number=random.randint(1, 100))\n\n\n# !!\n\n\n# ❓ Add wait for event\n@task_condition_workflow.task(\n    parents=[start],\n    wait_for=[\n        or_(\n            SleepCondition(duration=timedelta(minutes=1)),\n            UserEventCondition(event_key=\"wait_for_event:start\"),\n        )\n    ],\n)\ndef wait_for_event(input: EmptyModel, ctx: Context) -> StepOutput:\n    return StepOutput(random_number=random.randint(1, 100))\n\n\n# !!\n\n\n# ❓ Add sum\n@task_condition_workflow.task(\n    parents=[\n        start,\n        wait_for_sleep,\n        wait_for_event,\n        skip_on_event,\n        left_branch,\n        right_branch,\n    ],\n)\ndef sum(input: EmptyModel, ctx: Context) -> RandomSum:\n    one = ctx.task_output(start).random_number\n    two = ctx.task_output(wait_for_event).random_number\n    three = ctx.task_output(wait_for_sleep).random_number\n    four = (\n        ctx.task_output(skip_on_event).random_number\n        if not ctx.was_skipped(skip_on_event)\n        else 0\n    )\n\n    five = (\n        ctx.task_output(left_branch).random_number\n        if not ctx.was_skipped(left_branch)\n        else 0\n    )\n    six = (\n        ctx.task_output(right_branch).random_number\n        if not ctx.was_skipped(right_branch)\n        else 0\n    )\n\n    return RandomSum(sum=one + two + three + four + five + six)\n\n\n# !!\n\n\ndef main() -> None:\n    worker = hatchet.worker(\"dag-worker\", workflows=[task_condition_workflow])\n\n    worker.start()\n\n\nif __name__ == \"__main__\":\n    main()\n",
    "language": "py",
    "source": "/Users/gabrielruttner/dev/hatchet/examples/python/waits/worker.py"
  },
  "L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3B5dGhvbi93b3JrZXIucHk_": {
    "content": "from examples.affinity_workers.worker import affinity_worker_workflow\nfrom examples.bulk_fanout.worker import bulk_child_wf, bulk_parent_wf\nfrom examples.cancellation.worker import cancellation_workflow\nfrom examples.concurrency_limit.worker import concurrency_limit_workflow\nfrom examples.concurrency_limit_rr.worker import concurrency_limit_rr_workflow\nfrom examples.concurrency_multiple_keys.worker import concurrency_multiple_keys_workflow\nfrom examples.concurrency_workflow_level.worker import (\n    concurrency_workflow_level_workflow,\n)\nfrom examples.dag.worker import dag_workflow\nfrom examples.dedupe.worker import dedupe_child_wf, dedupe_parent_wf\nfrom examples.durable.worker import durable_workflow\nfrom examples.fanout.worker import child_wf, parent_wf\nfrom examples.fanout_sync.worker import sync_fanout_child, sync_fanout_parent\nfrom examples.lifespans.simple import lifespan, lifespan_task\nfrom examples.logger.workflow import logging_workflow\nfrom examples.non_retryable.worker import non_retryable_workflow\nfrom examples.on_failure.worker import on_failure_wf, on_failure_wf_with_details\nfrom examples.priority.worker import priority_workflow\nfrom examples.timeout.worker import refresh_timeout_wf, timeout_wf\nfrom examples.waits.worker import task_condition_workflow\nfrom hatchet_sdk import Hatchet\n\nhatchet = Hatchet(debug=True)\n\n\ndef main() -> None:\n    worker = hatchet.worker(\n        \"e2e-test-worker\",\n        slots=100,\n        workflows=[\n            affinity_worker_workflow,\n            bulk_child_wf,\n            bulk_parent_wf,\n            concurrency_limit_workflow,\n            concurrency_limit_rr_workflow,\n            concurrency_multiple_keys_workflow,\n            dag_workflow,\n            dedupe_child_wf,\n            dedupe_parent_wf,\n            durable_workflow,\n            child_wf,\n            parent_wf,\n            on_failure_wf,\n            on_failure_wf_with_details,\n            logging_workflow,\n            timeout_wf,\n            refresh_timeout_wf,\n            task_condition_workflow,\n            cancellation_workflow,\n            sync_fanout_parent,\n            sync_fanout_child,\n            non_retryable_workflow,\n            concurrency_workflow_level_workflow,\n            priority_workflow,\n            lifespan_task,\n        ],\n        lifespan=lifespan,\n    )\n\n    worker.start()\n\n\nif __name__ == \"__main__\":\n    main()\n",
    "language": "py",
    "source": "/Users/gabrielruttner/dev/hatchet/examples/python/worker.py"
  },
  "L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3B5dGhvbi93b3JrZXJfZXhpc3RpbmdfbG9vcC93b3JrZXIucHk_": {
    "content": "import asyncio\nfrom contextlib import suppress\n\nfrom hatchet_sdk import Context, EmptyModel, Hatchet\n\nhatchet = Hatchet(debug=True)\n\nexisting_loop_worker = hatchet.workflow(name=\"WorkerExistingLoopWorkflow\")\n\n\n@existing_loop_worker.task()\nasync def task(input: EmptyModel, ctx: Context) -> dict[str, str]:\n    print(\"started\")\n    await asyncio.sleep(10)\n    print(\"finished\")\n    return {\"result\": \"returned result\"}\n\n\nasync def async_main() -> None:\n    worker = None\n    try:\n        worker = hatchet.worker(\n            \"test-worker\", slots=1, workflows=[existing_loop_worker]\n        )\n        worker.start()\n\n        ref = existing_loop_worker.run_no_wait()\n        print(await ref.aio_result())\n        while True:\n            await asyncio.sleep(1)\n    finally:\n        if worker:\n            await worker.exit_gracefully()\n\n\ndef main() -> None:\n    with suppress(KeyboardInterrupt):\n        asyncio.run(async_main())\n\n\nif __name__ == \"__main__\":\n    main()\n",
    "language": "py",
    "source": "/Users/gabrielruttner/dev/hatchet/examples/python/worker_existing_loop/worker.py"
  },
  "L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3B5dGhvbi93b3JrZmxvd19yZWdpc3RyYXRpb24vd29ya2VyLnB5": {
    "content": "# ❓ WorkflowRegistration\n\nfrom hatchet_sdk import Hatchet\n\nhatchet = Hatchet(debug=True)\n\nwf_one = hatchet.workflow(name=\"wf_one\")\nwf_two = hatchet.workflow(name=\"wf_two\")\nwf_three = hatchet.workflow(name=\"wf_three\")\nwf_four = hatchet.workflow(name=\"wf_four\")\nwf_five = hatchet.workflow(name=\"wf_five\")\n\n# define tasks here\n\n\ndef main() -> None:\n    # 👀 Register workflows directly when instantiating the worker\n    worker = hatchet.worker(\"test-worker\", slots=1, workflows=[wf_one, wf_two])\n\n    # 👀 Register a single workflow after instantiating the worker\n    worker.register_workflow(wf_three)\n\n    # 👀 Register multiple workflows in bulk after instantiating the worker\n    worker.register_workflows([wf_four, wf_five])\n\n    worker.start()\n\n\n# ‼️\n\nif __name__ == \"__main__\":\n    main()\n",
    "language": "py",
    "source": "/Users/gabrielruttner/dev/hatchet/examples/python/workflow_registration/worker.py"
  },
  "L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL2dvL3pfdjAvYXNzaWdubWVudC1hZmZpbml0eS9ydW4uZ28_": {
    "content": "package main\n\nimport (\n\t\"context\"\n\t\"fmt\"\n\t\"log\"\n\t\"time\"\n\n\t\"github.com/hatchet-dev/hatchet/pkg/client\"\n\t\"github.com/hatchet-dev/hatchet/pkg/client/types\"\n\t\"github.com/hatchet-dev/hatchet/pkg/worker\"\n)\n\nfunc run() (func() error, error) {\n\tc, err := client.New()\n\tif err != nil {\n\t\treturn nil, fmt.Errorf(\"error creating client: %w\", err)\n\t}\n\n\tw, err := worker.NewWorker(\n\t\tworker.WithClient(\n\t\t\tc,\n\t\t),\n\t\tworker.WithLabels(map[string]interface{}{\n\t\t\t\"model\":  \"fancy-ai-model-v2\",\n\t\t\t\"memory\": 1024,\n\t\t}),\n\t)\n\tif err != nil {\n\t\treturn nil, fmt.Errorf(\"error creating worker: %w\", err)\n\t}\n\n\terr = w.RegisterWorkflow(\n\t\t&worker.WorkflowJob{\n\t\t\tOn:          worker.Events(\"user:create:affinity\"),\n\t\t\tName:        \"affinity\",\n\t\t\tDescription: \"affinity\",\n\t\t\tSteps: []*worker.WorkflowStep{\n\t\t\t\tworker.Fn(func(ctx worker.HatchetContext) (result *stepOneOutput, err error) {\n\n\t\t\t\t\tmodel := ctx.Worker().GetLabels()[\"model\"]\n\n\t\t\t\t\tif model != \"fancy-ai-model-v3\" {\n\t\t\t\t\t\tctx.Worker().UpsertLabels(map[string]interface{}{\n\t\t\t\t\t\t\t\"model\": nil,\n\t\t\t\t\t\t})\n\t\t\t\t\t\t// Do something to load the model\n\t\t\t\t\t\tctx.Worker().UpsertLabels(map[string]interface{}{\n\t\t\t\t\t\t\t\"model\": \"fancy-ai-model-v3\",\n\t\t\t\t\t\t})\n\t\t\t\t\t}\n\n\t\t\t\t\treturn &stepOneOutput{\n\t\t\t\t\t\tMessage: ctx.Worker().ID(),\n\t\t\t\t\t}, nil\n\t\t\t\t}).\n\t\t\t\t\tSetName(\"step-one\").\n\t\t\t\t\tSetDesiredLabels(map[string]*types.DesiredWorkerLabel{\n\t\t\t\t\t\t\"model\": {\n\t\t\t\t\t\t\tValue:  \"fancy-ai-model-v3\",\n\t\t\t\t\t\t\tWeight: 10,\n\t\t\t\t\t\t},\n\t\t\t\t\t\t\"memory\": {\n\t\t\t\t\t\t\tValue:      512,\n\t\t\t\t\t\t\tRequired:   true,\n\t\t\t\t\t\t\tComparator: types.ComparatorPtr(types.WorkerLabelComparator_GREATER_THAN),\n\t\t\t\t\t\t},\n\t\t\t\t\t}),\n\t\t\t},\n\t\t},\n\t)\n\tif err != nil {\n\t\treturn nil, fmt.Errorf(\"error registering workflow: %w\", err)\n\t}\n\n\tgo func() {\n\t\tlog.Printf(\"pushing event\")\n\n\t\ttestEvent := userCreateEvent{\n\t\t\tUsername: \"echo-test\",\n\t\t\tUserID:   \"1234\",\n\t\t\tData: map[string]string{\n\t\t\t\t\"test\": \"test\",\n\t\t\t},\n\t\t}\n\n\t\t// push an event\n\t\terr := c.Event().Push(\n\t\t\tcontext.Background(),\n\t\t\t\"user:create:affinity\",\n\t\t\ttestEvent,\n\t\t)\n\t\tif err != nil {\n\t\t\tpanic(fmt.Errorf(\"error pushing event: %w\", err))\n\t\t}\n\n\t\ttime.Sleep(10 * time.Second)\n\t}()\n\n\tcleanup, err := w.Start()\n\tif err != nil {\n\t\treturn nil, fmt.Errorf(\"error starting worker: %w\", err)\n\t}\n\n\treturn cleanup, nil\n}\n",
    "language": "go",
    "source": "/Users/gabrielruttner/dev/hatchet/examples/go/z_v0/assignment-affinity/run.go"
  },
  "L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL2dvL3pfdjAvYXNzaWdubWVudC1zdGlja3kvcnVuLmdv": {
    "content": "package main\n\nimport (\n\t\"context\"\n\t\"fmt\"\n\t\"log\"\n\t\"time\"\n\n\t\"github.com/hatchet-dev/hatchet/pkg/client\"\n\t\"github.com/hatchet-dev/hatchet/pkg/client/types\"\n\t\"github.com/hatchet-dev/hatchet/pkg/worker\"\n)\n\nfunc run() (func() error, error) {\n\tc, err := client.New()\n\tif err != nil {\n\t\treturn nil, fmt.Errorf(\"error creating client: %w\", err)\n\t}\n\n\tw, err := worker.NewWorker(\n\t\tworker.WithClient(\n\t\t\tc,\n\t\t),\n\t)\n\tif err != nil {\n\t\treturn nil, fmt.Errorf(\"error creating worker: %w\", err)\n\t}\n\n\t// ❓ StickyWorker\n\n\terr = w.RegisterWorkflow(\n\t\t&worker.WorkflowJob{\n\t\t\tOn:          worker.Events(\"user:create:sticky\"),\n\t\t\tName:        \"sticky\",\n\t\t\tDescription: \"sticky\",\n\t\t\t// 👀 Specify a sticky strategy when declaring the workflow\n\t\t\tStickyStrategy: types.StickyStrategyPtr(types.StickyStrategy_HARD),\n\t\t\tSteps: []*worker.WorkflowStep{\n\t\t\t\tworker.Fn(func(ctx worker.HatchetContext) (result *stepOneOutput, err error) {\n\n\t\t\t\t\tsticky := true\n\n\t\t\t\t\t_, err = ctx.SpawnWorkflow(\"sticky-child\", nil, &worker.SpawnWorkflowOpts{\n\t\t\t\t\t\tSticky: &sticky,\n\t\t\t\t\t})\n\n\t\t\t\t\tif err != nil {\n\t\t\t\t\t\treturn nil, fmt.Errorf(\"error spawning workflow: %w\", err)\n\t\t\t\t\t}\n\n\t\t\t\t\treturn &stepOneOutput{\n\t\t\t\t\t\tMessage: ctx.Worker().ID(),\n\t\t\t\t\t}, nil\n\t\t\t\t}).SetName(\"step-one\"),\n\t\t\t\tworker.Fn(func(ctx worker.HatchetContext) (result *stepOneOutput, err error) {\n\t\t\t\t\treturn &stepOneOutput{\n\t\t\t\t\t\tMessage: ctx.Worker().ID(),\n\t\t\t\t\t}, nil\n\t\t\t\t}).SetName(\"step-two\").AddParents(\"step-one\"),\n\t\t\t\tworker.Fn(func(ctx worker.HatchetContext) (result *stepOneOutput, err error) {\n\t\t\t\t\treturn &stepOneOutput{\n\t\t\t\t\t\tMessage: ctx.Worker().ID(),\n\t\t\t\t\t}, nil\n\t\t\t\t}).SetName(\"step-three\").AddParents(\"step-two\"),\n\t\t\t},\n\t\t},\n\t)\n\n\t// ‼️\n\n\tif err != nil {\n\t\treturn nil, fmt.Errorf(\"error registering workflow: %w\", err)\n\t}\n\n\t// ❓ StickyChild\n\n\terr = w.RegisterWorkflow(\n\t\t&worker.WorkflowJob{\n\t\t\tOn:          worker.NoTrigger(),\n\t\t\tName:        \"sticky-child\",\n\t\t\tDescription: \"sticky\",\n\t\t\tSteps: []*worker.WorkflowStep{\n\t\t\t\tworker.Fn(func(ctx worker.HatchetContext) (result *stepOneOutput, err error) {\n\t\t\t\t\treturn &stepOneOutput{\n\t\t\t\t\t\tMessage: ctx.Worker().ID(),\n\t\t\t\t\t}, nil\n\t\t\t\t}).SetName(\"step-one\"),\n\t\t\t},\n\t\t},\n\t)\n\n\t// ‼️\n\n\tif err != nil {\n\t\treturn nil, fmt.Errorf(\"error registering workflow: %w\", err)\n\t}\n\n\tgo func() {\n\t\tlog.Printf(\"pushing event\")\n\n\t\ttestEvent := userCreateEvent{\n\t\t\tUsername: \"echo-test\",\n\t\t\tUserID:   \"1234\",\n\t\t\tData: map[string]string{\n\t\t\t\t\"test\": \"test\",\n\t\t\t},\n\t\t}\n\n\t\t// push an event\n\t\terr := c.Event().Push(\n\t\t\tcontext.Background(),\n\t\t\t\"user:create:sticky\",\n\t\t\ttestEvent,\n\t\t)\n\t\tif err != nil {\n\t\t\tpanic(fmt.Errorf(\"error pushing event: %w\", err))\n\t\t}\n\n\t\ttime.Sleep(10 * time.Second)\n\t}()\n\n\tcleanup, err := w.Start()\n\tif err != nil {\n\t\treturn nil, fmt.Errorf(\"error starting worker: %w\", err)\n\t}\n\n\treturn cleanup, nil\n}\n",
    "language": "go",
    "source": "/Users/gabrielruttner/dev/hatchet/examples/go/z_v0/assignment-sticky/run.go"
  },
  "L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL2dvL3pfdjAvYnVsa193b3JrZmxvd3MvcnVuLmdv": {
    "content": "package main\n\nimport (\n\t\"fmt\"\n\t\"log\"\n\n\t\"github.com/hatchet-dev/hatchet/pkg/client\"\n)\n\nfunc runBulk(workflowName string, quantity int) error {\n\tc, err := client.New()\n\tif err != nil {\n\t\treturn fmt.Errorf(\"error creating client: %w\", err)\n\t}\n\n\tlog.Printf(\"pushing %d workflows in bulk\", quantity)\n\n\tvar workflows []*client.WorkflowRun\n\tfor i := 0; i < quantity; i++ {\n\t\tdata := map[string]interface{}{\n\t\t\t\"username\": fmt.Sprintf(\"echo-test-%d\", i),\n\t\t\t\"user_id\":  fmt.Sprintf(\"1234-%d\", i),\n\t\t}\n\t\tworkflows = append(workflows, &client.WorkflowRun{\n\t\t\tName:  workflowName,\n\t\t\tInput: data,\n\t\t\tOptions: []client.RunOptFunc{\n\t\t\t\t// setting a dedupe key so these shouldn't all run\n\t\t\t\tclient.WithRunMetadata(map[string]interface{}{\n\t\t\t\t\t// \"dedupe\": \"dedupe1\",\n\t\t\t\t}),\n\t\t\t},\n\t\t})\n\n\t}\n\n\touts, err := c.Admin().BulkRunWorkflow(workflows)\n\tif err != nil {\n\t\tpanic(fmt.Errorf(\"error pushing event: %w\", err))\n\t}\n\n\tfor _, out := range outs {\n\t\tlog.Printf(\"workflow run id: %v\", out)\n\t}\n\n\treturn nil\n\n}\n\nfunc runSingles(workflowName string, quantity int) error {\n\tc, err := client.New()\n\tif err != nil {\n\t\treturn fmt.Errorf(\"error creating client: %w\", err)\n\t}\n\n\tlog.Printf(\"pushing %d single workflows\", quantity)\n\n\tvar workflows []*client.WorkflowRun\n\tfor i := 0; i < quantity; i++ {\n\t\tdata := map[string]interface{}{\n\t\t\t\"username\": fmt.Sprintf(\"echo-test-%d\", i),\n\t\t\t\"user_id\":  fmt.Sprintf(\"1234-%d\", i),\n\t\t}\n\t\tworkflows = append(workflows, &client.WorkflowRun{\n\t\t\tName:  workflowName,\n\t\t\tInput: data,\n\t\t\tOptions: []client.RunOptFunc{\n\t\t\t\tclient.WithRunMetadata(map[string]interface{}{\n\t\t\t\t\t// \"dedupe\": \"dedupe1\",\n\t\t\t\t}),\n\t\t\t},\n\t\t})\n\t}\n\n\tfor _, wf := range workflows {\n\n\t\tgo func() {\n\t\t\tout, err := c.Admin().RunWorkflow(wf.Name, wf.Input, wf.Options...)\n\t\t\tif err != nil {\n\t\t\t\tpanic(fmt.Errorf(\"error pushing event: %w\", err))\n\t\t\t}\n\n\t\t\tlog.Printf(\"workflow run id: %v\", out)\n\t\t}()\n\n\t}\n\n\treturn nil\n}\n",
    "language": "go",
    "source": "/Users/gabrielruttner/dev/hatchet/examples/go/z_v0/bulk_workflows/run.go"
  },
  "L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL2dvL3pfdjAvY2FuY2VsbGF0aW9uL3J1bi5nbw__": {
    "content": "package main\n\nimport (\n\t\"context\"\n\t\"fmt\"\n\t\"log\"\n\t\"time\"\n\n\t\"github.com/google/uuid\"\n\t\"github.com/hatchet-dev/hatchet/pkg/client\"\n\t\"github.com/hatchet-dev/hatchet/pkg/client/rest\"\n\t\"github.com/hatchet-dev/hatchet/pkg/worker\"\n)\n\nfunc run(events chan<- string) (func() error, error) {\n\tc, err := client.New()\n\tif err != nil {\n\t\treturn nil, fmt.Errorf(\"error creating client: %w\", err)\n\t}\n\n\tw, err := worker.NewWorker(\n\t\tworker.WithClient(\n\t\t\tc,\n\t\t),\n\t)\n\tif err != nil {\n\t\treturn nil, fmt.Errorf(\"error creating worker: %w\", err)\n\t}\n\n\terr = w.RegisterWorkflow(\n\t\t&worker.WorkflowJob{\n\t\t\tOn:          worker.Events(\"user:create:cancellation\"),\n\t\t\tName:        \"cancellation\",\n\t\t\tDescription: \"cancellation\",\n\t\t\tSteps: []*worker.WorkflowStep{\n\t\t\t\tworker.Fn(func(ctx worker.HatchetContext) (result *stepOneOutput, err error) {\n\t\t\t\t\tselect {\n\t\t\t\t\tcase <-ctx.Done():\n\t\t\t\t\t\tevents <- \"done\"\n\t\t\t\t\t\tlog.Printf(\"context cancelled\")\n\t\t\t\t\t\treturn nil, nil\n\t\t\t\t\tcase <-time.After(30 * time.Second):\n\t\t\t\t\t\tlog.Printf(\"workflow never cancelled\")\n\t\t\t\t\t\treturn &stepOneOutput{\n\t\t\t\t\t\t\tMessage: \"done\",\n\t\t\t\t\t\t}, nil\n\t\t\t\t\t}\n\t\t\t\t}).SetName(\"step-one\"),\n\t\t\t},\n\t\t},\n\t)\n\tif err != nil {\n\t\treturn nil, fmt.Errorf(\"error registering workflow: %w\", err)\n\t}\n\n\tgo func() {\n\t\tlog.Printf(\"pushing event\")\n\n\t\ttestEvent := userCreateEvent{\n\t\t\tUsername: \"echo-test\",\n\t\t\tUserID:   \"1234\",\n\t\t\tData: map[string]string{\n\t\t\t\t\"test\": \"test\",\n\t\t\t},\n\t\t}\n\n\t\t// push an event\n\t\terr := c.Event().Push(\n\t\t\tcontext.Background(),\n\t\t\t\"user:create:cancellation\",\n\t\t\ttestEvent,\n\t\t)\n\t\tif err != nil {\n\t\t\tpanic(fmt.Errorf(\"error pushing event: %w\", err))\n\t\t}\n\n\t\ttime.Sleep(10 * time.Second)\n\n\t\tworkflowName := \"cancellation\"\n\n\t\tworkflows, err := c.API().WorkflowListWithResponse(context.Background(), uuid.MustParse(c.TenantId()), &rest.WorkflowListParams{\n\t\t\tName: &workflowName,\n\t\t})\n\n\t\tif err != nil {\n\t\t\tpanic(fmt.Errorf(\"error listing workflows: %w\", err))\n\t\t}\n\n\t\tif workflows.JSON200 == nil {\n\t\t\tpanic(fmt.Errorf(\"no workflows found\"))\n\t\t}\n\n\t\trows := *workflows.JSON200.Rows\n\n\t\tif len(rows) == 0 {\n\t\t\tpanic(fmt.Errorf(\"no workflows found\"))\n\t\t}\n\n\t\tworkflowId := uuid.MustParse(rows[0].Metadata.Id)\n\n\t\tworkflowRuns, err := c.API().WorkflowRunListWithResponse(context.Background(), uuid.MustParse(c.TenantId()), &rest.WorkflowRunListParams{\n\t\t\tWorkflowId: &workflowId,\n\t\t})\n\n\t\tif err != nil {\n\t\t\tpanic(fmt.Errorf(\"error listing workflow runs: %w\", err))\n\t\t}\n\n\t\tif workflowRuns.JSON200 == nil {\n\t\t\tpanic(fmt.Errorf(\"no workflow runs found\"))\n\t\t}\n\n\t\tworkflowRunsRows := *workflowRuns.JSON200.Rows\n\n\t\t_, err = c.API().WorkflowRunCancelWithResponse(context.Background(), uuid.MustParse(c.TenantId()), rest.WorkflowRunsCancelRequest{\n\t\t\tWorkflowRunIds: []uuid.UUID{uuid.MustParse(workflowRunsRows[0].Metadata.Id)},\n\t\t})\n\n\t\tif err != nil {\n\t\t\tpanic(fmt.Errorf(\"error cancelling workflow run: %w\", err))\n\t\t}\n\t}()\n\n\tcleanup, err := w.Start()\n\tif err != nil {\n\t\treturn nil, fmt.Errorf(\"error starting worker: %w\", err)\n\t}\n\n\treturn cleanup, nil\n}\n",
    "language": "go",
    "source": "/Users/gabrielruttner/dev/hatchet/examples/go/z_v0/cancellation/run.go"
  },
  "L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL2dvL3pfdjAvbG9hZHRlc3QvY2xpL3J1bi5nbw__": {
    "content": "package main\n\nimport (\n\t\"context\"\n\t\"fmt\"\n\t\"math/rand/v2\"\n\t\"sync\"\n\t\"time\"\n\n\t\"github.com/hatchet-dev/hatchet/pkg/client\"\n\t\"github.com/hatchet-dev/hatchet/pkg/worker\"\n)\n\ntype stepOneOutput struct {\n\tMessage string `json:\"message\"`\n}\n\nfunc run(ctx context.Context, delay time.Duration, executions chan<- time.Duration, concurrency, slots int, failureRate float32, eventFanout int) (int64, int64) {\n\tc, err := client.New(\n\t\tclient.WithLogLevel(\"warn\"),\n\t)\n\n\tif err != nil {\n\t\tpanic(err)\n\t}\n\n\tw, err := worker.NewWorker(\n\t\tworker.WithClient(\n\t\t\tc,\n\t\t),\n\t\tworker.WithLogLevel(\"warn\"),\n\t\tworker.WithMaxRuns(slots),\n\t)\n\n\tif err != nil {\n\t\tpanic(err)\n\t}\n\n\tmx := sync.Mutex{}\n\tvar count int64\n\tvar uniques int64\n\tvar executed []int64\n\n\tvar concurrencyOpts *worker.WorkflowConcurrency\n\tif concurrency > 0 {\n\t\tconcurrencyOpts = worker.Expression(\"'global'\").MaxRuns(int32(concurrency))\n\t}\n\n\tstep := func(ctx worker.HatchetContext) (result *stepOneOutput, err error) {\n\t\tvar input Event\n\t\terr = ctx.WorkflowInput(&input)\n\t\tif err != nil {\n\t\t\treturn nil, err\n\t\t}\n\n\t\ttook := time.Since(input.CreatedAt)\n\t\tl.Info().Msgf(\"executing %d took %s\", input.ID, took)\n\n\t\tmx.Lock()\n\t\texecutions <- took\n\t\t// detect duplicate in executed slice\n\t\tvar duplicate bool\n\t\t// for i := 0; i < len(executed)-1; i++ {\n\t\t// \tif executed[i] == input.ID {\n\t\t// \t\tduplicate = true\n\t\t// \t\tbreak\n\t\t// \t}\n\t\t// }\n\t\tif duplicate {\n\t\t\tl.Warn().Str(\"step-run-id\", ctx.StepRunId()).Msgf(\"duplicate %d\", input.ID)\n\t\t}\n\t\tif !duplicate {\n\t\t\tuniques++\n\t\t}\n\t\tcount++\n\t\texecuted = append(executed, input.ID)\n\t\tmx.Unlock()\n\n\t\ttime.Sleep(delay)\n\n\t\tif failureRate > 0 {\n\t\t\tif rand.Float32() < failureRate {\n\t\t\t\treturn nil, fmt.Errorf(\"random failure\")\n\t\t\t}\n\t\t}\n\n\t\treturn &stepOneOutput{\n\t\t\tMessage: \"This ran at: \" + time.Now().Format(time.RFC3339Nano),\n\t\t}, nil\n\t}\n\n\tfor i := range eventFanout {\n\t\terr = w.RegisterWorkflow(\n\t\t\t&worker.WorkflowJob{\n\t\t\t\tName:        fmt.Sprintf(\"load-test-%d\", i),\n\t\t\t\tDescription: \"Load testing\",\n\t\t\t\tOn:          worker.Event(\"load-test:event\"),\n\t\t\t\tConcurrency: concurrencyOpts,\n\t\t\t\t// ScheduleTimeout: \"30s\",\n\t\t\t\tSteps: []*worker.WorkflowStep{\n\t\t\t\t\tworker.Fn(step).SetName(\"step-one\").SetTimeout(\"5m\"),\n\t\t\t\t},\n\t\t\t},\n\t\t)\n\n\t\tif err != nil {\n\t\t\tpanic(err)\n\t\t}\n\t}\n\n\tcleanup, err := w.Start()\n\tif err != nil {\n\t\tpanic(fmt.Errorf(\"error starting worker: %w\", err))\n\t}\n\n\t<-ctx.Done()\n\n\tif err := cleanup(); err != nil {\n\t\tpanic(fmt.Errorf(\"error cleaning up: %w\", err))\n\t}\n\n\tmx.Lock()\n\tdefer mx.Unlock()\n\treturn count, uniques\n}\n",
    "language": "go",
    "source": "/Users/gabrielruttner/dev/hatchet/examples/go/z_v0/loadtest/cli/run.go"
  },
  "L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL2dvL3pfdjAvbG9hZHRlc3QvcmFtcHVwL3J1bi5nbw__": {
    "content": "package rampup\n\nimport (\n\t\"context\"\n\t\"fmt\"\n\t\"sync\"\n\t\"time\"\n\n\t\"github.com/hatchet-dev/hatchet/pkg/client\"\n\t\"github.com/hatchet-dev/hatchet/pkg/worker\"\n)\n\ntype stepOneOutput struct {\n\tMessage string `json:\"message\"`\n}\n\nfunc getConcurrencyKey(ctx worker.HatchetContext) (string, error) {\n\treturn \"my-key\", nil\n}\n\nfunc run(ctx context.Context, delay time.Duration, concurrency int, maxAcceptableDuration time.Duration, hook chan<- time.Duration, executedCh chan<- int64) (int64, int64) {\n\tc, err := client.New(\n\t\tclient.WithLogLevel(\"warn\"),\n\t)\n\n\tif err != nil {\n\t\tpanic(err)\n\t}\n\n\tw, err := worker.NewWorker(\n\t\tworker.WithClient(\n\t\t\tc,\n\t\t),\n\t\tworker.WithLogLevel(\"warn\"),\n\t\tworker.WithMaxRuns(200),\n\t)\n\n\tif err != nil {\n\t\tpanic(err)\n\t}\n\n\tmx := sync.Mutex{}\n\tvar count int64\n\tvar uniques int64\n\tvar executed []int64\n\n\tvar concurrencyOpts *worker.WorkflowConcurrency\n\tif concurrency > 0 {\n\t\tconcurrencyOpts = worker.Concurrency(getConcurrencyKey).MaxRuns(int32(concurrency))\n\t}\n\n\terr = w.On(\n\t\tworker.Event(\"load-test:event\"),\n\t\t&worker.WorkflowJob{\n\t\t\tName:        \"load-test\",\n\t\t\tDescription: \"Load testing\",\n\t\t\tConcurrency: concurrencyOpts,\n\t\t\tSteps: []*worker.WorkflowStep{\n\t\t\t\tworker.Fn(func(ctx worker.HatchetContext) (result *stepOneOutput, err error) {\n\t\t\t\t\tvar input Event\n\t\t\t\t\terr = ctx.WorkflowInput(&input)\n\t\t\t\t\tif err != nil {\n\t\t\t\t\t\treturn nil, err\n\t\t\t\t\t}\n\n\t\t\t\t\ttook := time.Since(input.CreatedAt)\n\n\t\t\t\t\tl.Debug().Msgf(\"executing %d took %s\", input.ID, took)\n\n\t\t\t\t\tif took > maxAcceptableDuration {\n\t\t\t\t\t\thook <- took\n\t\t\t\t\t}\n\n\t\t\t\t\texecutedCh <- input.ID\n\n\t\t\t\t\tmx.Lock()\n\n\t\t\t\t\t// detect duplicate in executed slice\n\t\t\t\t\tvar duplicate bool\n\t\t\t\t\tfor i := 0; i < len(executed)-1; i++ {\n\t\t\t\t\t\tif executed[i] == input.ID {\n\t\t\t\t\t\t\tduplicate = true\n\t\t\t\t\t\t}\n\t\t\t\t\t}\n\t\t\t\t\tif duplicate {\n\t\t\t\t\t\tl.Warn().Str(\"step-run-id\", ctx.StepRunId()).Msgf(\"duplicate %d\", input.ID)\n\t\t\t\t\t} else {\n\t\t\t\t\t\tuniques += 1\n\t\t\t\t\t}\n\t\t\t\t\tcount += 1\n\t\t\t\t\texecuted = append(executed, input.ID)\n\t\t\t\t\tmx.Unlock()\n\n\t\t\t\t\ttime.Sleep(delay)\n\n\t\t\t\t\treturn &stepOneOutput{\n\t\t\t\t\t\tMessage: \"This ran at: \" + time.Now().Format(time.RFC3339Nano),\n\t\t\t\t\t}, nil\n\t\t\t\t}).SetName(\"step-one\"),\n\t\t\t},\n\t\t},\n\t)\n\n\tif err != nil {\n\t\tpanic(err)\n\t}\n\n\tcleanup, err := w.Start()\n\tif err != nil {\n\t\tpanic(fmt.Errorf(\"error starting worker: %w\", err))\n\t}\n\n\t<-ctx.Done()\n\n\tif err := cleanup(); err != nil {\n\t\tpanic(fmt.Errorf(\"error cleaning up: %w\", err))\n\t}\n\n\tmx.Lock()\n\tdefer mx.Unlock()\n\treturn count, uniques\n}\n",
    "language": "go",
    "source": "/Users/gabrielruttner/dev/hatchet/examples/go/z_v0/loadtest/rampup/run.go"
  },
  "L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL2dvL3pfdjAvbWlkZGxld2FyZS9ydW4uZ28_": {
    "content": "package main\n\nimport (\n\t\"context\"\n\t\"fmt\"\n\t\"log\"\n\t\"time\"\n\n\t\"github.com/hatchet-dev/hatchet/pkg/client\"\n\t\"github.com/hatchet-dev/hatchet/pkg/worker\"\n)\n\nfunc run(events chan<- string) (func() error, error) {\n\tc, err := client.New()\n\tif err != nil {\n\t\treturn nil, fmt.Errorf(\"error creating client: %w\", err)\n\t}\n\n\tw, err := worker.NewWorker(\n\t\tworker.WithClient(\n\t\t\tc,\n\t\t),\n\t)\n\tif err != nil {\n\t\treturn nil, fmt.Errorf(\"error creating worker: %w\", err)\n\t}\n\n\tw.Use(func(ctx worker.HatchetContext, next func(worker.HatchetContext) error) error {\n\t\tlog.Printf(\"1st-middleware\")\n\t\tevents <- \"1st-middleware\"\n\t\tctx.SetContext(context.WithValue(ctx.GetContext(), \"testkey\", \"testvalue\"))\n\t\treturn next(ctx)\n\t})\n\n\tw.Use(func(ctx worker.HatchetContext, next func(worker.HatchetContext) error) error {\n\t\tlog.Printf(\"2nd-middleware\")\n\t\tevents <- \"2nd-middleware\"\n\n\t\t// time the function duration\n\t\tstart := time.Now()\n\t\terr := next(ctx)\n\t\tduration := time.Since(start)\n\t\tfmt.Printf(\"step function took %s\\n\", duration)\n\t\treturn err\n\t})\n\n\ttestSvc := w.NewService(\"test\")\n\n\ttestSvc.Use(func(ctx worker.HatchetContext, next func(worker.HatchetContext) error) error {\n\t\tevents <- \"svc-middleware\"\n\t\tctx.SetContext(context.WithValue(ctx.GetContext(), \"svckey\", \"svcvalue\"))\n\t\treturn next(ctx)\n\t})\n\n\terr = testSvc.On(\n\t\tworker.Events(\"user:create:middleware\"),\n\t\t&worker.WorkflowJob{\n\t\t\tName:        \"middleware\",\n\t\t\tDescription: \"This runs after an update to the user model.\",\n\t\t\tSteps: []*worker.WorkflowStep{\n\t\t\t\tworker.Fn(func(ctx worker.HatchetContext) (result *stepOneOutput, err error) {\n\t\t\t\t\tinput := &userCreateEvent{}\n\n\t\t\t\t\terr = ctx.WorkflowInput(input)\n\n\t\t\t\t\tif err != nil {\n\t\t\t\t\t\treturn nil, err\n\t\t\t\t\t}\n\n\t\t\t\t\tlog.Printf(\"step-one\")\n\t\t\t\t\tevents <- \"step-one\"\n\n\t\t\t\t\ttestVal := ctx.Value(\"testkey\").(string)\n\t\t\t\t\tevents <- testVal\n\t\t\t\t\tsvcVal := ctx.Value(\"svckey\").(string)\n\t\t\t\t\tevents <- svcVal\n\n\t\t\t\t\treturn &stepOneOutput{\n\t\t\t\t\t\tMessage: \"Username is: \" + input.Username,\n\t\t\t\t\t}, nil\n\t\t\t\t},\n\t\t\t\t).SetName(\"step-one\"),\n\t\t\t\tworker.Fn(func(ctx worker.HatchetContext) (result *stepOneOutput, err error) {\n\t\t\t\t\tinput := &stepOneOutput{}\n\t\t\t\t\terr = ctx.StepOutput(\"step-one\", input)\n\n\t\t\t\t\tif err != nil {\n\t\t\t\t\t\treturn nil, err\n\t\t\t\t\t}\n\n\t\t\t\t\tlog.Printf(\"step-two\")\n\t\t\t\t\tevents <- \"step-two\"\n\n\t\t\t\t\treturn &stepOneOutput{\n\t\t\t\t\t\tMessage: \"Above message is: \" + input.Message,\n\t\t\t\t\t}, nil\n\t\t\t\t}).SetName(\"step-two\").AddParents(\"step-one\"),\n\t\t\t},\n\t\t},\n\t)\n\tif err != nil {\n\t\treturn nil, fmt.Errorf(\"error registering workflow: %w\", err)\n\t}\n\n\tgo func() {\n\t\tlog.Printf(\"pushing event user:create:middleware\")\n\n\t\ttestEvent := userCreateEvent{\n\t\t\tUsername: \"echo-test\",\n\t\t\tUserID:   \"1234\",\n\t\t\tData: map[string]string{\n\t\t\t\t\"test\": \"test\",\n\t\t\t},\n\t\t}\n\n\t\t// push an event\n\t\terr := c.Event().Push(\n\t\t\tcontext.Background(),\n\t\t\t\"user:create:middleware\",\n\t\t\ttestEvent,\n\t\t)\n\t\tif err != nil {\n\t\t\tpanic(fmt.Errorf(\"error pushing event: %w\", err))\n\t\t}\n\t}()\n\n\tcleanup, err := w.Start()\n\tif err != nil {\n\t\treturn nil, fmt.Errorf(\"error starting worker: %w\", err)\n\t}\n\n\treturn cleanup, nil\n}\n",
    "language": "go",
    "source": "/Users/gabrielruttner/dev/hatchet/examples/go/z_v0/middleware/run.go"
  },
  "L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL2dvL3pfdjAvdGltZW91dC9ydW4uZ28_": {
    "content": "package main\n\nimport (\n\t\"context\"\n\t\"fmt\"\n\t\"log\"\n\t\"time\"\n\n\t\"github.com/hatchet-dev/hatchet/pkg/client\"\n\t\"github.com/hatchet-dev/hatchet/pkg/worker\"\n)\n\nfunc run(done chan<- string, job worker.WorkflowJob) (func() error, error) {\n\tc, err := client.New()\n\tif err != nil {\n\t\treturn nil, fmt.Errorf(\"error creating client: %w\", err)\n\t}\n\n\tw, err := worker.NewWorker(\n\t\tworker.WithClient(\n\t\t\tc,\n\t\t),\n\t)\n\tif err != nil {\n\t\treturn nil, fmt.Errorf(\"error creating worker: %w\", err)\n\t}\n\n\terr = w.On(\n\t\tworker.Events(\"user:create:timeout\"),\n\t\t&job,\n\t)\n\tif err != nil {\n\t\treturn nil, fmt.Errorf(\"error registering workflow: %w\", err)\n\t}\n\n\tgo func() {\n\t\tlog.Printf(\"pushing event\")\n\n\t\ttestEvent := userCreateEvent{\n\t\t\tUsername: \"echo-test\",\n\t\t\tUserID:   \"1234\",\n\t\t\tData: map[string]string{\n\t\t\t\t\"test\": \"test\",\n\t\t\t},\n\t\t}\n\n\t\t// push an event\n\t\terr := c.Event().Push(\n\t\t\tcontext.Background(),\n\t\t\t\"user:create:timeout\",\n\t\t\ttestEvent,\n\t\t)\n\t\tif err != nil {\n\t\t\tpanic(fmt.Errorf(\"error pushing event: %w\", err))\n\t\t}\n\n\t\ttime.Sleep(20 * time.Second)\n\n\t\tdone <- \"done\"\n\t}()\n\n\tcleanup, err := w.Start()\n\tif err != nil {\n\t\treturn nil, fmt.Errorf(\"error starting worker: %w\", err)\n\t}\n\n\treturn cleanup, nil\n}\n",
    "language": "go",
    "source": "/Users/gabrielruttner/dev/hatchet/examples/go/z_v0/timeout/run.go"
  },
  "L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL2dvL3pfdjAvd2ViaG9vay9ydW4uZ28_": {
    "content": "package main\n\nimport (\n\t\"context\"\n\t\"errors\"\n\t\"fmt\"\n\t\"log\"\n\t\"net/http\"\n\t\"time\"\n\n\t\"github.com/hatchet-dev/hatchet/pkg/client\"\n\t\"github.com/hatchet-dev/hatchet/pkg/worker\"\n)\n\nfunc run(\n\tname string,\n\tw *worker.Worker,\n\tport string,\n\thandler func(w http.ResponseWriter, r *http.Request), c client.Client, workflow string, event string,\n) error {\n\t// create webserver to handle webhook requests\n\tmux := http.NewServeMux()\n\n\t// Register the HelloHandler to the /hello route\n\tmux.HandleFunc(\"/webhook\", handler)\n\n\t// Create a custom server\n\tserver := &http.Server{\n\t\tAddr:         \":\" + port,\n\t\tHandler:      mux,\n\t\tReadTimeout:  10 * time.Second,\n\t\tWriteTimeout: 10 * time.Second,\n\t\tIdleTimeout:  15 * time.Second,\n\t}\n\n\tdefer func(server *http.Server, ctx context.Context) {\n\t\terr := server.Shutdown(ctx)\n\t\tif err != nil {\n\t\t\tpanic(err)\n\t\t}\n\t}(server, context.Background())\n\n\tgo func() {\n\t\tif err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {\n\t\t\tpanic(err)\n\t\t}\n\t}()\n\n\tsecret := \"secret\"\n\tif err := w.RegisterWebhook(worker.RegisterWebhookWorkerOpts{\n\t\tName:   \"test-\" + name,\n\t\tURL:    fmt.Sprintf(\"http://localhost:%s/webhook\", port),\n\t\tSecret: &secret,\n\t}); err != nil {\n\t\treturn fmt.Errorf(\"error setting up webhook: %w\", err)\n\t}\n\n\ttime.Sleep(30 * time.Second)\n\n\tlog.Printf(\"pushing event\")\n\n\ttestEvent := userCreateEvent{\n\t\tUsername: \"echo-test\",\n\t\tUserID:   \"1234\",\n\t\tData: map[string]string{\n\t\t\t\"test\": \"test\",\n\t\t},\n\t}\n\n\t// push an event\n\terr := c.Event().Push(\n\t\tcontext.Background(),\n\t\tevent,\n\t\ttestEvent,\n\t)\n\tif err != nil {\n\t\treturn fmt.Errorf(\"error pushing event: %w\", err)\n\t}\n\n\ttime.Sleep(5 * time.Second)\n\n\treturn nil\n}\n",
    "language": "go",
    "source": "/Users/gabrielruttner/dev/hatchet/examples/go/z_v0/webhook/run.go"
  }
} as const;

// Snippet mapping
const snips = {
  "typescript": {
    "cancellations": {
      "run": {
        "*": ":L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3R5cGVzY3JpcHQvY2FuY2VsbGF0aW9ucy9ydW4udHM_",
        "Running a Task with Results": "Running a Task with Results:L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3R5cGVzY3JpcHQvY2FuY2VsbGF0aW9ucy9ydW4udHM_"
      },
      "worker": {
        "*": ":L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3R5cGVzY3JpcHQvY2FuY2VsbGF0aW9ucy93b3JrZXIudHM_",
        "Declaring a Worker": "Declaring a Worker:L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3R5cGVzY3JpcHQvY2FuY2VsbGF0aW9ucy93b3JrZXIudHM_"
      },
      "workflow": {
        "*": ":L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3R5cGVzY3JpcHQvY2FuY2VsbGF0aW9ucy93b3JrZmxvdy50cw__",
        "Declaring a Task": "Declaring a Task:L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3R5cGVzY3JpcHQvY2FuY2VsbGF0aW9ucy93b3JrZmxvdy50cw__",
        "Abort Signal": "Abort Signal:L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3R5cGVzY3JpcHQvY2FuY2VsbGF0aW9ucy93b3JrZmxvdy50cw__"
      }
    },
    "child_workflows": {
      "run": {
        "*": ":L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3R5cGVzY3JpcHQvY2hpbGRfd29ya2Zsb3dzL3J1bi50cw__"
      },
      "worker": {
        "*": ":L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3R5cGVzY3JpcHQvY2hpbGRfd29ya2Zsb3dzL3dvcmtlci50cw__"
      },
      "workflow": {
        "*": ":L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3R5cGVzY3JpcHQvY2hpbGRfd29ya2Zsb3dzL3dvcmtmbG93LnRz",
        "Declaring a Child": "Declaring a Child:L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3R5cGVzY3JpcHQvY2hpbGRfd29ya2Zsb3dzL3dvcmtmbG93LnRz",
        "Declaring a Parent": "Declaring a Parent:L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3R5cGVzY3JpcHQvY2hpbGRfd29ya2Zsb3dzL3dvcmtmbG93LnRz"
      }
    },
    "concurrency-rr": {
      "run": {
        "*": ":L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3R5cGVzY3JpcHQvY29uY3VycmVuY3ktcnIvcnVuLnRz"
      },
      "worker": {
        "*": ":L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3R5cGVzY3JpcHQvY29uY3VycmVuY3ktcnIvd29ya2VyLnRz"
      },
      "workflow": {
        "*": ":L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3R5cGVzY3JpcHQvY29uY3VycmVuY3ktcnIvd29ya2Zsb3cudHM_",
        "Concurrency Strategy With Key": "Concurrency Strategy With Key:L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3R5cGVzY3JpcHQvY29uY3VycmVuY3ktcnIvd29ya2Zsb3cudHM_",
        "Multiple Concurrency Keys": "Multiple Concurrency Keys:L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3R5cGVzY3JpcHQvY29uY3VycmVuY3ktcnIvd29ya2Zsb3cudHM_"
      }
    },
    "dag": {
      "run": {
        "*": ":L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3R5cGVzY3JpcHQvZGFnL3J1bi50cw__"
      },
      "worker": {
        "*": ":L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3R5cGVzY3JpcHQvZGFnL3dvcmtlci50cw__"
      },
      "workflow": {
        "*": ":L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3R5cGVzY3JpcHQvZGFnL3dvcmtmbG93LnRz",
        "Declaring a DAG Workflow": "Declaring a DAG Workflow:L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3R5cGVzY3JpcHQvZGFnL3dvcmtmbG93LnRz"
      }
    },
    "dag_match_condition": {
      "run": {
        "*": ":L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3R5cGVzY3JpcHQvZGFnX21hdGNoX2NvbmRpdGlvbi9ydW4udHM_"
      },
      "worker": {
        "*": ":L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3R5cGVzY3JpcHQvZGFnX21hdGNoX2NvbmRpdGlvbi93b3JrZXIudHM_"
      },
      "workflow": {
        "*": ":L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3R5cGVzY3JpcHQvZGFnX21hdGNoX2NvbmRpdGlvbi93b3JrZmxvdy50cw__"
      }
    },
    "deep": {
      "run": {
        "*": ":L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3R5cGVzY3JpcHQvZGVlcC9ydW4udHM_"
      },
      "worker": {
        "*": ":L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3R5cGVzY3JpcHQvZGVlcC93b3JrZXIudHM_"
      },
      "workflow": {
        "*": ":L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3R5cGVzY3JpcHQvZGVlcC93b3JrZmxvdy50cw__"
      }
    },
    "durable-event": {
      "run": {
        "*": ":L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3R5cGVzY3JpcHQvZHVyYWJsZS1ldmVudC9ydW4udHM_"
      },
      "worker": {
        "*": ":L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3R5cGVzY3JpcHQvZHVyYWJsZS1ldmVudC93b3JrZXIudHM_"
      },
      "workflow": {
        "*": ":L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3R5cGVzY3JpcHQvZHVyYWJsZS1ldmVudC93b3JrZmxvdy50cw__",
        "Durable Event": "Durable Event:L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3R5cGVzY3JpcHQvZHVyYWJsZS1ldmVudC93b3JrZmxvdy50cw__",
        "Durable Event With Filter": "Durable Event With Filter:L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3R5cGVzY3JpcHQvZHVyYWJsZS1ldmVudC93b3JrZmxvdy50cw__"
      }
    },
    "durable-sleep": {
      "run": {
        "*": ":L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3R5cGVzY3JpcHQvZHVyYWJsZS1zbGVlcC9ydW4udHM_"
      },
      "worker": {
        "*": ":L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3R5cGVzY3JpcHQvZHVyYWJsZS1zbGVlcC93b3JrZXIudHM_"
      },
      "workflow": {
        "*": ":L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3R5cGVzY3JpcHQvZHVyYWJsZS1zbGVlcC93b3JrZmxvdy50cw__",
        "Durable Sleep": "Durable Sleep:L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3R5cGVzY3JpcHQvZHVyYWJsZS1zbGVlcC93b3JrZmxvdy50cw__"
      }
    },
    "inferred-typing": {
      "run": {
        "*": ":L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3R5cGVzY3JpcHQvaW5mZXJyZWQtdHlwaW5nL3J1bi50cw__"
      },
      "worker": {
        "*": ":L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3R5cGVzY3JpcHQvaW5mZXJyZWQtdHlwaW5nL3dvcmtlci50cw__"
      },
      "workflow": {
        "*": ":L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3R5cGVzY3JpcHQvaW5mZXJyZWQtdHlwaW5nL3dvcmtmbG93LnRz"
      }
    },
    "legacy": {
      "run": {
        "*": ":L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3R5cGVzY3JpcHQvbGVnYWN5L3J1bi50cw__"
      },
      "worker": {
        "*": ":L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3R5cGVzY3JpcHQvbGVnYWN5L3dvcmtlci50cw__"
      },
      "workflow": {
        "*": ":L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3R5cGVzY3JpcHQvbGVnYWN5L3dvcmtmbG93LnRz"
      }
    },
    "multiple_wf_concurrency": {
      "run": {
        "*": ":L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3R5cGVzY3JpcHQvbXVsdGlwbGVfd2ZfY29uY3VycmVuY3kvcnVuLnRz"
      },
      "worker": {
        "*": ":L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3R5cGVzY3JpcHQvbXVsdGlwbGVfd2ZfY29uY3VycmVuY3kvd29ya2VyLnRz"
      },
      "workflow": {
        "*": ":L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3R5cGVzY3JpcHQvbXVsdGlwbGVfd2ZfY29uY3VycmVuY3kvd29ya2Zsb3cudHM_",
        "Concurrency Strategy With Key": "Concurrency Strategy With Key:L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3R5cGVzY3JpcHQvbXVsdGlwbGVfd2ZfY29uY3VycmVuY3kvd29ya2Zsb3cudHM_"
      }
    },
    "non_retryable": {
      "run": {
        "*": ":L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3R5cGVzY3JpcHQvbm9uX3JldHJ5YWJsZS9ydW4udHM_"
      },
      "worker": {
        "*": ":L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3R5cGVzY3JpcHQvbm9uX3JldHJ5YWJsZS93b3JrZXIudHM_"
      },
      "workflow": {
        "*": ":L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3R5cGVzY3JpcHQvbm9uX3JldHJ5YWJsZS93b3JrZmxvdy50cw__",
        "Non-retrying task": "Non-retrying task:L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3R5cGVzY3JpcHQvbm9uX3JldHJ5YWJsZS93b3JrZmxvdy50cw__"
      }
    },
    "on_cron": {
      "worker": {
        "*": ":L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3R5cGVzY3JpcHQvb25fY3Jvbi93b3JrZXIudHM_"
      },
      "workflow": {
        "*": ":L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3R5cGVzY3JpcHQvb25fY3Jvbi93b3JrZmxvdy50cw__",
        "Run Workflow on Cron": "Run Workflow on Cron:L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3R5cGVzY3JpcHQvb25fY3Jvbi93b3JrZmxvdy50cw__"
      }
    },
    "on_event": {
      "worker": {
        "*": ":L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3R5cGVzY3JpcHQvb25fZXZlbnQvd29ya2VyLnRz"
      },
      "workflow": {
        "*": ":L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3R5cGVzY3JpcHQvb25fZXZlbnQvd29ya2Zsb3cudHM_",
        "Run workflow on event": "Run workflow on event:L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3R5cGVzY3JpcHQvb25fZXZlbnQvd29ya2Zsb3cudHM_"
      }
    },
    "on_event copy": {
      "worker": {
        "*": ":L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3R5cGVzY3JpcHQvb25fZXZlbnQgY29weS93b3JrZXIudHM_"
      },
      "workflow": {
        "*": ":L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3R5cGVzY3JpcHQvb25fZXZlbnQgY29weS93b3JrZmxvdy50cw__",
        "Run workflow on event": "Run workflow on event:L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3R5cGVzY3JpcHQvb25fZXZlbnQgY29weS93b3JrZmxvdy50cw__"
      }
    },
    "on_failure": {
      "run": {
        "*": ":L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3R5cGVzY3JpcHQvb25fZmFpbHVyZS9ydW4udHM_"
      },
      "worker": {
        "*": ":L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3R5cGVzY3JpcHQvb25fZmFpbHVyZS93b3JrZXIudHM_"
      },
      "workflow": {
        "*": ":L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3R5cGVzY3JpcHQvb25fZmFpbHVyZS93b3JrZmxvdy50cw__",
        "On Failure Task": "On Failure Task:L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3R5cGVzY3JpcHQvb25fZmFpbHVyZS93b3JrZmxvdy50cw__"
      }
    },
    "on_success": {
      "run": {
        "*": ":L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3R5cGVzY3JpcHQvb25fc3VjY2Vzcy9ydW4udHM_"
      },
      "worker": {
        "*": ":L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3R5cGVzY3JpcHQvb25fc3VjY2Vzcy93b3JrZXIudHM_"
      },
      "workflow": {
        "*": ":L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3R5cGVzY3JpcHQvb25fc3VjY2Vzcy93b3JrZmxvdy50cw__",
        "On Success DAG": "On Success DAG:L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3R5cGVzY3JpcHQvb25fc3VjY2Vzcy93b3JrZmxvdy50cw__"
      }
    },
    "priority": {
      "run": {
        "*": ":L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3R5cGVzY3JpcHQvcHJpb3JpdHkvcnVuLnRz",
        "Run a Task with a Priority": "Run a Task with a Priority:L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3R5cGVzY3JpcHQvcHJpb3JpdHkvcnVuLnRz",
        "Schedule and cron": "Schedule and cron:L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3R5cGVzY3JpcHQvcHJpb3JpdHkvcnVuLnRz"
      },
      "worker": {
        "*": ":L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3R5cGVzY3JpcHQvcHJpb3JpdHkvd29ya2VyLnRz"
      },
      "workflow": {
        "*": ":L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3R5cGVzY3JpcHQvcHJpb3JpdHkvd29ya2Zsb3cudHM_",
        "Simple Task Priority": "Simple Task Priority:L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3R5cGVzY3JpcHQvcHJpb3JpdHkvd29ya2Zsb3cudHM_",
        "Task Priority in a Workflow": "Task Priority in a Workflow:L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3R5cGVzY3JpcHQvcHJpb3JpdHkvd29ya2Zsb3cudHM_"
      }
    },
    "rate_limit": {
      "workflow": {
        "*": ":L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3R5cGVzY3JpcHQvcmF0ZV9saW1pdC93b3JrZmxvdy50cw__",
        "Upsert Rate Limit": "Upsert Rate Limit:L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3R5cGVzY3JpcHQvcmF0ZV9saW1pdC93b3JrZmxvdy50cw__",
        "Static": "Static:L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3R5cGVzY3JpcHQvcmF0ZV9saW1pdC93b3JrZmxvdy50cw__",
        "Dynamic": "Dynamic:L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3R5cGVzY3JpcHQvcmF0ZV9saW1pdC93b3JrZmxvdy50cw__"
      }
    },
    "retries": {
      "run": {
        "*": ":L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3R5cGVzY3JpcHQvcmV0cmllcy9ydW4udHM_"
      },
      "worker": {
        "*": ":L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3R5cGVzY3JpcHQvcmV0cmllcy93b3JrZXIudHM_"
      },
      "workflow": {
        "*": ":L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3R5cGVzY3JpcHQvcmV0cmllcy93b3JrZmxvdy50cw__",
        "Simple Step Retries": "Simple Step Retries:L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3R5cGVzY3JpcHQvcmV0cmllcy93b3JrZmxvdy50cw__",
        "Retries with Count": "Retries with Count:L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3R5cGVzY3JpcHQvcmV0cmllcy93b3JrZmxvdy50cw__",
        "Get the current retry count": "Get the current retry count:L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3R5cGVzY3JpcHQvcmV0cmllcy93b3JrZmxvdy50cw__",
        "Retries with Backoff": "Retries with Backoff:L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3R5cGVzY3JpcHQvcmV0cmllcy93b3JrZmxvdy50cw__"
      }
    },
    "simple": {
      "run": {
        "*": ":L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3R5cGVzY3JpcHQvc2ltcGxlL3J1bi50cw__",
        "Running a Task": "Running a Task:L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3R5cGVzY3JpcHQvc2ltcGxlL3J1bi50cw__",
        "Running Multiple Tasks": "Running Multiple Tasks:L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3R5cGVzY3JpcHQvc2ltcGxlL3J1bi50cw__",
        "Spawning Tasks from within a Task": "Spawning Tasks from within a Task:L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3R5cGVzY3JpcHQvc2ltcGxlL3J1bi50cw__"
      },
      "worker": {
        "*": ":L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3R5cGVzY3JpcHQvc2ltcGxlL3dvcmtlci50cw__",
        "Declaring a Worker": "Declaring a Worker:L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3R5cGVzY3JpcHQvc2ltcGxlL3dvcmtlci50cw__"
      },
      "workflow": {
        "*": ":L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3R5cGVzY3JpcHQvc2ltcGxlL3dvcmtmbG93LnRz",
        "Declaring a Task": "Declaring a Task:L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3R5cGVzY3JpcHQvc2ltcGxlL3dvcmtmbG93LnRz"
      }
    },
    "sticky": {
      "run": {
        "*": ":L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3R5cGVzY3JpcHQvc3RpY2t5L3J1bi50cw__"
      },
      "worker": {
        "*": ":L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3R5cGVzY3JpcHQvc3RpY2t5L3dvcmtlci50cw__"
      },
      "workflow": {
        "*": ":L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3R5cGVzY3JpcHQvc3RpY2t5L3dvcmtmbG93LnRz",
        "Sticky Task": "Sticky Task:L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3R5cGVzY3JpcHQvc3RpY2t5L3dvcmtmbG93LnRz"
      }
    },
    "timeouts": {
      "run": {
        "*": ":L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3R5cGVzY3JpcHQvdGltZW91dHMvcnVuLnRz",
        "Running a Task with Results": "Running a Task with Results:L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3R5cGVzY3JpcHQvdGltZW91dHMvcnVuLnRz"
      },
      "worker": {
        "*": ":L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3R5cGVzY3JpcHQvdGltZW91dHMvd29ya2VyLnRz",
        "Declaring a Worker": "Declaring a Worker:L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3R5cGVzY3JpcHQvdGltZW91dHMvd29ya2VyLnRz"
      },
      "workflow": {
        "*": ":L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3R5cGVzY3JpcHQvdGltZW91dHMvd29ya2Zsb3cudHM_",
        "Declaring a Task": "Declaring a Task:L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3R5cGVzY3JpcHQvdGltZW91dHMvd29ya2Zsb3cudHM_"
      }
    },
    "with_timeouts": {
      "workflow": {
        "*": ":L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3R5cGVzY3JpcHQvd2l0aF90aW1lb3V0cy93b3JrZmxvdy50cw__",
        "Declaring a Task": "Declaring a Task:L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3R5cGVzY3JpcHQvd2l0aF90aW1lb3V0cy93b3JrZmxvdy50cw__",
        "Execution Timeout": "Execution Timeout:L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3R5cGVzY3JpcHQvd2l0aF90aW1lb3V0cy93b3JrZmxvdy50cw__",
        "Refresh Timeout": "Refresh Timeout:L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3R5cGVzY3JpcHQvd2l0aF90aW1lb3V0cy93b3JrZmxvdy50cw__"
      }
    }
  },
  "python": {
    "affinity_workers": {
      "worker": {
        "*": ":L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3B5dGhvbi9hZmZpbml0eV93b3JrZXJzL3dvcmtlci5weQ__",
        "AffinityWorkflow": "AffinityWorkflow:L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3B5dGhvbi9hZmZpbml0eV93b3JrZXJzL3dvcmtlci5weQ__",
        "AffinityTask": "AffinityTask:L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3B5dGhvbi9hZmZpbml0eV93b3JrZXJzL3dvcmtlci5weQ__",
        "AffinityWorker": "AffinityWorker:L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3B5dGhvbi9hZmZpbml0eV93b3JrZXJzL3dvcmtlci5weQ__"
      }
    },
    "blocked_async": {
      "worker": {
        "*": ":L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3B5dGhvbi9ibG9ja2VkX2FzeW5jL3dvcmtlci5weQ__"
      }
    },
    "bulk_fanout": {
      "worker": {
        "*": ":L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3B5dGhvbi9idWxrX2Zhbm91dC93b3JrZXIucHk_",
        "BulkFanoutParent": "BulkFanoutParent:L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3B5dGhvbi9idWxrX2Zhbm91dC93b3JrZXIucHk_"
      }
    },
    "cancellation": {
      "worker": {
        "*": ":L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3B5dGhvbi9jYW5jZWxsYXRpb24vd29ya2VyLnB5",
        "Self-cancelling task": "Self-cancelling task:L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3B5dGhvbi9jYW5jZWxsYXRpb24vd29ya2VyLnB5",
        "Checking exit flag": "Checking exit flag:L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3B5dGhvbi9jYW5jZWxsYXRpb24vd29ya2VyLnB5"
      }
    },
    "child": {
      "worker": {
        "*": ":L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3B5dGhvbi9jaGlsZC93b3JrZXIucHk_",
        "Simple": "Simple:L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3B5dGhvbi9jaGlsZC93b3JrZXIucHk_"
      }
    },
    "concurrency_limit": {
      "worker": {
        "*": ":L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3B5dGhvbi9jb25jdXJyZW5jeV9saW1pdC93b3JrZXIucHk_",
        "Workflow": "Workflow:L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3B5dGhvbi9jb25jdXJyZW5jeV9saW1pdC93b3JrZXIucHk_"
      }
    },
    "concurrency_limit_rr": {
      "worker": {
        "*": ":L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3B5dGhvbi9jb25jdXJyZW5jeV9saW1pdF9yci93b3JrZXIucHk_",
        "Concurrency Strategy With Key": "Concurrency Strategy With Key:L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3B5dGhvbi9jb25jdXJyZW5jeV9saW1pdF9yci93b3JrZXIucHk_"
      }
    },
    "concurrency_limit_rr_load": {
      "worker": {
        "*": ":L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3B5dGhvbi9jb25jdXJyZW5jeV9saW1pdF9ycl9sb2FkL3dvcmtlci5weQ__"
      }
    },
    "concurrency_multiple_keys": {
      "worker": {
        "*": ":L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3B5dGhvbi9jb25jdXJyZW5jeV9tdWx0aXBsZV9rZXlzL3dvcmtlci5weQ__",
        "Concurrency Strategy With Key": "Concurrency Strategy With Key:L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3B5dGhvbi9jb25jdXJyZW5jeV9tdWx0aXBsZV9rZXlzL3dvcmtlci5weQ__"
      }
    },
    "concurrency_workflow_level": {
      "worker": {
        "*": ":L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3B5dGhvbi9jb25jdXJyZW5jeV93b3JrZmxvd19sZXZlbC93b3JrZXIucHk_",
        "Multiple Concurrency Keys": "Multiple Concurrency Keys:L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3B5dGhvbi9jb25jdXJyZW5jeV93b3JrZmxvd19sZXZlbC93b3JrZXIucHk_"
      }
    },
    "dag": {
      "worker": {
        "*": ":L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3B5dGhvbi9kYWcvd29ya2VyLnB5"
      }
    },
    "dedupe": {
      "worker": {
        "*": ":L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3B5dGhvbi9kZWR1cGUvd29ya2VyLnB5"
      }
    },
    "delayed": {
      "worker": {
        "*": ":L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3B5dGhvbi9kZWxheWVkL3dvcmtlci5weQ__"
      }
    },
    "durable": {
      "worker": {
        "*": ":L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3B5dGhvbi9kdXJhYmxlL3dvcmtlci5weQ__",
        "Create a durable workflow": "Create a durable workflow:L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3B5dGhvbi9kdXJhYmxlL3dvcmtlci5weQ__",
        "Add durable task": "Add durable task:L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3B5dGhvbi9kdXJhYmxlL3dvcmtlci5weQ__"
      }
    },
    "durable_event": {
      "worker": {
        "*": ":L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3B5dGhvbi9kdXJhYmxlX2V2ZW50L3dvcmtlci5weQ__",
        "Durable Event": "Durable Event:L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3B5dGhvbi9kdXJhYmxlX2V2ZW50L3dvcmtlci5weQ__",
        "Durable Event With Filter": "Durable Event With Filter:L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3B5dGhvbi9kdXJhYmxlX2V2ZW50L3dvcmtlci5weQ__"
      }
    },
    "durable_sleep": {
      "worker": {
        "*": ":L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3B5dGhvbi9kdXJhYmxlX3NsZWVwL3dvcmtlci5weQ__",
        "Durable Sleep": "Durable Sleep:L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3B5dGhvbi9kdXJhYmxlX3NsZWVwL3dvcmtlci5weQ__"
      }
    },
    "events": {
      "worker": {
        "*": ":L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3B5dGhvbi9ldmVudHMvd29ya2VyLnB5",
        "Event trigger": "Event trigger:L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3B5dGhvbi9ldmVudHMvd29ya2VyLnB5"
      }
    },
    "fanout": {
      "worker": {
        "*": ":L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3B5dGhvbi9mYW5vdXQvd29ya2VyLnB5",
        "FanoutParent": "FanoutParent:L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3B5dGhvbi9mYW5vdXQvd29ya2VyLnB5",
        "FanoutChild": "FanoutChild:L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3B5dGhvbi9mYW5vdXQvd29ya2VyLnB5"
      }
    },
    "fanout_sync": {
      "worker": {
        "*": ":L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3B5dGhvbi9mYW5vdXRfc3luYy93b3JrZXIucHk_"
      }
    },
    "lifespans": {
      "worker": {
        "*": ":L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3B5dGhvbi9saWZlc3BhbnMvd29ya2VyLnB5",
        "Use the lifespan in a task": "Use the lifespan in a task:L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3B5dGhvbi9saWZlc3BhbnMvd29ya2VyLnB5",
        "Define a lifespan": "Define a lifespan:L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3B5dGhvbi9saWZlc3BhbnMvd29ya2VyLnB5"
      }
    },
    "logger": {
      "worker": {
        "*": ":L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3B5dGhvbi9sb2dnZXIvd29ya2VyLnB5"
      },
      "workflow": {
        "*": ":L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3B5dGhvbi9sb2dnZXIvd29ya2Zsb3cucHk_",
        "LoggingWorkflow": "LoggingWorkflow:L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3B5dGhvbi9sb2dnZXIvd29ya2Zsb3cucHk_",
        "ContextLogger": "ContextLogger:L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3B5dGhvbi9sb2dnZXIvd29ya2Zsb3cucHk_"
      }
    },
    "manual_slot_release": {
      "worker": {
        "*": ":L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3B5dGhvbi9tYW51YWxfc2xvdF9yZWxlYXNlL3dvcmtlci5weQ__",
        "SlotRelease": "SlotRelease:L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3B5dGhvbi9tYW51YWxfc2xvdF9yZWxlYXNlL3dvcmtlci5weQ__"
      }
    },
    "non_retryable": {
      "worker": {
        "*": ":L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3B5dGhvbi9ub25fcmV0cnlhYmxlL3dvcmtlci5weQ__",
        "Non-retryable task": "Non-retryable task:L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3B5dGhvbi9ub25fcmV0cnlhYmxlL3dvcmtlci5weQ__"
      }
    },
    "on_failure": {
      "worker": {
        "*": ":L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3B5dGhvbi9vbl9mYWlsdXJlL3dvcmtlci5weQ__",
        "OnFailure Step": "OnFailure Step:L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3B5dGhvbi9vbl9mYWlsdXJlL3dvcmtlci5weQ__",
        "OnFailure With Details": "OnFailure With Details:L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3B5dGhvbi9vbl9mYWlsdXJlL3dvcmtlci5weQ__"
      }
    },
    "on_success": {
      "worker": {
        "*": ":L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3B5dGhvbi9vbl9zdWNjZXNzL3dvcmtlci5weQ__"
      }
    },
    "opentelemetry_instrumentation": {
      "worker": {
        "*": ":L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3B5dGhvbi9vcGVudGVsZW1ldHJ5X2luc3RydW1lbnRhdGlvbi93b3JrZXIucHk_"
      }
    },
    "priority": {
      "worker": {
        "*": ":L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3B5dGhvbi9wcmlvcml0eS93b3JrZXIucHk_",
        "Default priority": "Default priority:L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3B5dGhvbi9wcmlvcml0eS93b3JrZXIucHk_"
      }
    },
    "rate_limit": {
      "worker": {
        "*": ":L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3B5dGhvbi9yYXRlX2xpbWl0L3dvcmtlci5weQ__",
        "Workflow": "Workflow:L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3B5dGhvbi9yYXRlX2xpbWl0L3dvcmtlci5weQ__",
        "Static": "Static:L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3B5dGhvbi9yYXRlX2xpbWl0L3dvcmtlci5weQ__",
        "Dynamic": "Dynamic:L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3B5dGhvbi9yYXRlX2xpbWl0L3dvcmtlci5weQ__"
      }
    },
    "retries": {
      "worker": {
        "*": ":L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3B5dGhvbi9yZXRyaWVzL3dvcmtlci5weQ__",
        "Simple Step Retries": "Simple Step Retries:L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3B5dGhvbi9yZXRyaWVzL3dvcmtlci5weQ__",
        "Retries with Count": "Retries with Count:L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3B5dGhvbi9yZXRyaWVzL3dvcmtlci5weQ__",
        "Retries with Backoff": "Retries with Backoff:L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3B5dGhvbi9yZXRyaWVzL3dvcmtlci5weQ__"
      }
    },
    "simple": {
      "worker": {
        "*": ":L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3B5dGhvbi9zaW1wbGUvd29ya2VyLnB5",
        "Simple": "Simple:L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3B5dGhvbi9zaW1wbGUvd29ya2VyLnB5"
      }
    },
    "sticky_workers": {
      "worker": {
        "*": ":L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3B5dGhvbi9zdGlja3lfd29ya2Vycy93b3JrZXIucHk_",
        "StickyWorker": "StickyWorker:L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3B5dGhvbi9zdGlja3lfd29ya2Vycy93b3JrZXIucHk_",
        "StickyChild": "StickyChild:L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3B5dGhvbi9zdGlja3lfd29ya2Vycy93b3JrZXIucHk_"
      }
    },
    "streaming": {
      "worker": {
        "*": ":L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3B5dGhvbi9zdHJlYW1pbmcvd29ya2VyLnB5",
        "Streaming": "Streaming:L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3B5dGhvbi9zdHJlYW1pbmcvd29ya2VyLnB5"
      }
    },
    "timeout": {
      "worker": {
        "*": ":L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3B5dGhvbi90aW1lb3V0L3dvcmtlci5weQ__",
        "ScheduleTimeout": "ScheduleTimeout:L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3B5dGhvbi90aW1lb3V0L3dvcmtlci5weQ__",
        "ExecutionTimeout": "ExecutionTimeout:L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3B5dGhvbi90aW1lb3V0L3dvcmtlci5weQ__",
        "RefreshTimeout": "RefreshTimeout:L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3B5dGhvbi90aW1lb3V0L3dvcmtlci5weQ__"
      }
    },
    "waits": {
      "worker": {
        "*": ":L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3B5dGhvbi93YWl0cy93b3JrZXIucHk_",
        "Create a workflow": "Create a workflow:L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3B5dGhvbi93YWl0cy93b3JrZXIucHk_",
        "Add base task": "Add base task:L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3B5dGhvbi93YWl0cy93b3JrZXIucHk_",
        "Add wait for sleep": "Add wait for sleep:L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3B5dGhvbi93YWl0cy93b3JrZXIucHk_",
        "Add skip on event": "Add skip on event:L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3B5dGhvbi93YWl0cy93b3JrZXIucHk_",
        "Add branching": "Add branching:L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3B5dGhvbi93YWl0cy93b3JrZXIucHk_",
        "Add wait for event": "Add wait for event:L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3B5dGhvbi93YWl0cy93b3JrZXIucHk_",
        "Add sum": "Add sum:L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3B5dGhvbi93YWl0cy93b3JrZXIucHk_"
      }
    },
    "worker": {
      "*": ":L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3B5dGhvbi93b3JrZXIucHk_"
    },
    "worker_existing_loop": {
      "worker": {
        "*": ":L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3B5dGhvbi93b3JrZXJfZXhpc3RpbmdfbG9vcC93b3JrZXIucHk_"
      }
    },
    "workflow_registration": {
      "worker": {
        "*": ":L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3B5dGhvbi93b3JrZmxvd19yZWdpc3RyYXRpb24vd29ya2VyLnB5",
        "WorkflowRegistration": "WorkflowRegistration:L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL3B5dGhvbi93b3JrZmxvd19yZWdpc3RyYXRpb24vd29ya2VyLnB5"
      }
    }
  },
  "go": {
    "z_v0": {
      "assignment-affinity": {
        "run": {
          "*": ":L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL2dvL3pfdjAvYXNzaWdubWVudC1hZmZpbml0eS9ydW4uZ28_"
        }
      },
      "assignment-sticky": {
        "run": {
          "*": ":L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL2dvL3pfdjAvYXNzaWdubWVudC1zdGlja3kvcnVuLmdv",
          "StickyWorker": "StickyWorker:L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL2dvL3pfdjAvYXNzaWdubWVudC1zdGlja3kvcnVuLmdv",
          "StickyChild": "StickyChild:L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL2dvL3pfdjAvYXNzaWdubWVudC1zdGlja3kvcnVuLmdv"
        }
      },
      "bulk_workflows": {
        "run": {
          "*": ":L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL2dvL3pfdjAvYnVsa193b3JrZmxvd3MvcnVuLmdv"
        }
      },
      "cancellation": {
        "run": {
          "*": ":L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL2dvL3pfdjAvY2FuY2VsbGF0aW9uL3J1bi5nbw__"
        }
      },
      "loadtest": {
        "cli": {
          "run": {
            "*": ":L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL2dvL3pfdjAvbG9hZHRlc3QvY2xpL3J1bi5nbw__"
          }
        },
        "rampup": {
          "run": {
            "*": ":L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL2dvL3pfdjAvbG9hZHRlc3QvcmFtcHVwL3J1bi5nbw__"
          }
        }
      },
      "middleware": {
        "run": {
          "*": ":L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL2dvL3pfdjAvbWlkZGxld2FyZS9ydW4uZ28_"
        }
      },
      "timeout": {
        "run": {
          "*": ":L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL2dvL3pfdjAvdGltZW91dC9ydW4uZ28_"
        }
      },
      "webhook": {
        "run": {
          "*": ":L1VzZXJzL2dhYnJpZWxydXR0bmVyL2Rldi9oYXRjaGV0L2V4YW1wbGVzL2dvL3pfdjAvd2ViaG9vay9ydW4uZ28_"
        }
      }
    }
  }
} as const;

export default snips;
