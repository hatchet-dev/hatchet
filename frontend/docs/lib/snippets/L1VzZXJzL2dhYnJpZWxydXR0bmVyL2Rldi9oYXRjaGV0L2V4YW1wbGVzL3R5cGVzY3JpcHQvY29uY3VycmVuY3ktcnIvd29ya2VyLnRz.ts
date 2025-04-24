// Generated from /Users/gabrielruttner/dev/hatchet/examples/typescript/concurrency-rr/worker.ts
export const content = "import { hatchet } from '../hatchet-client';\nimport { simpleConcurrency } from './workflow';\n\nasync function main() {\n  const worker = await hatchet.worker('simple-concurrency-worker', {\n    workflows: [simpleConcurrency],\n  });\n\n  await worker.start();\n}\n\nif (require.main === module) {\n  main();\n}\n";
export const language = "ts";
