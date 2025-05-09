import { Snippet } from '@/next/lib/docs/generated/snips/types';

const snippet: Snippet = {
  language: 'typescript ',
  content:
    "import { hatchet } from '../hatchet-client';\n\n// (optional) Define the input type for the workflow\nexport type SimpleInput = {\n  Message: string;\n};\n\n// > Trigger on an event\nexport const simple = hatchet.task({\n  name: 'simple',\n  onEvents: ['user:created'],\n  fn: (input: SimpleInput) => {\n    // ...\n    return {\n      TransformedMessage: input.Message.toLowerCase(),\n    };\n  },\n});\n",
  source: 'out/typescript/landing_page/event-signaling.ts',
  blocks: {
    trigger_on_an_event: {
      start: 9,
      stop: 18,
    },
  },
  highlights: {},
}; // Then replace double quotes with single quotes

export default snippet;
