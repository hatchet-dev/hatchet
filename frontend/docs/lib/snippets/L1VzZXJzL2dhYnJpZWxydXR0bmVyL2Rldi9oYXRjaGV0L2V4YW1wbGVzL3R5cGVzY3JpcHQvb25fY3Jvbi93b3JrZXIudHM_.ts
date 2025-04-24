// Generated from /Users/gabrielruttner/dev/hatchet/examples/typescript/on_cron/worker.ts
export const content = "import { hatchet } from '../hatchet-client';\nimport { onCron } from './workflow';\n\nasync function main() {\n  const worker = await hatchet.worker('on-cron-worker', {\n    workflows: [onCron],\n  });\n\n  await worker.start();\n}\n\nif (require.main === module) {\n  main();\n}\n";
export const language = "ts";
