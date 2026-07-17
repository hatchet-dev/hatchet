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

export function isSpanError(span: OtelSpanTree): boolean {
  return isEngineSpan(span)
    ? span.statusCode === OtelStatusCode.ERROR
    : hasErrorInTree(span);
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

export type SpanIdentity = {
  label: string;
  /**
   * Distinguishes same-named sibling rows and stays visible when the label
   * truncates.
   */
  discriminator?: string;
};

/**
 * The Hatchet identity behind a span, shown as a badge next to the span name.
 * Returns undefined for spans with no Hatchet identity, such as application
 * spans.
 */
export function getSpanIdentityParts(
  span: OtelSpanTree,
): SpanIdentity | undefined {
  if (isStartStepRunSpan(span)) {
    const name =
      span.spanAttributes?.[ATTR.TASK_NAME] ??
      span.spanAttributes?.[ATTR.STEP_NAME];
    if (name) {
      const retryCount = Number(span.spanAttributes?.[ATTR.RETRY_COUNT]);
      return {
        label: name,
        discriminator: retryCount > 0 ? `(retry ${retryCount})` : undefined,
      };
    }
    return undefined;
  }

  if (span.spanName === SPAN.ENGINE_WORKFLOW_RUN) {
    const workflowName = span.spanAttributes?.[ATTR.WORKFLOW_NAME];
    if (workflowName) {
      const shortId =
        span.spanAttributes?.[ATTR.WORKFLOW_RUN_ID]?.split('-')[0];
      return {
        label: workflowName,
        discriminator: shortId ? `(${shortId})` : undefined,
      };
    }
    return undefined;
  }

  if (
    span.spanName === SPAN.ENGINE_EVENT ||
    span.spanName === SPAN.ENGINE_EVENT_EMITTED
  ) {
    const eventKey = span.spanAttributes?.[ATTR.EVENT_KEY];
    if (eventKey) {
      const shortId = span.spanAttributes?.[ATTR.EVENT_ID]?.split('-')[0];
      return {
        label: eventKey,
        discriminator: shortId ? `(${shortId})` : undefined,
      };
    }
    return undefined;
  }

  return undefined;
}

export function getSpanIdentityLabel(span: OtelSpanTree): string | undefined {
  const identity = getSpanIdentityParts(span);
  if (!identity) {
    return undefined;
  }
  return identity.discriminator
    ? `${identity.label} ${identity.discriminator}`
    : identity.label;
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
  return isSpanError(span) ? 'bg-red-500' : 'bg-green-500';
}
