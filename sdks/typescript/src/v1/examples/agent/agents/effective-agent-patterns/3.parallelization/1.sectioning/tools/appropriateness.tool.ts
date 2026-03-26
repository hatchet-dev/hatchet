import { client } from './../../../../../client';
import z from "zod";
import { generateObject, generateText } from "ai";

export const appropriatenessCheckTool = client.tool({
  name: "appropriateness-check-tool",
  description: "Determines if a message is appropriate and safe to respond to",
  inputSchema: z.object({
    message: z.string(),
  }),
  outputSchema: z.object({
    isAppropriate: z.boolean(),
    reason: z.string(),
  }),
  fn: async (input) => {
    const result = await generateObject({
      model: client.defaultLanguageModel,
      prompt: `
Analyze the following message to determine if it's appropriate to respond to.
Check for harmful content, inappropriate requests, or content that violates safety guidelines.

Message: "${input.message}"

Evaluate if the AI response is safe and appropriate. Consider:
- Does it contain harmful, offensive, or inappropriate content?
- Does it promote dangerous activities?
- Is it respectful and professional?

Return your evaluation with a clear reason.
`,
      schema: z.object({
        isAppropriate: z.boolean(),
        reason: z.string(),
      }),
    });

    return result.object;
  },
});
