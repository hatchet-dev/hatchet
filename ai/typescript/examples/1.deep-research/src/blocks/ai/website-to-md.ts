import { hatchet } from '@/hatchet.client';
import { z } from 'zod';
import { generateText as aiGenerateText } from 'ai';
import { openai } from '@ai-sdk/openai';

const WebsiteToMdxInputSchema = z.object({
  url: z.string().url(),
  index: z.number(),
  title: z.string(),
});

type WebsiteToMdxInput = z.infer<typeof WebsiteToMdxInputSchema>;

type WebsiteToMdxOutput = {
  index: number;
  title: string;
  url: string;
  markdown: string;
};

export const websiteToMd = hatchet.task({
  name: 'website-to-md',
  fn: async (input: WebsiteToMdxInput, ctx): Promise<WebsiteToMdxOutput> => {
    const validatedInput = WebsiteToMdxInputSchema.parse(input);

    const result = await aiGenerateText({
      abortSignal: ctx.abortController.signal,
      model: openai.responses('gpt-4.1-mini'),
      prompt: `Convert the content of this webpage to clean, well-formatted Markdown. Preserve the structure, headings, and important content while removing unnecessary elements like ads and navigation menus. Only include the content from the page, do not write any additional text. URL: ${validatedInput.url}`,
      tools: {
        web_search_preview: openai.tools.webSearchPreview({
          searchContextSize: 'high',
        }),
      },
      toolChoice: { type: 'tool', toolName: 'web_search_preview' },
    });

    return {
      index: validatedInput.index,
      title: validatedInput.title,
      url: validatedInput.url,
      markdown: result.text,
    };
  },
});
