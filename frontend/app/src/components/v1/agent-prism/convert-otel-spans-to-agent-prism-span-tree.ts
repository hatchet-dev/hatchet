import type {
  OtelSpanTree,
  RelevantOpenTelemetrySpanProperties,
} from "./span-tree-type";
import { OtelStatusCode } from "@/lib/api/generated/data-contracts";
import invariant from "tiny-invariant";

// ---------------------------------------------------------------------------
// Span name & attribute constants
// ---------------------------------------------------------------------------

const SPAN = {
  START_STEP_RUN: "hatchet.start_step_run",
  ENGINE_QUEUED: "hatchet.engine.queued",
  RUN_WORKFLOW: "hatchet.run_workflow",
  START_WORKFLOW: "hatchet.start_workflow",
} as const;

const ATTR = {
  STEP_RUN_ID: "hatchet.step_run_id",
  STEP_NAME: "hatchet.step_name",
  SPAN_SOURCE: "hatchet.span_source",
  ACTION_ID: "hatchet.action_id",
  WORKFLOW_NAME: "hatchet.workflow_name",
  TASK_NAME: "hatchet.task_name",
  INSTRUMENTOR: "instrumentor",
} as const;

// ---------------------------------------------------------------------------
// Shared helpers
// ---------------------------------------------------------------------------

function getStepRunId(node: OtelSpanTree): string | undefined {
  return node.spanAttributes?.[ATTR.STEP_RUN_ID];
}

function isEngineSpan(node: OtelSpanTree): boolean {
  return node.spanAttributes?.[ATTR.SPAN_SOURCE] === "engine";
}

function removeByPredicate(
  nodes: OtelSpanTree[],
  predicate: (node: OtelSpanTree) => boolean,
): void {
  let write = 0;
  for (let read = 0; read < nodes.length; read++) {
    if (!predicate(nodes[read])) {
      nodes[write++] = nodes[read];
    }
  }
  nodes.length = write;
}

function removeBySpanIds(nodes: OtelSpanTree[], ids: Set<string>): void {
  if (ids.size > 0) {
    removeByPredicate(nodes, (n) => ids.has(n.spanId));
  }
}

function countParents(nodes: OtelSpanTree[]): Map<string, number> {
  const counts = new Map<string, number>();
  for (const node of nodes) {
    if (node.parentSpanId) {
      counts.set(node.parentSpanId, (counts.get(node.parentSpanId) ?? 0) + 1);
    }
  }
  return counts;
}

// ---------------------------------------------------------------------------
// Synthetic span factories
// ---------------------------------------------------------------------------

function makeSyntheticRoot(
  children: OtelSpanTree[],
  overrides?: Partial<OtelSpanTree>,
): OtelSpanTree {
  let earliestStart = Date.now();
  for (const child of children) {
    const t = new Date(child.createdAt).getTime();
    if (t < earliestStart) {
      earliestStart = t;
    }
  }

  return {
    spanId: "__synthetic_workflow_start__",
    parentSpanId: undefined,
    spanName: SPAN.START_WORKFLOW,
    statusCode: OtelStatusCode.UNSET,
    durationNs: 0,
    createdAt: new Date(earliestStart).toISOString(),
    spanAttributes: { [ATTR.INSTRUMENTOR]: "hatchet" },
    children,
    ...overrides,
  };
}

function makeQueuedPhase(
  spanId: string,
  parentSpanId: string | undefined,
  durationNs: number,
  createdAt: string,
  attrs: Record<string, string>,
): OtelSpanTree {
  return {
    spanId,
    parentSpanId,
    spanName: SPAN.ENGINE_QUEUED,
    statusCode: OtelStatusCode.OK,
    durationNs,
    createdAt,
    spanAttributes: { [ATTR.SPAN_SOURCE]: "engine", ...attrs },
    children: [],
  };
}

function makeStepRunSpan(
  spanId: string,
  parentSpanId: string | undefined,
  createdAt: string,
  attrs: Record<string, string>,
  extra?: Partial<OtelSpanTree>,
): OtelSpanTree {
  return {
    spanId,
    parentSpanId,
    spanName: SPAN.START_STEP_RUN,
    statusCode: OtelStatusCode.UNSET,
    durationNs: 0,
    createdAt,
    spanAttributes: { [ATTR.SPAN_SOURCE]: "engine", ...attrs },
    children: [],
    ...extra,
  };
}

// ---------------------------------------------------------------------------
// Public types
// ---------------------------------------------------------------------------

export interface TaskSummaryForSynthesis {
  externalId: string;
  displayName: string;
  status: string;
  createdAt: string;
  startedAt?: string;
}

