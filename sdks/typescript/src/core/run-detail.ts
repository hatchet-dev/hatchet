import { V1TaskStatus } from '@hatchet/clients/rest/generated/data-contracts';
import { GetRunDetailsResponse, RunStatus, runStatusToJSON } from '@hatchet/protoc/v1/workflows';
import type { RunDetail, TaskRunDetail } from './types';

// EVICTED is not in V1TaskStatus; treat as RUNNING per the proto comment.
const PROTO_STATUS_MAP: Record<RunStatus, V1TaskStatus> = {
  [RunStatus.QUEUED]: V1TaskStatus.QUEUED,
  [RunStatus.RUNNING]: V1TaskStatus.RUNNING,
  [RunStatus.COMPLETED]: V1TaskStatus.COMPLETED,
  [RunStatus.FAILED]: V1TaskStatus.FAILED,
  [RunStatus.CANCELLED]: V1TaskStatus.CANCELLED,
  [RunStatus.EVICTED]: V1TaskStatus.RUNNING,
  [RunStatus.UNRECOGNIZED]: V1TaskStatus.RUNNING,
};

function decodeBytes(b: Uint8Array | undefined): unknown {
  if (!b?.length) return null;
  try {
    return JSON.parse(new TextDecoder().decode(b));
  } catch {
    return null;
  }
}

/**
 * Converts a `GetRunDetails` response into the `RunDetail` both clients return: statuses
 * become `V1TaskStatus` values and the JSON payloads are decoded.
 */
export function toRunDetail(raw: GetRunDetailsResponse): RunDetail {
  return {
    status: PROTO_STATUS_MAP[raw.status] ?? V1TaskStatus.RUNNING,
    done: raw.done,
    input: decodeBytes(raw.input),
    additionalMetadata: decodeBytes(raw.additionalMetadata),
    isEvicted: raw.isEvicted,
    taskRuns: Object.fromEntries(
      Object.entries(raw.taskRuns).map(([id, tr]) => [
        id,
        {
          externalId: tr.externalId,
          readableId: tr.readableId,
          status: PROTO_STATUS_MAP[tr.status] ?? V1TaskStatus.RUNNING,
          output: decodeBytes(tr.output),
          error: tr.error,
          isEvicted: tr.isEvicted,
        } satisfies TaskRunDetail,
      ])
    ),
  };
}

// Keep runStatusToJSON importable for callers who want the raw proto status as a string.
export { runStatusToJSON };
