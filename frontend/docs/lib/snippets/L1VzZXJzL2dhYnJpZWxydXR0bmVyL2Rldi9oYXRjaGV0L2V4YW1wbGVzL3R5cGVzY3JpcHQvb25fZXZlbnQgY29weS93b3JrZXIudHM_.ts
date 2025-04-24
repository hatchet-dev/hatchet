// Generated from /Users/gabrielruttner/dev/hatchet/examples/typescript/on_event copy/worker.ts
export const content = "import { hatchet } from '../hatchet-client';\nimport { lower, upper } from './workflow';\n\nasync function main() {\n  const worker = await hatchet.worker('on-event-worker', {\n    workflows: [lower, upper],\n  });\n\n  await worker.start();\n}\n\nif (require.main === module) {\n  main();\n}\n";
export const language = "ts";
