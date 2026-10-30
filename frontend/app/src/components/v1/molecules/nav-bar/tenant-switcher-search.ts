/**
 * Search helpers for the tenant switcher.
 *
 * Matching ignores case and separators (`-`, `_`, `.`, `/`, `:`, spaces), and the
 * query's words can appear in any order: "prod eu" matches "production-eu".
 * Every word must match at least one of the searchable fields.
 */

export type HighlightRange = { start: number; end: number };

type Normalized = {
  text: string;
  // indexMap[i] is the index in the original string of normalized char i.
  indexMap: number[];
};

const SEPARATOR = /[^\p{L}\p{N}]/u;

function normalizeWithMap(value: string): Normalized {
  let text = '';
  const indexMap: number[] = [];
  for (let i = 0; i < value.length; i++) {
    const ch = value[i];
    if (SEPARATOR.test(ch)) {
      continue;
    }
    text += ch.toLowerCase();
    indexMap.push(i);
  }
  return { text, indexMap };
}

export function normalizeSearchText(value: string): string {
  return normalizeWithMap(value).text;
}

export function parseSearchQuery(query: string): string[] {
  return query
    .split(/\s+/)
    .map(normalizeSearchText)
    .filter((word) => word.length > 0);
}

export function matchesSearch(
  words: string[],
  fields: Array<string | undefined>,
): boolean {
  if (words.length === 0) {
    return true;
  }
  const normalizedFields = fields
    .filter((field): field is string => !!field)
    .map(normalizeSearchText);
  return words.every((word) =>
    normalizedFields.some((field) => field.includes(word)),
  );
}

/**
 * Ranges of `value` (in original string indices) matched by any query word.
 * Overlapping and adjacent ranges are merged.
 */
export function getHighlightRanges(
  value: string,
  words: string[],
): HighlightRange[] {
  if (words.length === 0 || !value) {
    return [];
  }

  const { text, indexMap } = normalizeWithMap(value);
  const ranges: HighlightRange[] = [];

  for (const word of words) {
    let from = 0;
    while (from <= text.length - word.length) {
      const at = text.indexOf(word, from);
      if (at < 0) {
        break;
      }
      ranges.push({
        start: indexMap[at],
        end: indexMap[at + word.length - 1] + 1,
      });
      from = at + word.length;
    }
  }

  ranges.sort((a, b) => a.start - b.start);

  const merged: HighlightRange[] = [];
  for (const range of ranges) {
    const last = merged[merged.length - 1];
    if (last && range.start <= last.end) {
      last.end = Math.max(last.end, range.end);
    } else {
      merged.push({ ...range });
    }
  }
  return merged;
}
