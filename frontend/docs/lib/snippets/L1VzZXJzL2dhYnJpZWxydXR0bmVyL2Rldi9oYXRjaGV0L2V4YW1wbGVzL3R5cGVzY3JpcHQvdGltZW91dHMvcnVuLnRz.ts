// Generated from /Users/gabrielruttner/dev/hatchet/examples/typescript/timeouts/run.ts
export const content = "/* eslint-disable no-console */\n// ❓ Running a Task with Results\nimport { cancellation } from './workflow';\n// ...\nasync function main() {\n  // 👀 Run the workflow with results\n  const res = await cancellation.run({});\n\n  // 👀 Access the results of the workflow\n  console.log(res.Completed);\n  // !!\n}\n\nif (require.main === module) {\n  main()\n    .catch(console.error)\n    .finally(() => process.exit(0));\n}\n";
export const language = "ts";
