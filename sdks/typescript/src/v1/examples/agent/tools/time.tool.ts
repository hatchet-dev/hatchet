import { z } from "zod";
import { client } from './../client';

const SimpleInput = z.object({
  message: z.string()
})

const SimpleOutput = z.object({
  response: z.string()
})

export const simple = client.tool({
  name: "simple-tool",
  description: "Scaffold tool ",
  inputSchema: SimpleInput,
  outputSchema: SimpleOutput,
  fn: async (input) => {

    // TODO: Replace this with your actual tool implementation

    return {
      response: 'Hello, World!'
    };
  }
});
