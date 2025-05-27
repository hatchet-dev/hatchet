import { Snippet } from '@/next/lib/docs/generated/snips/types';

const snippet: Snippet = {
  language: 'typescript ',
  content:
    "import { hatchet } from '../hatchet-client';\nimport { SimpleOutput } from './stub-workflow';\n// > Enqueuing a Workflow (Fire and Forget)\nimport { simple } from './workflow';\n// ...\n\nasync function main() {\n  // 👀 Enqueue the workflow\n  const run = await simple.runNoWait({\n    Message: 'hello',\n  });\n\n  // 👀 Get the run ID of the workflow\n  const runId = await run.getWorkflowRunId();\n  // It may be helpful to store the run ID of the workflow\n  // in a database or other persistent storage for later use\n  console.log(runId);\n\n  // > Subscribing to results\n  // the return object of the enqueue method is a WorkflowRunRef which includes a listener for the result of the workflow\n  const result = await run.output;\n  console.log(result);\n\n  // if you need to subscribe to the result of the workflow at a later time, you can use the runRef method and the stored runId\n  const ref = hatchet.runRef<SimpleOutput>(runId);\n  const result2 = await ref.output;\n  console.log(result2);\n}\n\nif (require.main === module) {\n  main();\n}\n",
  source: 'out/typescript/simple/enqueue.ts',
  blocks: {
    enqueuing_a_workflow_fire_and_forget: {
      start: 4,
      stop: 17,
    },
    subscribing_to_results: {
      start: 20,
      stop: 27,
    },
  },
  highlights: {},
};

export default snippet;
