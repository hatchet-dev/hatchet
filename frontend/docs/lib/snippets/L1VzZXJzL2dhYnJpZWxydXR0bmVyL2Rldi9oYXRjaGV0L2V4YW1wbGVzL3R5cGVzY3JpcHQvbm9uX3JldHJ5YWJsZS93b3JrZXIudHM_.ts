// Generated from /Users/gabrielruttner/dev/hatchet/examples/typescript/non_retryable/worker.ts
export const content = "import { hatchet } from '../hatchet-client';\nimport { nonRetryableWorkflow } from './workflow';\n\nasync function main() {\n  const worker = await hatchet.worker('no-retry-worker', {\n    workflows: [nonRetryableWorkflow],\n  });\n\n  await worker.start();\n}\n\nif (require.main === module) {\n  main();\n}\n";
export const language = "ts";
