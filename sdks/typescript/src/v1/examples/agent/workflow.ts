// > Declaring a Task
import { hatchet } from '../hatchet-client';
import { z } from 'zod';

// Note that for agent tools, Zod must be used to create the input and output types for workflows/tasks
const TemperatureCoordinates = z.object({
  latitude: z.number(),
  longitude: z.number(),
});

const TemperatureInput = z.object({
  locationName: z.string(),
  coords: TemperatureCoordinates,
});

export type TemperatureInputWithZod = z.infer<typeof TemperatureInput>;

const temperatureRequest = async (input: TemperatureInputWithZod) => {
  const response = await fetch(
    `https://api.open-meteo.com/v1/forecast?latitude=${input.coords.latitude}&longitude=${input.coords.longitude}&current=temperature_2m&temperature_unit=fahrenheit`
  );
  const data: any = await response.json();

  return {
    text: `Temperature: ${data.current.temperature_2m}°F`,
  };
};

export const getTemperature = hatchet.task({
  name: 'getTemperature',
  retries: 3,
  fn: temperatureRequest,
  inputValidator: TemperatureInput,
  description: 'Get the current temperature at a location',
});

export const getTemperatureWorkflow = hatchet.workflow<TemperatureInputWithZod>({
  name: 'getTemperatureWorkflow',
  inputValidator: TemperatureInput,
  description: 'Get the current temperature at a location',
});

getTemperatureWorkflow.task({
  name: 'getTemperature',
  fn: async (input: TemperatureInputWithZod, _) => {
    return await temperatureRequest(input);
  },
});

// !!
