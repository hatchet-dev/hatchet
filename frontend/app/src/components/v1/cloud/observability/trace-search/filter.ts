import type { ParsedTraceQuery } from "./types";
import type { OtelSpanTree } from "@/components/v1/agent-prism/span-tree-type";
import { OtelStatusCode } from "@/lib/api/generated/data-contracts";

export type FilteredSpanTree = OtelSpanTree & {
  matchesFilter: boolean;
  children: FilteredSpanTree[];
};

const STATUS_MAP: Record<string, OtelStatusCode> = {
  ok: OtelStatusCode.OK,
  error: OtelStatusCode.ERROR,
  unset: OtelStatusCode.UNSET,
};

function spanMatchesQuery(
  span: OtelSpanTree,
  query: ParsedTraceQuery,
): boolean {
  if (query.status) {
    const expected = STATUS_MAP[query.status];
    if (span.statusCode !== expected) {
      return false;
    }
  }

  for (const [key, value] of query.attributes) {
    const attrValue = span.spanAttributes?.[key];
    if (attrValue === undefined || attrValue !== value) {
      return false;
    }
  }

  if (query.search) {
    const lower = query.search.toLowerCase();
    const nameMatch = span.spanName.toLowerCase().includes(lower);
    const attrMatch = Object.values(span.spanAttributes ?? {}).some((v) =>
      v.toLowerCase().includes(lower),
    );
    if (!nameMatch && !attrMatch) {
      return false;
    }
  }

  return true;
}

function filterNode(
  node: OtelSpanTree,
  query: ParsedTraceQuery,
): FilteredSpanTree | null {
  const filteredChildren: FilteredSpanTree[] = [];

  for (const child of node.children) {
    const result = filterNode(child, query);
    if (result) {
      filteredChildren.push(result);
    }
  }

  const selfMatches = spanMatchesQuery(node, query);
  const hasMatchingDescendant = filteredChildren.length > 0;

  if (!selfMatches && !hasMatchingDescendant) {
    return null;
  }

  return {
    ...node,
    matchesFilter: selfMatches,
    children: filteredChildren,
  };
}

function hasActiveFilter(query: ParsedTraceQuery): boolean {
  return !!(query.search || query.status || query.attributes.length > 0);
}

export function filterSpanTrees(
  trees: OtelSpanTree[],
  query: ParsedTraceQuery,
): FilteredSpanTree[] {
  if (!hasActiveFilter(query)) {
    return markAllMatching(trees);
  }

  const result: FilteredSpanTree[] = [];
  for (const tree of trees) {
    const filtered = filterNode(tree, query);
    if (filtered) {
      result.push(filtered);
    }
  }
  return result;
}

function markAllMatching(trees: OtelSpanTree[]): FilteredSpanTree[] {
  return trees.map((node) => ({
    ...node,
    matchesFilter: true,
    children: markAllMatching(node.children),
  }));
}
