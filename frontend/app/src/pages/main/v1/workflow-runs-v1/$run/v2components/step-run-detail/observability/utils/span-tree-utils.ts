import {
  isStartStepRunSpan,
  SPAN,
  ATTR,
} from '@/components/v1/agent-prism/convert-otel-spans-to-agent-prism-span-tree';
import type { OtelSpanTree } from '@/components/v1/agent-prism/span-tree-type';
import { OtelStatusCode } from '@/lib/api/generated/data-contracts';

export function hasErrorInTree(span: OtelSpanTree): boolean {
  if (span.hasErrorInSubtree !== undefined) {
    return span.hasErrorInSubtree;
  }
  if (span.statusCode === OtelStatusCode.ERROR) {
    return true;
  }
  return span.children.some(hasErrorInTree);
}

export function isEngineSpan(span: OtelSpanTree): boolean {
  return span.spanAttributes?.['hatchet.span_source'] === 'engine';
}

export function hasOnlyEngineSpans(trees: OtelSpanTree[]): boolean {
  const stack = [...trees];
  let realSpanCount = 0;
  let hasOkSpan = false;

  while (stack.length > 0) {
    const span = stack.pop()!;
    stack.push(...span.children);

    if (span.spanId.startsWith('__synthetic_')) {
      continue;
    }

    realSpanCount++;

    if (!isEngineSpan(span)) {
      return false;
    }
    if (span.statusCode === OtelStatusCode.OK) {
      hasOkSpan = true;
    }
  }

  return realSpanCount > 0 && hasOkSpan;
}

export function isQueuedOnlyRoot(span: OtelSpanTree): boolean {
  if (!span.spanId.startsWith('__synthetic_') || !span.queuedPhase) {
    return false;
  }
  const hasRunningChild = span.children.some((c) => c.inProgress);
  return !hasRunningChild && !span.inProgress;
}

export function isQueuedOnly(span: OtelSpanTree): boolean {
  return !!span.queuedPhase && span.durationNs <= 0 && !span.inProgress;
}

/**
 * The SDK instrumentor copies the task run's `hatchet.*` attributes onto every
 * span created inside a task, so a custom span carries the name of the task it
 * ran in.
 */
export function getSpanAttributeLabel(span: OtelSpanTree): string | undefined {
  return (
    span.spanAttributes?.[ATTR.TASK_NAME] ??
    span.spanAttributes?.[ATTR.STEP_NAME]
  );
}

/**
 * The short run id is appended to a workflow-run label because run display names
 * are not unique across concurrent runs. The raw span name stays visible in the
 * detail panel and the tooltip subtitle.
 */
export function getSpanDisplayLabel(span: OtelSpanTree): string {
  if (isStartStepRunSpan(span)) {
    return (
      span.spanAttributes?.[ATTR.TASK_NAME] ??
      span.spanAttributes?.[ATTR.STEP_NAME] ??
      span.spanName
    );
  }

  if (span.spanName === SPAN.ENGINE_WORKFLOW_RUN) {
    const workflowName = span.spanAttributes?.[ATTR.WORKFLOW_NAME];
    if (workflowName) {
      const shortId =
        span.spanAttributes?.[ATTR.WORKFLOW_RUN_ID]?.split('-')[0];
      return shortId ? `${workflowName} (${shortId})` : workflowName;
    }
  }

  return span.spanName;
}

/**
 * Groups stay keyed by the raw span name, so this changes only the header text,
 * not how spans are grouped.
 */
export function getSpanGroupLabel(spanName: string): string {
  switch (spanName) {
    case SPAN.ENGINE_WORKFLOW_RUN:
      return 'workflow runs';
    case SPAN.ENGINE_START_STEP_RUN:
    case SPAN.START_STEP_RUN:
      return 'tasks';
    default:
      return spanName;
  }
}

export function getStableKey(span: OtelSpanTree): string {
  return isStartStepRunSpan(span) &&
    span.spanAttributes?.['hatchet.step_run_id']
    ? span.spanAttributes['hatchet.step_run_id']
    : span.spanId;
}

export function getSpanColor(span: OtelSpanTree): string {
  if (span.inProgress) {
    return 'bg-yellow-500';
  }
  if (isQueuedOnlyRoot(span) || isQueuedOnly(span)) {
    return 'bg-yellow-500';
  }
  if (hasErrorInTree(span)) {
    return 'bg-red-500';
  }
  return 'bg-green-500';
}

export function isQueuedEngineSpan(span: OtelSpanTree): boolean {
  return isEngineSpan(span) && span.spanName === 'hatchet.engine.queued';
}

export function statusLabel(code: string): string {
  switch (code) {
    case OtelStatusCode.OK:
      return 'OK';
    case OtelStatusCode.ERROR:
      return 'Error';
    default:
      return 'Unset';
  }
}

export function effectiveStatusLabel(
  span: OtelSpanTree,
  queuedOnly: boolean,
): string {
  if (queuedOnly) {
    return 'Queued';
  }
  if (span.inProgress) {
    return 'In Progress';
  }
  if (span.statusCode !== OtelStatusCode.ERROR && hasErrorInTree(span)) {
    return 'Error (child)';
  }
  return statusLabel(span.statusCode);
}

export function getBarColor(span: OtelSpanTree): string {
  if (span.inProgress) {
    return 'bg-yellow-500';
  }
  if (isQueuedOnlyRoot(span) || isQueuedOnly(span)) {
    return 'bg-yellow-500';
  }
  if (isEngineSpan(span)) {
    return span.statusCode === OtelStatusCode.ERROR
      ? 'bg-red-500'
      : 'bg-green-500';
  }
  if (hasErrorInTree(span)) {
    return 'bg-red-500';
  }
  return 'bg-green-500';
}
