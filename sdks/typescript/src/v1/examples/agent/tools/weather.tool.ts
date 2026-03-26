import { client } from './../client';
import { z } from 'zod';

const WeatherInput = z.object({
  city: z.string().describe('The city to get the weather for'),
});

const WeatherOutput = z.object({
  weather: z.string(),
});

export const weather = client.tool({
  name: 'weather',
  description: 'Get the weather in a given city',
  inputSchema: WeatherInput,
  outputSchema: WeatherOutput,
  fn: async (input) => {
    return {
      weather: 'sunny',
    };
  },
});
