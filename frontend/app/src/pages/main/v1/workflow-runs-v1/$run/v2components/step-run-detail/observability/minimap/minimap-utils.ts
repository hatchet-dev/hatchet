import {
  hasErrorInTree,
  isQueuedOnly,
  isQueuedOnlyRoot,
} from '../utils/span-tree-utils';
import type { SpanMarker } from './minimap-types';
import type { OtelSpanTree } from '@/components/v1/agent-prism/span-tree-type';

export function collectSpanMarkers(
  trees: OtelSpanTree[],
  minMs: number,
  totalMs: number,
  expandedIds?: Set<string>,
): SpanMarker[] {
  const markers: SpanMarker[] = [];

  const traverse = (
    node: OtelSpanTree,
    parentVisible: boolean,
    ancestors: string[],
  ) => {
    if (isQueuedOnlyRoot(node) || isQueuedOnly(node)) {
      return;
    }
    const startMs = new Date(node.createdAt).getTime();
    const pct = totalMs > 0 ? (startMs - minMs) / totalMs : 0;
    markers.push({
      pct: Math.max(0, Math.min(1, pct)),
      statusCode: node.statusCode,
      hasErrorInTree: hasErrorInTree(node),
      inProgress: !!node.inProgress,
      spanName: node.spanName,
      durationMs: node.durationNs / 1_000_000,
      visible: parentVisible,
      span: node,
      ancestorSpanIds: ancestors,
    });
    const childrenVisible =
      parentVisible && (!expandedIds || expandedIds.has(node.spanId));
    const nextAncestors = [...ancestors, node.spanId];
    node.children?.forEach((child) =>
      traverse(child, childrenVisible, nextAncestors),
    );
  };

  trees.forEach((tree) => traverse(tree, true, []));
  return markers.sort((a, b) => a.pct - b.pct);
}

export function pctFromEvent(e: { clientX: number }, el: HTMLElement): number {
  const rect = el.getBoundingClientRect();
  return Math.max(0, Math.min(1, (e.clientX - rect.left) / rect.width));
}
