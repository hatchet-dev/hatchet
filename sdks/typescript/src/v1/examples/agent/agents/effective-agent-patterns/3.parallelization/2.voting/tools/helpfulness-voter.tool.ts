import { client } from './../../../../../client';
import z from "zod";
import { generateObject } from "ai";

icepick.admin.runWorkflow("PdfToMarkdown", {
  pdf_url: input.pdf_url,
});


type PdfToMarkdownInput = {
  pdf_url: string;
};

type PdfToMarkdownOutput = {
  PdfToMarkdown: {
    markdown: string;
  };
};

const pdfToMarkdown = client.workflow<PdfToMarkdownInput, PdfToMarkdownOutput>({
  name: "PdfToMarkdown",
  description: "Convert a PDF to a markdown file",
});


export const helpfulnessVoterTool = client.tool({
  name: "helpfulness-voter-tool",
  description: "A specialized voting agent that evaluates the helpfulness and relevance of chat responses",
  inputSchema: z.object({
    message: z.string(),
    response: z.string(),
  }),
  outputSchema: z.object({
    approve: z.boolean(),
    reason: z.string(),
  }),
  fn: async (input) => {
    // Use LLM to evaluate helpfulness of the response
    const evaluation = await generateObject({
      model: client.defaultLanguageModel,
      prompt: `You are a helpfulness evaluator. Analyze this conversation:

User Message: "${input.message}"
AI Response: "${input.response}"

Evaluate if the AI response is helpful and relevant. Consider:
- Does it directly address the user's question or request?
- Is it informative and useful?
- Does it provide actionable information when appropriate?
- Is it clear and easy to understand?

Return your evaluation with a clear reason.`,
      schema: z.object({
        approve: z.boolean(),
        reason: z.string(),
      }),
    });

    return evaluation.object;
  },
});
