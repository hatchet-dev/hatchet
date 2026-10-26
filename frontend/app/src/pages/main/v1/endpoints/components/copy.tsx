import source from '../copy.mdx?raw';
import {
  CopyKey,
  CopyParams,
  CopyRef,
  createCopy,
  parseCopy,
  ref,
} from '../lib/copy';
import { cn } from '@/lib/utils';
import { Fragment, ReactNode } from 'react';

const { resolveText, resolveList } = createCopy(parseCopy(source));

/** The copy for key as plain text, for attributes and strings; markdown is left as written. */
export const t = (key: CopyKey, params?: CopyParams): string =>
  resolveText(ref(key, params));

export const textOf = resolveText;
export const listOf = resolveList;

// absolute http(s) and mailto links, or paths on this site (not protocol-relative //host)
const SAFE_HREF = /^(https?:|mailto:|\/(?!\/))/i;

const INLINE = /(`[^`]+`)|(\*\*[^*]+\*\*)|(\[[^\]]+\]\([^)\s]+\))/g;

/** Renders the inline markdown copy may use: `code`, **bold** and [links](url). */
export function renderInline(text: string): ReactNode[] {
  const nodes: ReactNode[] = [];
  let last = 0;

  for (const match of text.matchAll(INLINE)) {
    const [token] = match;
    const at = match.index ?? 0;

    if (at > last) {
      nodes.push(text.slice(last, at));
    }

    if (token.startsWith('`')) {
      nodes.push(
        <code key={at} className="font-mono text-[0.9em]">
          {token.slice(1, -1)}
        </code>,
      );
    } else if (token.startsWith('**')) {
      nodes.push(
        <strong key={at} className="font-semibold">
          {token.slice(2, -2)}
        </strong>,
      );
    } else {
      const [, label, href] = token.match(/^\[([^\]]+)\]\(([^)]+)\)$/) ?? [];

      // copy is ours, but a link is only ever rendered with a scheme that cannot run script
      if (!SAFE_HREF.test(href ?? '')) {
        nodes.push(label);
        last = at + token.length;
        continue;
      }

      nodes.push(
        <a
          key={at}
          href={href}
          target="_blank"
          rel="noreferrer"
          className="text-primary underline underline-offset-2 hover:opacity-80"
        >
          {label}
        </a>,
      );
    }

    last = at + token.length;
  }

  if (last < text.length) {
    nodes.push(text.slice(last));
  }

  return nodes;
}

/** Copy rendered with its inline markdown. Pass a key and params, or a ref. */
export const Copy = (
  props: { k: CopyKey; params?: CopyParams } | { copyRef: CopyRef },
) => {
  const text =
    'copyRef' in props ? resolveText(props.copyRef) : t(props.k, props.params);

  return <>{renderInline(text)}</>;
};

/** Every item of the given copy refs as list items, lists flattened in order. */
export const CopyListItems = ({
  refs,
  className,
}: {
  refs: CopyRef[];
  className?: string;
}) => (
  <>
    {refs.flatMap((r, i) =>
      resolveList(r).map((item, j) => (
        <li key={`${i}-${j}`} className={cn(className)}>
          {renderInline(item)}
        </li>
      )),
    )}
  </>
);

/** Several refs as one paragraph's worth of sentences. */
export const CopySentences = ({ refs }: { refs: CopyRef[] }) => (
  <>
    {refs.map((r, i) => (
      <Fragment key={i}>
        {i > 0 && ' '}
        {renderInline(resolveText(r))}
      </Fragment>
    ))}
  </>
);
