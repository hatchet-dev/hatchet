import { openai } from '@ai-sdk/openai';
import { Hatchet } from '@hatchet/index';

export const client = Hatchet.init({
  defaultLanguageModel: openai('gpt-4o-mini'),
});
