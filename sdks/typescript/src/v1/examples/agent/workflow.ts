import { hatchet } from '../hatchet-client';
import { z } from 'zod/v4';

// > Models
// Agent tools require a Zod v4 inputValidator so the SDK can generate the tool input schema.
const TemperatureCoordinates = z.object({
  latitude: z.number(),
  longitude: z.number(),
});

const TemperatureInput = z.object({
  locationName: z.string(),
  coords: TemperatureCoordinates,
});

export type TemperatureInputWithZod = z.infer<typeof TemperatureInput>;
// !!

const temperatureRequest = async (input: TemperatureInputWithZod) => {
  const response = await fetch(
    `https://api.open-meteo.com/v1/forecast?latitude=${input.coords.latitude}&longitude=${input.coords.longitude}&current=temperature_2m&temperature_unit=fahrenheit`
  );
  const data: any = await response.json();

  return {
    text: `Temperature in ${input.locationName}: ${data.current.temperature_2m}°F`,
  };
};

// > Workflow definition
export const getTemperatureWorkflow = hatchet.workflow<TemperatureInputWithZod>({
  name: 'getTemperatureWorkflow',
  inputValidator: TemperatureInput,
  description: 'Get the current temperature at a location',
});

getTemperatureWorkflow.task({
  name: 'getTemperature',
  fn: temperatureRequest,
});
// !!

// > Standalone task
export const getTemperature = hatchet.task({
  name: 'getTemperature',
  retries: 3,
  fn: temperatureRequest,
  inputValidator: TemperatureInput,
  description: 'Get the current temperature at a location',
});
// !!

// > Create MCP tools
export function createTemperatureWorkflowToolClaude() {
  return getTemperatureWorkflow.mcpTool('claude');
}

export function createTemperatureWorkflowToolOpenai() {
  return getTemperatureWorkflow.mcpTool('openai');
}

export function createTemperatureTaskToolClaude() {
  return getTemperature.mcpTool('claude');
}

export function createTemperatureTaskToolOpenai() {
  return getTemperature.mcpTool('openai');
}
// !!
