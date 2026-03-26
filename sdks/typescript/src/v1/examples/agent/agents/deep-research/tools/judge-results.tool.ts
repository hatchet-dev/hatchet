import { z } from "zod";
import { generateObject } from "ai";
import { openai } from "@ai-sdk/openai";
import { client } from './../../../client';

const JudgeResultsInputSchema = z.object({
  query: z.string(),
  result: z.string(),
});

const JudgeResultsOutputSchema = z.object({
  reason: z.string(),
  isComplete: z.boolean(),
});

export const judgeResults = client.tool({
  name: "judge-results",
  description: "Judge if the result is complete",
  inputSchema: JudgeResultsInputSchema,
  outputSchema: JudgeResultsOutputSchema,
  fn: async (input, ctx) => {
    const validatedInput = JudgeResultsInputSchema.parse(input);

    const result = await generateObject({
      abortSignal: ctx.abortController.signal,
      prompt: `
Judge the following answer to the query for completeness:
"""${validatedInput.query}"""

Answer:
"""${validatedInput.result}"""

Completeness means that the answer includes all the information that is relevant to the query and that the answer is not missing any important details. Does the answer leave any new questions unanswered?
`,
      model: openai("gpt-4.1-mini"),
      schema: z.object({
        reason: z.string(),
        isComplete: z.boolean(),
      }),
    });

    return {
      reason: result.object.reason,
      isComplete: result.object.isComplete,
    };
  },
});
