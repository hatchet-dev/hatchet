import { z } from 'zod';

export function isValidUUID(uuid: string): boolean {
  return z.string().uuid().safeParse(uuid).success;
}
