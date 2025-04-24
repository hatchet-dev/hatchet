// Generated from /Users/gabrielruttner/dev/hatchet/examples/typescript/on_success/worker.ts
export const content = "import { hatchet } from '../hatchet-client';\nimport { onSuccessDag } from './workflow';\n\nasync function main() {\n  const worker = await hatchet.worker('always-succeed-worker', {\n    workflows: [onSuccessDag],\n  });\n\n  await worker.start();\n}\n\nif (require.main === module) {\n  main();\n}\n";
export const language = "ts";
