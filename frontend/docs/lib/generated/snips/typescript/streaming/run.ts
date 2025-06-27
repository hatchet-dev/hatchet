import { Snippet } from '@/lib/generated/snips/types';

const snippet: Snippet = {
  "language": "typescript ",
  "content": "import { streamingTask } from './workflow';\nimport { hatchet } from '../hatchet-client';\n\nasync function main() {\n  // > Consume\n  const ref = await streamingTask.runNoWait({});\n  const id = await ref.getWorkflowRunId();\n\n  for await (const content of hatchet.runs.subscribeToStream(id)) {\n    process.stdout.write(content);\n  }\n}\n\nif (require.main === module) {\n  main()\n    .catch(console.error)\n    .finally(() => {\n      process.exit(0);\n    });\n}\n",
  "source": "out/typescript/streaming/run.ts",
  "blocks": {
    "consume": {
      "start": 6,
      "stop": 11
    }
  },
  "highlights": {}
};

export default snippet;
