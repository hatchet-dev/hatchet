// Generated from /Users/gabrielruttner/dev/hatchet/examples/typescript/sticky/run.ts
export const content = "/* eslint-disable no-console */\nimport { retries } from '../retries/workflow';\n\nasync function main() {\n  try {\n    const res = await retries.run({});\n    console.log(res);\n  } catch (e) {\n    console.log('error', e);\n  }\n}\n\nif (require.main === module) {\n  main()\n    .catch(console.error)\n    .finally(() => process.exit(0));\n}\n";
export const language = "ts";
