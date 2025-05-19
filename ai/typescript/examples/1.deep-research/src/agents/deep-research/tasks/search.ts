import { hatchet } from '@/hatchet.client';
import { z } from 'zod';
import { generateText as aiGenerateText } from 'ai';
import { openai } from '@ai-sdk/openai';

export const SearchInputSchema = z.object({
  query: z.string(),
});

export type SearchInput = z.infer<typeof SearchInputSchema>;

export type SearchOutput = {   
  query: string;
  sources: {
    url: string;
    title?: string;
  }[];
};

export const search = hatchet.task({
  name: 'search',
  fn: async (input: SearchInput, ctx): Promise<SearchOutput> => {
    const validatedInput = SearchInputSchema.parse(input);

    const result = await aiGenerateText({
      abortSignal: ctx.abortController.signal,
      model: openai.responses('gpt-4o-mini'),
      prompt: `${validatedInput.query}`,
      tools: {
        web_search_preview: openai.tools.webSearchPreview({
          // optional configuration:
          searchContextSize: 'high',
          userLocation: {
            type: 'approximate',
            city: 'San Francisco',
            region: 'California',
          },
        }),
      },
      // Force web search tool:
      toolChoice: { type: 'tool', toolName: 'web_search_preview' },
    });
    
    // URL sources
    return {
      query: validatedInput.query,
      sources: result.sources.map((source) => ({
        url: source.url,
        title: source.title,
      })),
    };
  },
});
