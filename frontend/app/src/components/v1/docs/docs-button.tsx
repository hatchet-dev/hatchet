import { Button } from '../ui/button';
import { BookOpenText } from 'lucide-react';

export type DocPage = {
  href: string;
};

type DocsButtonProps = {
  doc: DocPage;
  label: string;
  queryParams?: Record<string, string>;
  scrollTo?: string;
  variant?: 'button' | 'text';
};

function buildUrl(
  doc: DocPage,
  queryParams?: Record<string, string>,
  scrollTo?: string,
): string {
  const qs = queryParams ? '?' + new URLSearchParams(queryParams).toString() : '';
  const hash = scrollTo ? '#' + scrollTo : '';
  return doc.href + qs + hash;
}

export const DocsButton = ({
  doc,
  label,
  queryParams,
  scrollTo,
  variant = 'button',
}: DocsButtonProps) => {
  const url = buildUrl(doc, queryParams, scrollTo);

  switch (variant) {
    case 'button':
      return (
        <Button asChild leftIcon={<BookOpenText className="size-4" />} variant="outline">
          <a href={url} target="_blank" rel="noreferrer">
            <span>{label}</span>
          </a>
        </Button>
      );
    case 'text':
      return (
        <a
          href={url}
          target="_blank"
          rel="noreferrer"
          className="underline hover:text-gray-900 dark:hover:text-gray-100 hover:cursor-pointer"
        >
          {label}
        </a>
      );
    default:
      const exhaustiveCheck: never = variant;
      throw new Error(`Unhandled variant type: ${exhaustiveCheck}`);
  }
};
