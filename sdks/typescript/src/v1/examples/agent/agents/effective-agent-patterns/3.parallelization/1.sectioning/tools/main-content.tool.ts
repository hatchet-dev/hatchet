import { client } from './../../../../../client';
import z from "zod";
import { generateText } from "ai";

export const mainContentTool = client.tool({
  name: "main-content-tool",
  description: "Generates the main content section of a response",
  inputSchema: z.object({
    message: z.string(),
  }),
  outputSchema: z.object({
    mainContent: z.string(),
  }),
  fn: async (input) => {
    const result = await generateText({
      model: client.defaultLanguageModel,
      prompt: `
        Respond to the following user message.
        This should be the detailed, substantive part of the response that directly addresses the user's query.
        Provide helpful information, explanations, or answers as appropriate.

        User message: "${input.message}"
      `,
    });

    return {
      mainContent: result.text.trim(),
    };
  },
});
