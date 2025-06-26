import { Snippet } from '@/next/lib/docs/generated/snips/types';

const snippet: Snippet = {
  "language": "typescript ",
  "content": "import { RunEventType } from '@hatchet-dev/typescript-sdk-dev/typescript-sdk/clients/listeners/run-listener/child-listener-client';\nimport { streamingTask } from './workflow';\n\nasync function main() {\n  // > Consume\n  const ref = await streamingTask.runNoWait({});\n\n  const stream = await ref.stream();\n\n  for await (const event of stream) {\n    if (event.type === RunEventType.STEP_RUN_EVENT_TYPE_STREAM) {\n      process.stdout.write(event.payload);\n    }\n  }\n}\n\nif (require.main === module) {\n  main()\n    .catch(console.error)\n    .finally(() => {\n      process.exit(0);\n    });\n}\n",
  "source": "out/typescript/streaming/run.ts",
  "blocks": {
    "consume": {
      "start": 6,
      "stop": 14
    }
  },
  "highlights": {}
};

export default snippet;
