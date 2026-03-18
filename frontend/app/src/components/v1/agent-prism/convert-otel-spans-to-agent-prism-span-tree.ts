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

function synthesizeInProgressSpans(nodes: OtelSpanTree[]): void {
  const stepRunIds = new Set<string>();
  for (const node of nodes) {
    if (node.spanName === 'hatchet.start_step_run') {
      const id = node.spanAttributes?.['hatchet.step_run_id'];
      if (id) {
        stepRunIds.add(id);
      }
    }
  }

  for (let i = nodes.length - 1; i >= 0; i--) {
    const node = nodes[i];
    if (
      node.spanName === 'hatchet.engine.queued' &&
      node.spanAttributes?.['hatchet.step_run_id'] &&
      !stepRunIds.has(node.spanAttributes['hatchet.step_run_id'])
    ) {
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
    synthesizeInProgressSpans(node.children);
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
      nodes.push({
        spanId: `__synthetic_queuing_${task.externalId}__`,
        parentSpanId,
        spanName: 'hatchet.start_step_run',
        statusCode: OtelStatusCode.UNSET,
        durationNs: 0,
        createdAt: task.createdAt,
        spanAttributes: {
          'hatchet.span_source': 'engine',
          'hatchet.step_run_id': task.externalId,
          'hatchet.step_name': task.displayName,
        },
        children: [],
        inProgress: true,
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

export const convertOtelSpansToOtelSpanTree = (
  spans:
    | [
        RelevantOpenTelemetrySpanProperties,
        ...RelevantOpenTelemetrySpanProperties[],
      ]
    | undefined,
  tasks?: TaskSummaryForSynthesis[],
): OtelSpanTree[] => {
  if (!spans) {
    const rootSpans: OtelSpanTree[] = [];
    if (tasks?.length) {
      synthesizePendingTaskSpans(rootSpans, tasks, undefined);
    }
    if (rootSpans.length === 0) {
      return [];
    }
    if (rootSpans.length > 1) {
      const earliestStart = Math.min(
        ...rootSpans.map((s) => new Date(s.createdAt).getTime()),
      );
      return [
        {
          spanId: '__synthetic_workflow_start__',
          parentSpanId: undefined,
          spanName: 'hatchet.start_workflow',
          statusCode: OtelStatusCode.UNSET,
          durationNs: 0,
          createdAt: new Date(earliestStart).toISOString(),
          spanAttributes: { instrumentor: 'hatchet' },
          children: rootSpans,
          inProgress: true,
        },
      ];
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

  deduplicateStepRunSpans(rootSpans);
  mergeQueuedSpans(rootSpans);
  synthesizeInProgressSpans(rootSpans);

  if (tasks?.length) {
    const parentSpanId =
      rootSpans.length === 1 ? rootSpans[0].spanId : undefined;
    const targetNodes =
      rootSpans.length === 1 ? rootSpans[0].children : rootSpans;
    synthesizePendingTaskSpans(targetNodes, tasks, parentSpanId);
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

    return [syntheticRoot];
  }

  return rootSpans;
};
