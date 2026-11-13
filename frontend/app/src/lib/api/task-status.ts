import { V1TaskStatus } from './generated/data-contracts';

// EVICTED is a storage-level status; the API reports evicted tasks as RUNNING, so it is never offered as a filter
export const FILTERABLE_TASK_STATUSES: V1TaskStatus[] = Object.values(
  V1TaskStatus,
).filter((s) => s !== V1TaskStatus.EVICTED);
