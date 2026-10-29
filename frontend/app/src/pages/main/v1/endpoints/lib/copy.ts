import { CopyKey } from './copy-keys';

export type { CopyKey } from './copy-keys';

// A parameter is plain text, or another piece of copy resolved in turn.
export type CopyParams = Record<string, string | number | CopyRef>;

/** A piece of copy to show: its key in copy.mdx, and the values of its [[placeholders]]. */
export type CopyRef = { key: CopyKey; params?: CopyParams };

export const ref = (key: CopyKey, params?: CopyParams): CopyRef => ({
  key,
  params,
});

// One entry of copy.mdx: running text, or a bulleted list.
export type CopyEntry = string | string[];

/**
 * Parses copy.mdx: each `## key` heading starts an entry holding the text up to the next
 * heading. Text lines are joined into one string; an entry whose lines are all "- " bullets is
 * a list, where indented lines continue the bullet above them. Comments and `#` headings are
 * dropped.
 */
export function parseCopy(source: string): Map<string, CopyEntry> {
  const entries = new Map<string, CopyEntry>();
  const lines = source.replace(/\{\/\*[\s\S]*?\*\/\}/g, '').split('\n');

  let key: string | undefined;
  let body: string[] = [];

  const flush = () => {
    if (key === undefined) {
      return;
    }

    const content = body.filter((l) => l.trim() !== '');
    const isList =
      content.length > 0 &&
      content.every((l) => /^- /.test(l) || /^\s+\S/.test(l)) &&
      /^- /.test(content[0]);

    if (isList) {
      const items: string[] = [];

      for (const l of content) {
        if (/^- /.test(l)) {
          items.push(l.slice(2).trim());
        } else {
          items[items.length - 1] += ` ${l.trim()}`;
        }
      }

      entries.set(key, items);
    } else {
      entries.set(key, content.map((l) => l.trim()).join(' '));
    }
  };

  for (const line of lines) {
    const heading = line.match(/^(#{1,6})\s+(.*?)\s*$/);

    if (heading) {
      flush();
      key = heading[1] === '##' ? heading[2] : undefined;
      body = [];
      continue;
    }

    body.push(line);
  }

  flush();

  return entries;
}

/** Looks copy up in parsed entries and fills in its placeholders. */
export function createCopy(entries: Map<string, CopyEntry>) {
  const fill = (text: string, params?: CopyParams): string =>
    text.replace(/\[\[(\w+)\]\]/g, (match, name: string) => {
      const value = params?.[name];

      if (value === undefined) {
        return match;
      }

      return typeof value === 'object' ? resolveText(value) : String(value);
    });

  const lookup = (key: CopyKey): CopyEntry => {
    const entry = entries.get(key);

    if (entry === undefined) {
      // copy.test.ts keeps copy.mdx complete; show the key rather than nothing
      return key;
    }

    return entry;
  };

  /** The copy as one string; a list's items are joined with spaces. */
  const resolveText = ({ key, params }: CopyRef): string => {
    const entry = lookup(key);

    return fill(Array.isArray(entry) ? entry.join(' ') : entry, params);
  };

  /** The copy as list items; running text is a list of one. */
  const resolveList = ({ key, params }: CopyRef): string[] => {
    const entry = lookup(key);

    return (Array.isArray(entry) ? entry : [entry]).map((item) =>
      fill(item, params),
    );
  };

  return { resolveText, resolveList };
}
