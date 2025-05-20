import { Snippet } from '@/next/lib/docs/generated/snips/types';

const snippet: Snippet = {
  language: 'typescript ',
  content:
    "import { hatchet } from '../hatchet-client';\nimport { Input } from './workflow';\n\nasync function main() {\n  // > Pushing an Event\n  const res = await hatchet.events.push<Input>('simple-event:create', {\n    Message: 'hello',\n    ShouldSkip: false,\n  });\n\n  console.log(res.eventId);\n}\n\nif (require.main === module) {\n  main();\n}\n",
  source: 'out/typescript/on_event/event.ts',
  blocks: {
    pushing_an_event: {
      start: 6,
      stop: 9,
    },
  },
  highlights: {},
};

export default snippet;
