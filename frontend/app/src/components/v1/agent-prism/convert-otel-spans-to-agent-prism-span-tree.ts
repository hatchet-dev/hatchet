import type {
  OtelSpanTree,
  RelevantOpenTelemetrySpanProperties,
} from './span-tree-type';
import { OtelStatusCode } from '@/lib/api/generated/data-contracts';
import invariant from 'tiny-invariant';

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
          toRemove.add(queuedByStepRunId.get(stepRunId)!.spanId);
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

export const convertOtelSpansToOtelSpanTree = (
  spans: [
    RelevantOpenTelemetrySpanProperties,
    ...RelevantOpenTelemetrySpanProperties[],
  ],
): OtelSpanTree[] => {
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
    };

    return [syntheticRoot];
  }

  return rootSpans;
};
