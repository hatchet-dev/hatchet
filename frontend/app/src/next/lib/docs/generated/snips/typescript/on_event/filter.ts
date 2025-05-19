import { Snippet } from '@/next/lib/docs/generated/snips/types';

const snippet: Snippet = {
  language: 'typescript ',
  content:
    "import { hatchet } from '../hatchet-client';\nimport { lower, SIMPLE_EVENT } from './workflow';\n\n// > Create a filter\nhatchet.filters.create({\n    workflowId: lower.id,\n    expression: 'input.ShouldSkip == false',\n    scope: 'foobarbaz',\n    payload: {\n        main_character: 'Anna',\n        supporting_character: 'Stiva',\n        location: 'Moscow',\n    },\n})\n\n// > Skip a run\nhatchet.events.push(\n    SIMPLE_EVENT,\n    {\n        'Message': 'hello',\n        'ShouldSkip': true,\n    },\n    {\n        scope: 'foobarbaz',\n    }\n)\n\n// > Trigger a run\nhatchet.events.push(\n    SIMPLE_EVENT,\n    {\n        'Message': 'hello',\n        'ShouldSkip': false,\n    },\n    {\n        scope: 'foobarbaz',\n    }\n)",
  source: 'out/typescript/on_event/filter.ts',
  blocks: {
    create_a_filter: {
      start: 5,
      stop: 14,
    },
    skip_a_run: {
      start: 17,
      stop: 26,
    },
    trigger_a_run: {
      start: 29,
      stop: 38,
    },
  },
  highlights: {},
}; // Then replace double quotes with single quotes

export default snippet;
