import { V1ConcurrencyScope } from '@/lib/api';
import { z } from 'zod';

export const concurrencySearchSchema = z.object({
  scope: z.nativeEnum(V1ConcurrencyScope).optional().catch(undefined),
  strategy: z.string().optional().catch(undefined),
  page: z.number().int().min(1).optional().catch(undefined),
});

export type ConcurrencySearch = z.infer<typeof concurrencySearchSchema>;

export function normalizeConcurrencySearch(
  search: ConcurrencySearch,
): ConcurrencySearch {
  return {
    ...(search.scope && { scope: search.scope }),
    ...(search.strategy && { strategy: search.strategy }),
    ...(search.page && search.page > 1 && { page: search.page }),
  };
}
