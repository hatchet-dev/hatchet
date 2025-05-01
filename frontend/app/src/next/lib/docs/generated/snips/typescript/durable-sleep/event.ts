import { Snippet } from '@/next/lib/docs/generated/snips/types';

const snippet: Snippet = {
  language: 'typescript ',
  content:
    "import { hatchet } from '../hatchet-client';\n\nasync function main() {\n  const event = await hatchet.events.push('user:event', {\n    Data: { Hello: 'World' },\n  });\n}\n\nif (require.main === module) {\n  main()\n    .then(() => process.exit(0))\n    .catch((error) => {\n      console.error('Error:', error);\n      process.exit(1);\n    });\n}\n",
  source: 'out/typescript/durable-sleep/event.ts',
  blocks: {},
  highlights: {},
}; // Then replace double quotes with single quotes

export default snippet;
