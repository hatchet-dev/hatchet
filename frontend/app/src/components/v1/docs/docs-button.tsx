import { Button, ButtonProps, ReviewedButtonTemp } from '../ui/button';
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
};

export const DocsButton = ({
  doc,
  label,
  queryParams,
  scrollTo,
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

  return (
    <ReviewedButtonTemp
      onClick={handleClick}
      leftIcon={<BookOpenText className="size-4" />}
      variant="outline"
    >
      <span>{label}</span>
    </ReviewedButtonTemp>
  );
};
