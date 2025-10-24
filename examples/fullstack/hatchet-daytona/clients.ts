import { Hatchet } from '@hatchet-dev/typescript-sdk';
import { Daytona } from '@daytonaio/sdk'

export const hatchet = Hatchet.init();
export const daytona = new Daytona({ apiKey: process.env.DAYTONA_API_KEY });

