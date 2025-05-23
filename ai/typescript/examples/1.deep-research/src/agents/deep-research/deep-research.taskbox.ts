import { taskbox } from '@/blocks/ai';
import { search, SearchInputSchema } from '@/agents/deep-research/tasks/search';
import { summarize, SummarizeInputSchema } from '@/agents/deep-research/tasks/summarize';

export const deepResearchTaskbox = taskbox({
  tasks: [
    { task: search, schema: SearchInputSchema },
    { task: summarize, schema: SummarizeInputSchema },
  ],
});
