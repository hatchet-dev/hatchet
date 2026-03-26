import { client } from './../../../../client';
import z from "zod";
import { generateText } from "ai";

export const oneTool = client.tool({
  name: "one-tool",
  description: "A tool that returns 1",
  inputSchema: z.object({
    message: z.string(),
  }),
  outputSchema: z.object({
    oneOutput: z.boolean(),
  }),
  fn: async (input) => {

    // Make an LLM call to get the oneOutput
    const oneOutput = await generateText({
      model: client.defaultLanguageModel,
      prompt: `Is the following text about an animal? If so, return "yes", otherwise return "no": ${input.message}`,
    });

    return {
      oneOutput: oneOutput.text === "yes",
    };
  },
});
