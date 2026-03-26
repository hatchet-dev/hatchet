import { client } from './../../../client';
import z from "zod";
import { oneTool } from "./tools/one.tool";
import { twoTool } from "./tools/two.tool";
import { threeTool } from "./tools/three.tool";

/**
 * PROMPT CHAINING PATTERN
 *
 * Based on Anthropic's "Building Effective Agents" blog post:
 * https://www.anthropic.com/engineering/building-effective-agents
 *
 * Pattern Description:
 * Prompt chaining decomposes a task into a sequence of steps, where each LLM call
 * processes the output of the previous one. You can add programmatic checks (gates)
 * on intermediate steps to ensure the process stays on track.
 *
 * When to use:
 * - Tasks can be easily decomposed into fixed subtasks
 * - Trading latency for higher accuracy by making each LLM call easier
 * - Need validation gates between steps
 *
 * Examples:
 * - Generating marketing copy, then translating it
 * - Writing outline → checking criteria → writing full document
 * - Multi-step content processing with validation
 */

const PromptChainingAgentInput = z.object({
  message: z.string(),
});

const PromptChainingAgentOutput = z.object({
  result: z.string(),
});

export const promptChainingAgent = client.agent({
  name: "prompt-chaining-agent",
  executionTimeout: "1m",
  inputSchema: PromptChainingAgentInput,
  outputSchema: PromptChainingAgentOutput,
  description: "Demonstrates prompt chaining: sequential LLM calls with validation gates",
  fn: async (input, ctx) => {

    // STEP 1: First LLM call - Process the initial message
    // This step determines if the message is about an animal
    const { oneOutput } = await oneTool.run({
        message: input.message,
    });

    // GATE: Programmatic validation check between steps
    // This is a key feature of prompt chaining - we can validate intermediate results
    // and control the flow based on that validation
    if(!oneOutput) {
        // FAIL: If validation fails, we can terminate early or redirect
        return {
            result: 'Please provide a message about an animal'
        }
    }

    // PASS: If validation succeeds, continue to next step
    // STEP 2: Second LLM call - Transform the validated input
    // Since we know it's about an animal, translate to Spanish
    const { twoOutput } = await twoTool.run({
        message: input.message,
    });

    // STEP 3: Third LLM call - Final transformation
    // Convert the Spanish message into a haiku format
    const { threeOutput } = await threeTool.run({
        twoOutput, // Note: Using output from previous step as input
    });

    // Return the final processed result
    return {
        result: threeOutput
    }
  },
});