export type WorkflowRunTiming = {
  createdAt: string;
  startedAt?: string;
};

type ConvertOptions = {
  enableTraceInProgressSynthesis?: boolean;
};

// ---------------------------------------------------------------------------
// Tree transforms (each operates on a mutable node list)
// ---------------------------------------------------------------------------

function deduplicateStepRunSpans(nodes: OtelSpanTree[]): void {
  const byStepRunId = new Map<
    string,
    { sdk?: OtelSpanTree; engine?: OtelSpanTree }
  >();

  for (const node of nodes) {
    if (node.spanName !== SPAN.START_STEP_RUN) {
      continue;
    }
    const stepRunId = getStepRunId(node);
    if (!stepRunId) {
      continue;
    }

    const entry = byStepRunId.get(stepRunId) ?? {};
    if (isEngineSpan(node)) {
      entry.engine = node;
    } else {
      entry.sdk = node;
    }
    byStepRunId.set(stepRunId, entry);
  }

  const toRemove = new Set<string>();
  for (const { sdk, engine } of byStepRunId.values()) {
    if (sdk && engine) {
      toRemove.add(engine.spanId);
    }
  }
  removeBySpanIds(nodes, toRemove);

  for (const node of nodes) {
    deduplicateStepRunSpans(node.children);
  }
}

function mergeQueuedSpans(nodes: OtelSpanTree[]): void {
  const queuedByStepRunId = new Map<string, OtelSpanTree>();
  for (const node of nodes) {
    const id = getStepRunId(node);
    if (node.spanName === SPAN.ENGINE_QUEUED && id) {
      queuedByStepRunId.set(id, node);
    }
  }

  if (queuedByStepRunId.size > 0) {
    const toRemove = new Set<string>();
    for (const node of nodes) {
      if (node.spanName === SPAN.START_STEP_RUN) {
        const stepRunId = getStepRunId(node);
        if (stepRunId && queuedByStepRunId.has(stepRunId)) {
          node.queuedPhase = queuedByStepRunId.get(stepRunId);
          toRemove.add(node.queuedPhase!.spanId);
        }
      }
    }
    removeBySpanIds(nodes, toRemove);
  }

  for (const node of nodes) {
    mergeQueuedSpans(node.children);
  }
}

function suppressStandaloneQueuedSpans(nodes: OtelSpanTree[]): void {
  removeByPredicate(nodes, (n) => n.spanName === SPAN.ENGINE_QUEUED);

  for (const node of nodes) {
    suppressStandaloneQueuedSpans(node.children);
  }
}

function synthesizeInProgressSpans(
  nodes: OtelSpanTree[],
  spanIdLookup: { has(key: string): boolean },
): void {
  const stepRunIds = new Set<string>();
  for (const node of nodes) {
    if (node.spanName === SPAN.START_STEP_RUN) {
      const id = getStepRunId(node);
      if (id) {
        stepRunIds.add(id);
      }
    }
  }

  const parentCounts = countParents(nodes);

  let write = 0;
  for (let read = 0; read < nodes.length; read++) {
    const node = nodes[read];
    const stepRunId = getStepRunId(node);

    if (
      node.spanName !== SPAN.ENGINE_QUEUED ||
      !stepRunId ||
      stepRunIds.has(stepRunId)
    ) {
      nodes[write++] = nodes[read];
      continue;
    }

    if (
      node.parentSpanId &&
      !spanIdLookup.has(node.parentSpanId) &&
      (parentCounts.get(node.parentSpanId) ?? 0) <= 1
    ) {
      continue;
    }

    const qEndMs = new Date(node.createdAt).getTime() + node.durationNs / 1e6;

    nodes[write++] = makeStepRunSpan(
      `__synthetic_running_${stepRunId}__`,
      node.parentSpanId,
      new Date(qEndMs).toISOString(),
      { ...node.spanAttributes! },
      { queuedPhase: node, inProgress: true },
    );
  }
  nodes.length = write;

  for (const node of nodes) {
    synthesizeInProgressSpans(node.children, spanIdLookup);
  }
}

function buildStepRunIndex(
  nodes: OtelSpanTree[],
  index: Map<string, OtelSpanTree>,
): void {
  for (const node of nodes) {
    if (node.spanName === SPAN.START_STEP_RUN) {
      const id = getStepRunId(node);
      if (id) {
        index.set(id, node);
      }
    }
    buildStepRunIndex(node.children, index);
  }
}

