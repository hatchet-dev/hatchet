// Generated from /Users/gabrielruttner/dev/hatchet/examples/typescript/durable-event/worker.ts
export const content = "import { hatchet } from '../hatchet-client';\nimport { durableEvent } from './workflow';\n\nasync function main() {\n  const worker = await hatchet.worker('durable-event-worker', {\n    workflows: [durableEvent],\n  });\n\n  await worker.start();\n}\n\nif (require.main === module) {\n  main();\n}\n";
export const language = "ts";
