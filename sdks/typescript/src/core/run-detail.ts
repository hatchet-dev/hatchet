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

/** The stored text, or `undefined` when the engine stored nothing. */
function decodeText(b: Uint8Array | undefined): string | undefined {
  return b?.length ? new TextDecoder().decode(b) : undefined;
}

/**
 * The forgiving decode `getDetails` shapes payloads with: absent, `null` and malformed JSON
 * all read as `null`, so inspecting a run never throws.
 */
function decodeJson(text: string | undefined): unknown {
  if (text === undefined) return null;
  try {
    return JSON.parse(text);
  } catch {
    return null;
  }
}

/**
 * A task's output the way the Node client's `result()` reads it (`JSON.parse(output || '{}')`):
 * nothing stored is `{}`, a stored `null` is `null`, and stored text that is not JSON throws
 * the `SyntaxError`, so a corrupt output is never mistaken for an empty one.
 */
export function parseTaskOutput(task: TaskRunDetail): unknown {
  if (task.rawOutput === undefined) {
    return task.output ?? {};
  }
  // The forgiving decode already parsed valid JSON; only a `null` result needs the strict
  // parse to tell a stored `null` from text that failed to parse.
  return task.output !== null ? task.output : JSON.parse(task.rawOutput);
}

/**
 * Converts a `GetRunDetails` response into the `RunDetail` both clients return: statuses
 * become `V1TaskStatus` values and the JSON payloads are decoded.
 */
export function toRunDetail(raw: GetRunDetailsResponse): RunDetail {
  return {
    status: PROTO_STATUS_MAP[raw.status] ?? V1TaskStatus.RUNNING,
    done: raw.done,
    input: decodeJson(decodeText(raw.input)),
    additionalMetadata: decodeJson(decodeText(raw.additionalMetadata)),
    isEvicted: raw.isEvicted,
    taskRuns: Object.fromEntries(
      Object.entries(raw.taskRuns).map(([id, tr]) => {
        const rawOutput = decodeText(tr.output);
        return [
          id,
          {
            externalId: tr.externalId,
            readableId: tr.readableId,
            status: PROTO_STATUS_MAP[tr.status] ?? V1TaskStatus.RUNNING,
            output: decodeJson(rawOutput),
            rawOutput,
            error: tr.error,
            isEvicted: tr.isEvicted,
          } satisfies TaskRunDetail,
        ];
      })
    ),
  };
}

// Keep runStatusToJSON importable for callers who want the raw proto status as a string.
export { runStatusToJSON };
