import { Button } from '../ui/button';
import { useSidePanel } from '@/hooks/use-side-panel';
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

export const DocsButton = ({
  doc,
  label,
  queryParams,
  scrollTo,
  variant = 'button',
}: DocsButtonProps) => {
  const { open } = useSidePanel();

  const handleClick = () => {
    open({
      type: 'docs',
      content: doc,
      queryParams,
      scrollTo,
    });
  };

  switch (variant) {
    case 'button':
      return (
        <Button
          onClick={handleClick}
          leftIcon={<BookOpenText className="size-4" />}
          variant="outline"
        >
          <span>{label}</span>
        </Button>
      );
    case 'text':
      return (
        <span
          onClick={handleClick}
          className="underline hover:text-gray-900 dark:hover:text-gray-100 hover:cursor-pointer"
        >
          {label}
        </span>
      );
    default:
      const exhaustiveCheck: never = variant;
      throw new Error(`Unhandled variant type: ${exhaustiveCheck}`);
  }
};
