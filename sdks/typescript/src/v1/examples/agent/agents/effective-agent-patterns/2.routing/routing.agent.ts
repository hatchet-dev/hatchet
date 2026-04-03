import { client } from './../../../client';
import z from "zod";
import { salesTool, supportTool } from "./tools/calls.tool";

/**
 * ROUTING PATTERN
 *
 * Based on Anthropic's "Building Effective Agents" blog post:
 * https://www.anthropic.com/engineering/building-effective-agents
 *
 * Pattern Description:
 * Routing classifies an input and directs it to a specialized followup task.
 * This allows separation of concerns and building more specialized prompts
 * without one input type hurting performance on others.
 *
 * When to use:
 * - Complex tasks with distinct categories better handled separately
 * - Classification can be handled accurately by LLM or traditional algorithms
 * - Need specialized handling for different input types
 *
 * Examples:
 * - Customer service: routing questions, refunds, technical support
 * - Multi-model routing: easy questions to smaller models, hard to larger
 * - Content classification with specialized processors
 */

const RoutingAgentInput = z.object({
  message: z.string(),
});

const RoutingAgentOutput = z.object({
  message: z.string(),
  canHelp: z.boolean(),
});

export const routingToolbox = client.toolbox({
    tools: [supportTool, salesTool],
});

export const routingAgent = client.agent({
  name: "routing-agent",
  executionTimeout: "1m",
  inputSchema: RoutingAgentInput,
  outputSchema: RoutingAgentOutput,
  description: "Demonstrates routing: classify input and direct to specialized handlers",
  fn: async (input, ctx) => {

    // STEP 1: Classification - Determine the type of request
    // This is the key step in routing - understanding what kind of input we have
    // so we can direct it to the most appropriate specialized handler
    const route = await routingToolbox.pickAndRun({
        prompt: input.message,
    });

    // STEP 2: Route to specialized handler based on classification
    // Each case represents a different specialized workflow optimized for that type
    switch(route.name) {
        case "support-tool": {
            // Route to support-specialized LLM with support-specific tools and prompts
            return {
                message: route.output.response,
                canHelp: true,
            }
        }
        case "sales-tool": {
            // Route to sales-specialized LLM with sales-specific tools and prompts
            return {
                message: route.output.response,
                canHelp: true,
            }
        }
        default:
            routingToolbox.assertExhaustive(route);
            return {
                message: "I am sorry, I cannot help with that yet.",
                canHelp: false,
            }
    }
  },
});
