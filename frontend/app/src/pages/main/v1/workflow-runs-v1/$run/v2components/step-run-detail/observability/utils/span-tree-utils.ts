import { isStartStepRunSpan } from '@/components/v1/agent-prism/convert-otel-spans-to-agent-prism-span-tree';
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

const ENGINE_SPAN_DISPLAY_NAMES: Record<string, string> = {
  'hatchet.engine.queued': 'Queued',
  'hatchet.engine.scheduling': 'Scheduling',
  'hatchet.engine.retry_backoff': 'Retry Backoff',
  'hatchet.engine.workflow_run': 'Workflow Run',
};

// O11Y-FIXME: there is a naming consistency issue on the SDKs
export function getDisplayName(span: OtelSpanTree): string {
  if (ENGINE_SPAN_DISPLAY_NAMES[span.spanName]) {
    return ENGINE_SPAN_DISPLAY_NAMES[span.spanName];
  }
  if (!span.spanName.startsWith('hatchet.')) {
    return span.spanName;
  }
  if (span.spanAttributes?.['hatchet.task_name']) {
    return span.spanAttributes['hatchet.task_name'];
  }
  if (span.spanAttributes?.['hatchet.step_name']) {
    return span.spanAttributes['hatchet.step_name'];
  }
  if (span.spanAttributes?.['hatchet.workflow_name']) {
    return span.spanAttributes['hatchet.workflow_name'];
  }
  const actionId = span.spanAttributes?.['hatchet.action_id'];
  if (actionId?.includes(':')) {
    return actionId.split(':')[0];
  }
  return span.spanName;
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
