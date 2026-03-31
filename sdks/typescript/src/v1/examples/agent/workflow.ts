// > Declaring a Task
import { hatchet } from '../hatchet-client';
import { z } from 'zod';

const TemperatureCoordinates = z.object({
  latitude: z.number(),
  longitude: z.number(),
});

const TemperatureInput = z.object({
  locationName: z.string(),
  coords: TemperatureCoordinates,
});

export type TemperatureInputWithZod = z.infer<typeof TemperatureInput>;

export const getTemperature = hatchet.task({
  name: 'getTemperature',
  retries: 3,
  fn: async (input: TemperatureInputWithZod) => {
    const response = await fetch(
      `https://api.open-meteo.com/v1/forecast?latitude=${input.coords.latitude}&longitude=${input.coords.longitude}&current=temperature_2m&temperature_unit=fahrenheit`
    );
    const data: any = await response.json();

    return {
      text: `Temperature: ${data.current.temperature_2m}°F`,
    };
  },
  inputValidator: TemperatureInput,
});

// !!