function reparentOrphans(rootSpans: OtelSpanTree[]): void {
  const stepRunIndex = new Map<string, OtelSpanTree>();
  buildStepRunIndex(rootSpans, stepRunIndex);

  const ancestorIds = new Set<string>();

  const reparentLevel = (nodes: OtelSpanTree[]) => {
    const reparented = new Set<string>();

    for (const orphan of nodes) {
      if (!orphan.parentSpanId) {
        continue;
      }
      const stepRunId = getStepRunId(orphan);
      if (!stepRunId) {
        continue;
      }
      const surrogate = stepRunIndex.get(stepRunId);
      if (
        surrogate &&
        surrogate.spanId !== orphan.spanId &&
        !ancestorIds.has(surrogate.spanId)
      ) {
        surrogate.children.push(orphan);
        reparented.add(orphan.spanId);
      }
    }

    if (reparented.size > 0) {
      removeByPredicate(nodes, (n) => reparented.has(n.spanId));
    }

    for (const node of nodes) {
      ancestorIds.add(node.spanId);
      reparentLevel(node.children);
      ancestorIds.delete(node.spanId);
    }
  };

  reparentLevel(rootSpans);
}

function suppressOrphanedChildWorkflows(
  rootSpans: OtelSpanTree[],
  allSpanIds: { has(key: string): boolean },
): void {
  if (rootSpans.length <= 1) {
    return;
  }

  const parentCounts = countParents(rootSpans);

  removeByPredicate(rootSpans, (node) => {
    if (!node.parentSpanId || allSpanIds.has(node.parentSpanId)) {
      return false;
    }
    const isUniqueOrphan = (parentCounts.get(node.parentSpanId) ?? 0) <= 1;
    const isOrphanRunWorkflow = node.spanName === SPAN.RUN_WORKFLOW;
    return isUniqueOrphan || isOrphanRunWorkflow;
  });
}

// ---------------------------------------------------------------------------
// Task-summary synthesis (tasks without matching spans)
// ---------------------------------------------------------------------------

function collectTaskIds(nodes: OtelSpanTree[], out: Set<string>): void {
  for (const node of nodes) {
    const id = getStepRunId(node);
    if (id) {
      out.add(id);
    }
    collectTaskIds(node.children, out);
  }
}

function markRunningTaskSpans(
  nodes: OtelSpanTree[],
  tasks: TaskSummaryForSynthesis[],
): void {
  const runningIds = new Set<string>();
  for (const task of tasks) {
    if (task.status === "RUNNING") {
      runningIds.add(task.externalId);
    }
  }
  if (runningIds.size === 0) {
    return;
  }

  const walk = (list: OtelSpanTree[]) => {
    for (const node of list) {
      if (node.spanName === SPAN.START_STEP_RUN && isEngineSpan(node)) {
        const id = getStepRunId(node);
        if (id && runningIds.has(id)) {
          node.inProgress = true;
        }
      }
      walk(node.children);
    }
  };
  walk(nodes);
}

const MIN_VALID_TIMESTAMP = new Date("2020-01-01").getTime();

function taskStepAttrs(task: TaskSummaryForSynthesis): Record<string, string> {
  return {
    [ATTR.SPAN_SOURCE]: "engine",
    [ATTR.STEP_RUN_ID]: task.externalId,
    [ATTR.STEP_NAME]: task.displayName,
  };
}

function synthesizePendingTaskSpans(
  nodes: OtelSpanTree[],
  tasks: TaskSummaryForSynthesis[],
  parentSpanId: string | undefined,
): void {
  const taskIdsWithSpans = new Set<string>();
  collectTaskIds(nodes, taskIdsWithSpans);

  for (const task of tasks) {
    if (taskIdsWithSpans.has(task.externalId)) {
      continue;
    }
    if (task.status !== "QUEUED" && task.status !== "RUNNING") {
      continue;
    }

    const taskCreatedMs = new Date(task.createdAt).getTime();
    if (!taskCreatedMs || taskCreatedMs < MIN_VALID_TIMESTAMP) {
      continue;
    }

    const attrs = taskStepAttrs(task);

    if (task.status === "QUEUED") {
      const nowMs = Date.now();
      const queuedDurationMs = Math.max(0, nowMs - taskCreatedMs);
      const queuedPhase = makeQueuedPhase(
        `__synthetic_queued_phase_${task.externalId}__`,
        parentSpanId,
        queuedDurationMs * 1e6,
        task.createdAt,
        attrs,
      );

      nodes.push(
        makeStepRunSpan(
          `__synthetic_queuing_${task.externalId}__`,
          parentSpanId,
          new Date(nowMs).toISOString(),
          attrs,
          { queuedPhase },
        ),
      );
    } else {
      const startedMs = task.startedAt
        ? new Date(task.startedAt).getTime()
        : taskCreatedMs;

      const queuedPhase = makeQueuedPhase(
        `__synthetic_queued_phase_${task.externalId}__`,
        parentSpanId,
        Math.max(0, startedMs - taskCreatedMs) * 1e6,
        task.createdAt,
        attrs,
      );

      nodes.push(
        makeStepRunSpan(
          `__synthetic_running_${task.externalId}__`,
          parentSpanId,
          new Date(startedMs).toISOString(),
          attrs,
          { queuedPhase, inProgress: true },
        ),
      );
    }
  }
}

