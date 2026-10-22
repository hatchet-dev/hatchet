import { V1ConcurrencyScope } from '@/lib/api';
import { z } from 'zod';

// Imported by the router, so this module must stay free of page components.
export const concurrencySearchSchema = z.object({
  // absent means all scopes
  scope: z.nativeEnum(V1ConcurrencyScope).optional().catch(undefined),
  // strategyRef of the strategy whose keys are open in the side panel
  strategy: z.string().optional().catch(undefined),
  // one-based page of the strategies table; absent is the first page
  page: z.number().int().min(1).optional().catch(undefined),
});

export type ConcurrencySearch = z.infer<typeof concurrencySearchSchema>;

// Drops default values so the URL only carries what differs from the default view.
export function normalizeConcurrencySearch(
  search: ConcurrencySearch,
): ConcurrencySearch {
  return {
    ...(search.scope && { scope: search.scope }),
    ...(search.strategy && { strategy: search.strategy }),
    ...(search.page && search.page > 1 && { page: search.page }),
  };
}
