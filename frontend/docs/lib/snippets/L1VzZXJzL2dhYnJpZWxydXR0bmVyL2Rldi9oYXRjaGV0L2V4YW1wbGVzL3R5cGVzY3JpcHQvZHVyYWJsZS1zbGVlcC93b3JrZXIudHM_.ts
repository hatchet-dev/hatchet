// Generated from /Users/gabrielruttner/dev/hatchet/examples/typescript/durable-sleep/worker.ts
export const content = "import { hatchet } from '../hatchet-client';\nimport { durableSleep } from './workflow';\n\nasync function main() {\n  const worker = await hatchet.worker('sleep-worker', {\n    workflows: [durableSleep],\n  });\n\n  await worker.start();\n}\n\nif (require.main === module) {\n  main();\n}\n";
export const language = "ts";
