import { Snippet } from '@/next/lib/docs/generated/snips/types';

const snippet: Snippet = {
  language: 'typescript ',
  content:
    "// > Client Run Methods\nimport { hatchet } from '../hatchet-client';\n\nhatchet.run('simple', { Message: 'Hello, World!' });\n\nhatchet.runNoWait('simple', { Message: 'Hello, World!' }, {});\n\nhatchet.schedules.create('simple', {\n  triggerAt: new Date(Date.now() + 1000 * 60 * 60 * 24),\n  input: { Message: 'Hello, World!' },\n});\n\nhatchet.crons.create('simple', {\n  name: 'my-cron',\n  expression: '0 0 * * *',\n  input: { Message: 'Hello, World!' },\n});\n",
  source: 'out/typescript/simple/client-run.ts',
  blocks: {
    client_run_methods: {
      start: 2,
      stop: 17,
    },
  },
  highlights: {},
}; // Then replace double quotes with single quotes

export default snippet;
