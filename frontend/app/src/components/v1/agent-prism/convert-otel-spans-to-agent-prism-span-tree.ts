import type {
  OtelSpanTree,
  RelevantOpenTelemetrySpanProperties,
} from './span-tree-type';
import { OtelStatusCode } from '@/lib/api/generated/data-contracts';
import invariant from 'tiny-invariant';

export interface TaskSummaryForSynthesis {
  externalId: string;
  displayName: string;
  status: string;
  createdAt: string;
  startedAt?: string;
}

function deduplicateStepRunSpans(nodes: OtelSpanTree[]): void {
  const byStepRunId = new Map<
    string,
    { sdk?: OtelSpanTree; engine?: OtelSpanTree }
  >();

  for (const node of nodes) {
    if (node.spanName !== 'hatchet.start_step_run') {
      continue;
    }
    const stepRunId = node.spanAttributes?.['hatchet.step_run_id'];
    if (!stepRunId) {
      continue;
    }

    const entry = byStepRunId.get(stepRunId) ?? {};
    if (node.spanAttributes?.['hatchet.span_source'] === 'engine') {
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

  if (toRemove.size > 0) {
    for (let i = nodes.length - 1; i >= 0; i--) {
      if (toRemove.has(nodes[i].spanId)) {
        nodes.splice(i, 1);
      }
    }
  }

  for (const node of nodes) {
    deduplicateStepRunSpans(node.children);
  }
}

function mergeQueuedSpans(nodes: OtelSpanTree[]): void {
  const queuedByStepRunId = new Map<string, OtelSpanTree>();
  for (const node of nodes) {
    if (
      node.spanName === 'hatchet.engine.queued' &&
      node.spanAttributes?.['hatchet.step_run_id']
    ) {
      queuedByStepRunId.set(node.spanAttributes['hatchet.step_run_id'], node);
    }
  }

  if (queuedByStepRunId.size > 0) {
    const toRemove = new Set<string>();
    for (const node of nodes) {
      if (node.spanName === 'hatchet.start_step_run') {
        const stepRunId = node.spanAttributes?.['hatchet.step_run_id'];
        if (stepRunId && queuedByStepRunId.has(stepRunId)) {
          node.queuedPhase = queuedByStepRunId.get(stepRunId);
          toRemove.add(node.queuedPhase!.spanId);
        }
      }
    }
    for (let i = nodes.length - 1; i >= 0; i--) {
      if (toRemove.has(nodes[i].spanId)) {
        nodes.splice(i, 1);
      }
    }
  }

  for (const node of nodes) {
    mergeQueuedSpans(node.children);
  }
}

function suppressStandaloneQueuedSpans(nodes: OtelSpanTree[]): void {
  for (let i = nodes.length - 1; i >= 0; i--) {
    if (nodes[i].spanName === 'hatchet.engine.queued') {
      nodes.splice(i, 1);
      continue;
    }

    suppressStandaloneQueuedSpans(nodes[i].children);
  }
}

function synthesizeInProgressSpans(
  nodes: OtelSpanTree[],
  allSpanIds: Set<string>,
): void {
  const stepRunIds = new Set<string>();
  for (const node of nodes) {
    if (node.spanName === 'hatchet.start_step_run') {
      const id = node.spanAttributes?.['hatchet.step_run_id'];
      if (id) {
        stepRunIds.add(id);
      }
    }
  }

  // Count siblings sharing each parentSpanId. If multiple siblings share
  // the same missing parent, it's the implicit trace root → keep the span.
  // If the parent is unique and missing, it's a child-workflow orphan → drop.
  const parentCounts = new Map<string, number>();
  for (const node of nodes) {
    if (node.parentSpanId) {
      parentCounts.set(
        node.parentSpanId,
        (parentCounts.get(node.parentSpanId) ?? 0) + 1,
      );
    }
  }

  for (let i = nodes.length - 1; i >= 0; i--) {
    const node = nodes[i];
    if (
      node.spanName === 'hatchet.engine.queued' &&
      node.spanAttributes?.['hatchet.step_run_id'] &&
      !stepRunIds.has(node.spanAttributes['hatchet.step_run_id'])
    ) {
      if (
        node.parentSpanId &&
        !allSpanIds.has(node.parentSpanId) &&
        (parentCounts.get(node.parentSpanId) ?? 0) <= 1
      ) {
        nodes.splice(i, 1);
        continue;
      }

      const stepRunId = node.spanAttributes['hatchet.step_run_id'];
      const qEndMs = new Date(node.createdAt).getTime() + node.durationNs / 1e6;

      nodes[i] = {
        spanId: `__synthetic_running_${stepRunId}__`,
        parentSpanId: node.parentSpanId,
        spanName: 'hatchet.start_step_run',
        statusCode: OtelStatusCode.UNSET,
        durationNs: 0,
        createdAt: new Date(qEndMs).toISOString(),
        spanAttributes: { ...node.spanAttributes },
        children: [],
        queuedPhase: node,
        inProgress: true,
      };
    }
  }

  for (const node of nodes) {
    synthesizeInProgressSpans(node.children, allSpanIds);
  }
}

function buildStepRunIndex(
  nodes: OtelSpanTree[],
  index: Map<string, OtelSpanTree>,
): void {
  for (const node of nodes) {
    if (
      node.spanName === 'hatchet.start_step_run' &&
      node.spanAttributes?.['hatchet.step_run_id']
    ) {
      index.set(node.spanAttributes['hatchet.step_run_id'], node);
    }
    buildStepRunIndex(node.children, index);
  }
}

function reparentOrphans(rootSpans: OtelSpanTree[]): void {
  const stepRunIndex = new Map<string, OtelSpanTree>();
  buildStepRunIndex(rootSpans, stepRunIndex);

  const toRemove = new Set<number>();

  for (let i = 0; i < rootSpans.length; i++) {
    const orphan = rootSpans[i];
    if (!orphan.parentSpanId) {
      continue;
    }

    const stepRunId = orphan.spanAttributes?.['hatchet.step_run_id'];
    if (!stepRunId) {
      continue;
    }

    const surrogate = stepRunIndex.get(stepRunId);
    if (surrogate && surrogate.spanId !== orphan.spanId) {
      surrogate.children.push(orphan);
      toRemove.add(i);
    }
  }

  if (toRemove.size > 0) {
    for (let i = rootSpans.length - 1; i >= 0; i--) {
      if (toRemove.has(i)) {
        rootSpans.splice(i, 1);
      }
    }
  }

  for (const node of rootSpans) {
    reparentOrphans(node.children);
  }
}

// Remove root-level orphan spans whose parent is missing from the span data.
// Two cases:
//   1. Unique missing parent → child-workflow engine span whose run_workflow
//      parent hasn't arrived yet.
//   2. hatchet.run_workflow with missing parent → SDK run_workflow spans that
//      arrived before their dag-confirmation step_run. These are suppressed
//      even when multiple siblings share the same missing parent, because
//      they should only appear nested under their step_run. They'll reappear
//      correctly on the next poll when the step_run span is present.
function suppressOrphanedChildWorkflows(
  rootSpans: OtelSpanTree[],
  allSpanIds: Set<string>,
): void {
  if (rootSpans.length <= 1) {
    return;
  }

  const parentCounts = new Map<string, number>();
  for (const node of rootSpans) {
    if (node.parentSpanId) {
      parentCounts.set(
        node.parentSpanId,
        (parentCounts.get(node.parentSpanId) ?? 0) + 1,
      );
    }
  }

  for (let i = rootSpans.length - 1; i >= 0; i--) {
    const node = rootSpans[i];
    if (!node.parentSpanId || allSpanIds.has(node.parentSpanId)) {
      continue;
    }

    const isUniqueOrphan =
      (parentCounts.get(node.parentSpanId) ?? 0) <= 1;
    const isOrphanRunWorkflow = node.spanName === 'hatchet.run_workflow';

    if (isUniqueOrphan || isOrphanRunWorkflow) {
      rootSpans.splice(i, 1);
    }
  }
}

function collectTaskIds(nodes: OtelSpanTree[], out: Set<string>): void {
  for (const node of nodes) {
    const id = node.spanAttributes?.['hatchet.step_run_id'];
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
    if (task.status === 'RUNNING') {
      runningIds.add(task.externalId);
    }
  }
  if (runningIds.size === 0) {
    return;
  }

  const walk = (list: OtelSpanTree[]) => {
    for (const node of list) {
      if (
        node.spanName === 'hatchet.start_step_run' &&
        node.spanAttributes?.['hatchet.step_run_id'] &&
        runningIds.has(node.spanAttributes['hatchet.step_run_id']) &&
        node.spanAttributes?.['hatchet.span_source'] === 'engine'
      ) {
        node.inProgress = true;
      }
      walk(node.children);
    }
  };
  walk(nodes);
}

function stableSortKey(node: OtelSpanTree): number {
  if (node.queuedPhase) {
    return new Date(node.queuedPhase.createdAt).getTime();
  }
  return new Date(node.createdAt).getTime();
}

function sortChildrenStable(nodes: OtelSpanTree[]): void {
  nodes.sort((a, b) => stableSortKey(a) - stableSortKey(b));
  for (const node of nodes) {
    if (node.children.length > 1) {
      sortChildrenStable(node.children);
    }
  }
}

function synthesizePendingTaskSpans(
  nodes: OtelSpanTree[],
  tasks: TaskSummaryForSynthesis[],
  parentSpanId: string | undefined,
): void {
  const taskIdsWithSpans = new Set<string>();
  collectTaskIds(nodes, taskIdsWithSpans);

  const MIN_VALID_TIMESTAMP = new Date('2020-01-01').getTime();

  for (const task of tasks) {
    if (taskIdsWithSpans.has(task.externalId)) {
      continue;
    }
    if (task.status !== 'QUEUED' && task.status !== 'RUNNING') {
      continue;
    }

    const taskCreatedMs = new Date(task.createdAt).getTime();
    if (!taskCreatedMs || taskCreatedMs < MIN_VALID_TIMESTAMP) {
      continue;
    }

    if (task.status === 'QUEUED') {
      const nowMs = Date.now();
      const queuedDurationMs = Math.max(0, nowMs - taskCreatedMs);
      const queuedPhase: OtelSpanTree = {
        spanId: `__synthetic_queued_phase_${task.externalId}__`,
        parentSpanId,
        spanName: 'hatchet.engine.queued',
        statusCode: OtelStatusCode.OK,
        durationNs: queuedDurationMs * 1e6,
        createdAt: task.createdAt,
        spanAttributes: {
          'hatchet.span_source': 'engine',
          'hatchet.step_run_id': task.externalId,
          'hatchet.step_name': task.displayName,
        },
        children: [],
      };

      nodes.push({
        spanId: `__synthetic_queuing_${task.externalId}__`,
        parentSpanId,
        spanName: 'hatchet.start_step_run',
        statusCode: OtelStatusCode.UNSET,
        durationNs: 0,
        createdAt: new Date(nowMs).toISOString(),
        spanAttributes: {
          'hatchet.span_source': 'engine',
          'hatchet.step_run_id': task.externalId,
          'hatchet.step_name': task.displayName,
        },
        children: [],
        queuedPhase,
      });
    } else if (task.status === 'RUNNING') {
      const startedMs = task.startedAt
        ? new Date(task.startedAt).getTime()
        : taskCreatedMs;

      const queuedPhase: OtelSpanTree = {
        spanId: `__synthetic_queued_phase_${task.externalId}__`,
        parentSpanId,
        spanName: 'hatchet.engine.queued',
        statusCode: OtelStatusCode.OK,
        durationNs: Math.max(0, startedMs - taskCreatedMs) * 1e6,
        createdAt: task.createdAt,
        spanAttributes: {
          'hatchet.span_source': 'engine',
          'hatchet.step_run_id': task.externalId,
          'hatchet.step_name': task.displayName,
        },
        children: [],
      };

      nodes.push({
        spanId: `__synthetic_running_${task.externalId}__`,
        parentSpanId,
        spanName: 'hatchet.start_step_run',
        statusCode: OtelStatusCode.UNSET,
        durationNs: 0,
        createdAt: new Date(startedMs).toISOString(),
        spanAttributes: {
          'hatchet.span_source': 'engine',
          'hatchet.step_run_id': task.externalId,
          'hatchet.step_name': task.displayName,
        },
        children: [],
        queuedPhase,
        inProgress: true,
      });
    }
  }
}

export type WorkflowRunTiming = {
  createdAt: string;
  startedAt?: string;
};

type ConvertOptions = {
  enableTraceInProgressSynthesis?: boolean;
};

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
    spanId: '__synthetic_workflow_queued__',
    parentSpanId: root.spanId,
    spanName: 'hatchet.engine.queued',
    statusCode: OtelStatusCode.OK,
    durationNs: durationMs * 1e6,
    createdAt: timing.createdAt,
    spanAttributes: { 'hatchet.span_source': 'engine' },
    children: [],
  };
}

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
  const enableTraceInProgressSynthesis =
    options?.enableTraceInProgressSynthesis ?? true;

  if (!spans) {
    const rootSpans: OtelSpanTree[] = [];
    if (tasks?.length) {
      synthesizePendingTaskSpans(rootSpans, tasks, undefined);
    }
    if (rootSpans.length === 0) {
      if (workflowRunTiming) {
        const syntheticRoot: OtelSpanTree = {
          spanId: '__synthetic_workflow_start__',
          parentSpanId: undefined,
          spanName: 'hatchet.start_workflow',
          statusCode: OtelStatusCode.UNSET,
          durationNs: 0,
          createdAt: new Date().toISOString(),
          spanAttributes: { instrumentor: 'hatchet' },
          children: [],
          inProgress: true,
        };

        attachWorkflowQueuedPhase(syntheticRoot, workflowRunTiming);
        if (syntheticRoot.queuedPhase) {
          return [syntheticRoot];
        }
      }

      return [];
    }
    if (rootSpans.length > 1) {
      const earliestStart = Math.min(
        ...rootSpans.map((s) => new Date(s.createdAt).getTime()),
      );
      const syntheticRoot: OtelSpanTree = {
        spanId: '__synthetic_workflow_start__',
        parentSpanId: undefined,
        spanName: 'hatchet.start_workflow',
        statusCode: OtelStatusCode.UNSET,
        durationNs: 0,
        createdAt: new Date(earliestStart).toISOString(),
        spanAttributes: { instrumentor: 'hatchet' },
        children: rootSpans,
        inProgress: true,
      };
      if (workflowRunTiming) {
        attachWorkflowQueuedPhase(syntheticRoot, workflowRunTiming);
      }
      return [syntheticRoot];
    }
    if (workflowRunTiming) {
      attachWorkflowQueuedPhase(rootSpans[0], workflowRunTiming);
    }
    return rootSpans;
  }

  const spanMap = new Map<string, OtelSpanTree>();
  const rootSpans: OtelSpanTree[] = [];

  spans.forEach((span) => {
    spanMap.set(span.spanId, {
      spanId: span.spanId,
      parentSpanId: span.parentSpanId,
      spanName: span.spanName,
      statusCode: span.statusCode,
      durationNs: span.durationNs,
      createdAt: span.createdAt,
      spanAttributes: span.spanAttributes,
      children: [],
    });
  });

  spans.forEach((span) => {
    const converted = spanMap.get(span.spanId)!;
    const parentSpanId = span.parentSpanId;
    if (parentSpanId) {
      const parent = spanMap.get(parentSpanId);
      if (parent) {
        parent.children.push(converted);
      } else {
        rootSpans.push(converted);
      }
    } else {
      rootSpans.push(converted);
    }
  });

  invariant(rootSpans.length > 0, 'Must have at least one root span');

  const allSpanIds = new Set(spanMap.keys());

  deduplicateStepRunSpans(rootSpans);
  mergeQueuedSpans(rootSpans);
  if (enableTraceInProgressSynthesis) {
    synthesizeInProgressSpans(rootSpans, allSpanIds);
  } else {
    suppressStandaloneQueuedSpans(rootSpans);
  }
  reparentOrphans(rootSpans);
  suppressOrphanedChildWorkflows(rootSpans, allSpanIds);

  if (tasks?.length) {
    const parentSpanId =
      rootSpans.length === 1 ? rootSpans[0].spanId : undefined;
    const targetNodes =
      rootSpans.length === 1 ? rootSpans[0].children : rootSpans;
    synthesizePendingTaskSpans(targetNodes, tasks, parentSpanId);
    markRunningTaskSpans(targetNodes, tasks);
  }

  sortChildrenStable(rootSpans);

  const hasInProgress = (nodes: OtelSpanTree[]): boolean =>
    nodes.some((n) => n.inProgress || hasInProgress(n.children));

  if (rootSpans.length === 1 && hasInProgress(rootSpans[0].children)) {
    rootSpans[0].inProgress = true;
  }

  if (rootSpans.length > 1) {
    const earliestStart = Math.min(
      ...rootSpans.map((s) => new Date(s.createdAt).getTime()),
    );
    const latestEnd = Math.max(
      ...rootSpans.map(
        (s) => new Date(s.createdAt).getTime() + s.durationNs / 1e6,
      ),
    );
    const durationNs = (latestEnd - earliestStart) * 1e6;

    const hasError = rootSpans.some(
      (s) => s.statusCode === OtelStatusCode.ERROR,
    );

    const actionId = rootSpans
      .map((s) => s.spanAttributes?.['hatchet.action_id'])
      .find((id) => id?.includes(':'));
    const workflowName = actionId ? actionId.split(':')[0] : undefined;

    const anyInProgress = hasInProgress(rootSpans);
    const syntheticRoot: OtelSpanTree = {
      spanId: '__synthetic_workflow_start__',
      parentSpanId: undefined,
      spanName: 'hatchet.start_workflow',
      statusCode: hasError ? OtelStatusCode.ERROR : OtelStatusCode.OK,
      durationNs,
      createdAt: new Date(earliestStart).toISOString(),
      spanAttributes: {
        instrumentor: 'hatchet',
        ...(workflowName && { 'hatchet.workflow_name': workflowName }),
      },
      children: rootSpans,
      inProgress: anyInProgress,
    };

    if (workflowRunTiming) {
      attachWorkflowQueuedPhase(syntheticRoot, workflowRunTiming);
    }

    return [syntheticRoot];
  }

  if (workflowRunTiming && rootSpans.length > 0) {
    attachWorkflowQueuedPhase(rootSpans[0], workflowRunTiming);
  }

  return rootSpans;
};
