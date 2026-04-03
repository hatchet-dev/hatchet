import { client } from './../../../../client';
import z from "zod";
import { generateText } from "ai";

export const threeTool = client.tool({
  name: "three-tool",
  description: "A tool that makes text into a haiku",
  inputSchema: z.object({
    twoOutput: z.string(),
  }),
  outputSchema: z.object({
    threeOutput: z.string(),
  }),
  fn: async (input) => {

    // Make
    const threeOutput = await generateText({
      model: client.defaultLanguageModel,
      prompt: `Make the following text into a haiku: ${input.twoOutput}`,
    });

    return {
      threeOutput: threeOutput.text,
    };
  },
});
