import { z } from 'zod/v4';

export function isValidUUID(uuid: string): boolean {
  return z.uuid().safeParse(uuid).success;
}