// ---------------------------------------------------------------------------
// Sorting
// ---------------------------------------------------------------------------

function sortChildrenStable(nodes: OtelSpanTree[]): void {
  nodes.sort((a, b) => {
    const aKey = (a.queuedPhase ?? a).createdAt;
    const bKey = (b.queuedPhase ?? b).createdAt;
    return aKey < bKey ? -1 : aKey > bKey ? 1 : 0;
  });

  for (const node of nodes) {
    if (node.children.length > 1) {
      sortChildrenStable(node.children);
    }
  }
}

// ---------------------------------------------------------------------------
// Workflow-level queued phase
// ---------------------------------------------------------------------------

function attachWorkflowQueuedPhase(
  root: OtelSpanTree,
  timing: WorkflowRunTiming,
): void {
  const queueStartMs = new Date(timing.createdAt).getTime();
  if (Number.isNaN(queueStartMs)) {
    return;
  }

  let queueEndMs: number;
  if (timing.startedAt) {
    const startedAtMs = new Date(timing.startedAt).getTime();
    if (!Number.isNaN(startedAtMs) && startedAtMs >= queueStartMs) {
      queueEndMs = startedAtMs;
    } else {
      const rootStartMs = new Date(root.createdAt).getTime();
      queueEndMs = Number.isNaN(rootStartMs) ? Date.now() : rootStartMs;
    }
  } else {
    const rootStartMs = new Date(root.createdAt).getTime();
    queueEndMs = Number.isNaN(rootStartMs) ? Date.now() : rootStartMs;
  }

  const durationMs = queueEndMs - queueStartMs;
  if (durationMs <= 0) {
    return;
  }

  root.queuedPhase = {
    spanId: "__synthetic_workflow_queued__",
    parentSpanId: root.spanId,
    spanName: SPAN.ENGINE_QUEUED,
    statusCode: OtelStatusCode.OK,
    durationNs: durationMs * 1e6,
    createdAt: timing.createdAt,
    spanAttributes: { [ATTR.SPAN_SOURCE]: "engine" },
    children: [],
  };
}

function computeSubtreeFlags(node: OtelSpanTree): boolean {
  let childHasError = false;
  let childHasInProgress = false;
  for (const child of node.children) {
    if (computeSubtreeFlags(child)) {
      childHasInProgress = true;
    }
    if (child.hasErrorInSubtree) {
      childHasError = true;
    }
  }
  node.hasErrorInSubtree =
    node.statusCode === OtelStatusCode.ERROR || childHasError;
  return node.inProgress === true || childHasInProgress;
}

// ---------------------------------------------------------------------------
// Pipeline: build raw tree from flat spans
// ---------------------------------------------------------------------------

function buildRawTree(
  spans: [
    RelevantOpenTelemetrySpanProperties,
    ...RelevantOpenTelemetrySpanProperties[],
  ],
): { rootSpans: OtelSpanTree[]; spanMap: Map<string, OtelSpanTree> } {
  const spanMap = new Map<string, OtelSpanTree>();

  for (const span of spans) {
    spanMap.set(span.spanId, {
      spanId: span.spanId,
      parentSpanId: span.parentSpanId,
      spanName: span.spanName,
      statusCode: span.statusCode,
      statusMessage: span.statusMessage,
      durationNs: span.durationNs,
      createdAt: span.createdAt,
      spanAttributes: span.spanAttributes,
      children: [],
    });
  }

  const rootSpans: OtelSpanTree[] = [];
  for (const span of spans) {
    const converted = spanMap.get(span.spanId)!;
    if (span.parentSpanId) {
      const parent = spanMap.get(span.parentSpanId);
      if (parent) {
        parent.children.push(converted);
      } else {
        rootSpans.push(converted);
      }
    } else {
      rootSpans.push(converted);
    }
  }

  invariant(rootSpans.length > 0, "Must have at least one root span");
  return { rootSpans, spanMap };
}

// ---------------------------------------------------------------------------
// Pipeline: wrap multiple roots into a single synthetic root
// ---------------------------------------------------------------------------

