
import { z } from "zod";
import { client } from './../../../client';
import { generateText } from "ai";
import { openai } from "@ai-sdk/openai";

export const SummarizeInputSchema = z.object({
  text: z.string(),
  facts: z.array(
    z.object({
      text: z.string(),
      sourceIndex: z.number(),
    })
  ),
  sources: z.array(
    z.object({
      url: z.string(),
      title: z.string().optional(),
      index: z.number(),
    })
  ),
});

export const SummarizeOutputSchema = z.object({
  summary: z.string(),
});

export const summarize = client.tool({
  name: "summarize",
  description: "Summarize a set of facts",
  inputSchema: SummarizeInputSchema,
  outputSchema: SummarizeOutputSchema,
  fn: async (input, ctx) => {
    // Create a map of source indices to source information for easy lookup
    const sourceMap = new Map(
      input.sources.map((source) => [source.index, source])
    );

    // Group facts by source
    const factsBySource = new Map<number, string[]>();
    input.facts.forEach((fact, index) => {
      const facts = factsBySource.get(fact.sourceIndex) || [];
      facts.push(`${index + 1}. ${fact.text}`);
      factsBySource.set(fact.sourceIndex, facts);
    });

    // Format facts grouped by source
    const formattedFacts = Array.from(factsBySource.entries()).map(
      ([sourceIndex, facts]) => {
        const source = sourceMap.get(sourceIndex);
        if (!source) {
          throw new Error(`Source with index ${sourceIndex} not found`);
        }
        return `From ${source.title || "Untitled"} (${
          source.url
        }):\n${facts.join("\n")}`;
      }
    );
    const result = await generateText({
      abortSignal: ctx.abortController.signal,
      system: `You are a professional researcher helping to write a detailed report based on verified facts.`,
      prompt: `
Write a comprehensive summary based on these verified facts:

${formattedFacts.join("\n\n")}

Requirements:
1. The summary should be based ONLY on the provided facts
2. Each fact should be referenced using its number in brackets (e.g. [1], [2])
3. The summary should be well-structured and flow logically
4. The summary should be written in the style of a professional researcher
5. The summary should be written in the language of the original query
6. Include a "Sources" section at the end listing all referenced sources with their numbers
7. Write the summary in markdown format and present relevant information in a table format

Original query:
"""
${input.text}
"""
`,
      model: openai("gpt-4.1-mini"),
    });

    return {
      summary: result.text,
    };
  },
});
