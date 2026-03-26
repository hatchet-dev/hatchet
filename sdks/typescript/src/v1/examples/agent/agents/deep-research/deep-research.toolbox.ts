import { search } from '@/agents/deep-research/tools/search.tool';
import { summarize } from '@/agents/deep-research/tools/summarize.tool';
import { icepick } from '@/icepick-client';

export const deepResearchTaskbox = icepick.toolbox({
  tools: [
    search,
    summarize,
  ],
});
