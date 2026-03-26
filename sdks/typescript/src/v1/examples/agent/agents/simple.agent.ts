import { client } from './../client';
import { weather } from '../tools/weather.tool';
import { holiday, time } from '../tools/time.tool';
import z from 'zod';

const SimpleAgentInput = z.object({
  message: z.string(),
});

const SimpleAgentOutput = z.object({
  message: z.string(),
});

export const simpleToolbox = client.toolbox({
  tools: [weather, time, holiday],
});

export const simpleAgent = client.agent({
  name: 'simple-agent',
  executionTimeout: '1m',
  inputSchema: SimpleAgentInput,
  outputSchema: SimpleAgentOutput,
  description: 'A simple agent to get the weather and time',
  fn: async (input, ctx) => {
    const result = await simpleToolbox.pickAndRun({
      prompt: input.message,
    });

    switch (result.name) {
      case 'weather':
        return {
          message: `The weather in ${result.args.city} is ${result.output}`,
        };
      case 'time':
        return {
          message: `The time in ${result.args.city} is ${result.output}`,
        };
      case 'holiday':
        return {
          message: `The holiday in ${result.args.country} is ${result.output}`,
        };
      default:
        return simpleToolbox.assertExhaustive(result);
    }
  },
});
