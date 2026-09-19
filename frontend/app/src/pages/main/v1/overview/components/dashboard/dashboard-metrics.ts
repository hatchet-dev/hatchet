import {
  V1TaskStatus,
  WorkerStatus,
  type V1TaskRunMetric,
  type Worker,
} from '@/lib/api';

// Worker utilization is defined as occupied slots / total slots across ACTIVE
// workers only. For each ACTIVE worker we sum, over every slot type in its
// slotConfig, the slot "limit" (total) and "available" (free). Occupied is
// total minus available, so utilization = 1 - sum(available) / sum(limit).
// PAUSED and INACTIVE workers are excluded because their slots are not
// schedulable. A slot with no reported "available" value is treated as fully
// available (contributes zero occupancy). The result is clamped to [0, 1] and
// returns 0 when there are no active slots (divide-by-zero guard), so callers
// can render it directly as a percentage.
export function computeWorkerUtilization(workers: Worker[]): number {
  let totalSlots = 0;
  let availableSlots = 0;

  for (const worker of workers) {
    if (worker.status !== WorkerStatus.ACTIVE) {
      continue;
    }

    const slotConfig = worker.slotConfig ?? {};
    for (const cfg of Object.values(slotConfig)) {
      const limit = cfg?.limit ?? 0;
      const available = cfg?.available ?? limit;
      totalSlots += limit;
      availableSlots += available;
    }
  }

  if (totalSlots <= 0) {
    return 0;
  }

  const occupied = totalSlots - availableSlots;
  return Math.min(1, Math.max(0, occupied / totalSlots));
}

// Count of a single status in the status-metrics array, defaulting to 0 when
// the status is absent (the API omits statuses with no runs).
export function statusCount(
  metrics: V1TaskRunMetric[] | undefined,
  status: V1TaskStatus,
): number {
  return metrics?.find((m) => m.status === status)?.count ?? 0;
}

// Sum of run counts across every status bucket.
export function totalTaskCount(metrics: V1TaskRunMetric[] | undefined): number {
  return (metrics ?? []).reduce((sum, m) => sum + (m.count ?? 0), 0);
}

// Sum of queued step-run counts across the queue map returned by
// getStepRunQueueMetrics. The map is typed as `object` in the generated
// client; values are per-queue counts.
export function sumQueueMetrics(queues: object | undefined): number {
  if (!queues) {
    return 0;
  }

  return Object.values(queues).reduce<number>((sum, value) => {
    const n = typeof value === 'number' ? value : Number(value);
    return sum + (Number.isFinite(n) ? n : 0);
  }, 0);
}

// Truncate long error strings for compact display in the errors panel.
const ERROR_MAX_CHARS = 140;
export function truncateError(message: string | undefined): string {
  if (!message) {
    return '';
  }

  const normalized = message.replace(/\s+/g, ' ').trim();
  if (normalized.length <= ERROR_MAX_CHARS) {
    return normalized;
  }

  return `${normalized.slice(0, ERROR_MAX_CHARS)}...`;
}