function wrapMultipleRoots(
  rootSpans: OtelSpanTree[],
  workflowRunTiming?: WorkflowRunTiming,
): OtelSpanTree[] {
  if (rootSpans.length <= 1) {
    if (workflowRunTiming && rootSpans.length > 0) {
      attachWorkflowQueuedPhase(rootSpans[0], workflowRunTiming);
    }
    for (const root of rootSpans) {
      if (computeSubtreeFlags(root)) {
        root.inProgress = true;
      }
    }
    return rootSpans;
  }

  let earliestStart = Infinity;
  let latestEnd = -Infinity;
  for (const s of rootSpans) {
    const startMs = new Date(s.createdAt).getTime();
    if (startMs < earliestStart) {
      earliestStart = startMs;
    }
    const endMs = startMs + s.durationNs / 1e6;
    if (endMs > latestEnd) {
      latestEnd = endMs;
    }
  }

  const hasError = rootSpans.some((s) => s.statusCode === OtelStatusCode.ERROR);

  const actionId = rootSpans
    .map((s) => s.spanAttributes?.[ATTR.ACTION_ID])
    .find((id) => id?.includes(":"));
  const workflowName = actionId ? actionId.split(":")[0] : undefined;

  const syntheticRoot = makeSyntheticRoot(rootSpans, {
    statusCode: hasError ? OtelStatusCode.ERROR : OtelStatusCode.OK,
    durationNs: (latestEnd - earliestStart) * 1e6,
    createdAt: new Date(earliestStart).toISOString(),
    spanAttributes: {
      [ATTR.INSTRUMENTOR]: "hatchet",
      ...(workflowName && { [ATTR.WORKFLOW_NAME]: workflowName }),
    },
  });

  if (workflowRunTiming) {
    attachWorkflowQueuedPhase(syntheticRoot, workflowRunTiming);
  }

  if (computeSubtreeFlags(syntheticRoot)) {
    syntheticRoot.inProgress = true;
  }
  return [syntheticRoot];
}

// ---------------------------------------------------------------------------
// Pipeline: handle "no spans" path (only task summaries / timing)
// ---------------------------------------------------------------------------

function buildTreeFromTaskSummaries(
  tasks: TaskSummaryForSynthesis[] | undefined,
  workflowRunTiming?: WorkflowRunTiming,
): OtelSpanTree[] {
  const rootSpans: OtelSpanTree[] = [];

  if (tasks?.length) {
    synthesizePendingTaskSpans(rootSpans, tasks, undefined);
  }

  if (rootSpans.length === 0) {
    if (workflowRunTiming) {
      const syntheticRoot = makeSyntheticRoot([], {
        createdAt: new Date().toISOString(),
      });
      attachWorkflowQueuedPhase(syntheticRoot, workflowRunTiming);
      if (syntheticRoot.queuedPhase) {
        computeSubtreeFlags(syntheticRoot);
        return [syntheticRoot];
      }
    }
    return [];
  }

  return wrapMultipleRoots(rootSpans, workflowRunTiming);
}

// ---------------------------------------------------------------------------
// Main export
// ---------------------------------------------------------------------------

export const convertOtelSpansToOtelSpanTree = (
  spans:
    | [
        RelevantOpenTelemetrySpanProperties,
        ...RelevantOpenTelemetrySpanProperties[],
      ]
    | undefined,
  tasks?: TaskSummaryForSynthesis[],
  workflowRunTiming?: WorkflowRunTiming,
  options?: ConvertOptions,
): OtelSpanTree[] => {
  if (!spans) {
    return buildTreeFromTaskSummaries(tasks, workflowRunTiming);
  }

  const enableTraceInProgressSynthesis =
    options?.enableTraceInProgressSynthesis ?? true;

  const { rootSpans, spanMap } = buildRawTree(spans);

  deduplicateStepRunSpans(rootSpans);
  mergeQueuedSpans(rootSpans);

  if (enableTraceInProgressSynthesis) {
    synthesizeInProgressSpans(rootSpans, spanMap);
  } else {
    suppressStandaloneQueuedSpans(rootSpans);
  }

  reparentOrphans(rootSpans);
  suppressOrphanedChildWorkflows(rootSpans, spanMap);

  if (tasks?.length) {
    const parentSpanId =
      rootSpans.length === 1 ? rootSpans[0].spanId : undefined;
    const targetNodes =
      rootSpans.length === 1 ? rootSpans[0].children : rootSpans;
    synthesizePendingTaskSpans(targetNodes, tasks, parentSpanId);
    markRunningTaskSpans(targetNodes, tasks);
  }

  sortChildrenStable(rootSpans);

  return wrapMultipleRoots(rootSpans, workflowRunTiming);
};
