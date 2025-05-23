import { hatchet } from '@/hatchet.client';
import {
  deepResearchAgent,
  deepResearchTaskbox,
} from './agents/deep-research';
import { generateText } from './blocks/ai/generateText';
import { summarize } from './agents/deep-research/tasks/summarize';
import { search } from './agents/deep-research/tasks/search';
import { planSearch } from './agents/deep-research/tasks/plan-search';
import { websiteToMd } from './blocks/ai/website-to-md';
import { judgeResults } from './agents/deep-research/tasks/judge-results';
import { extractFacts } from './agents/deep-research/tasks/extract-facts';
import { judgeFacts } from './agents/deep-research/tasks/judge-facts';

const main = async () => {
  const worker = await hatchet.worker('deep-research', {
    workflows: [
      deepResearchAgent,
      ...deepResearchTaskbox.register,
      generateText,
      summarize,
      planSearch,
      search,
      websiteToMd,
      judgeResults,
      extractFacts,
      judgeFacts,
    ],
  });

  await worker.start();
};

main();