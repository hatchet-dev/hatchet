import { z } from "zod";
import { generateText as aiGenerateText } from "ai";
import { openai } from "@ai-sdk/openai";
import { client } from './../../../client';

const WebsiteToMdxInputSchema = z.object({
  url: z.string().url(),
  index: z.number(),
  title: z.string(),
});

const WebsiteToMdxOutputSchema = z.object({
  index: z.number(),
  title: z.string(),
  url: z.string(),
  markdown: z.string(),
});

export const websiteToMd = client.tool({
  name: "website-to-md",
  description: "Load a website by its url and convert it to Markdown",
  inputSchema: WebsiteToMdxInputSchema,
  outputSchema: WebsiteToMdxOutputSchema,
  fn: async (input, ctx) => {
    const result = await aiGenerateText({
      abortSignal: ctx.abortController.signal,
      model: openai.responses("gpt-4.1-mini"),
      prompt: `Convert the content of this webpage to clean, well-formatted Markdown. Preserve the structure, headings, and important content while removing unnecessary elements like ads and navigation menus. Only include the content from the page, do not write any additional text. URL: ${input.url}`,
      tools: {
        web_search_preview: openai.tools.webSearchPreview({
          searchContextSize: "high",
        }),
      },
      toolChoice: { type: "tool", toolName: "web_search_preview" },
    });

    return {
      index: input.index,
      title: input.title,
      url: input.url,
      markdown: result.text,
    };
  },
});
