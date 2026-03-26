
import { z } from "zod";
import { generateText as aiGenerateText } from "ai";
import { client } from './../../../client';
import { openai } from "@ai-sdk/openai";

export const SearchInputSchema = z.object({
  query: z.string(),
});
const SearchOutputSchema = z.object({
  query: z.string(),
  sources: z.array(z.object({
    url: z.string(),
    title: z.string().optional(),
  })),
});

export const search = client.tool({
  name: "search",
  description: "Search the web for information about a topic",
  inputSchema: SearchInputSchema,
  outputSchema: SearchOutputSchema,
  fn: async (input, ctx) => {
    const validatedInput = SearchInputSchema.parse(input);

    const result = await aiGenerateText({
      abortSignal: ctx.abortController.signal,
      model: openai.responses("gpt-4o-mini"),
      prompt: `${validatedInput.query}`,
      tools: {
        web_search_preview: openai.tools.webSearchPreview({
          // optional configuration:
          searchContextSize: "high",
          userLocation: {
            type: "approximate",
            city: "San Francisco",
            region: "California",
          },
        }),
      },
      // Force web search tool:
      toolChoice: { type: "tool", toolName: "web_search_preview" },
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
