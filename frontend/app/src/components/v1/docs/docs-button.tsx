import { Button, ButtonProps } from '../ui/button';
import { useSidePanel } from '@/hooks/use-side-panel';
import { BookOpenText } from 'lucide-react';

export type DocPage = {
  href: string;
};

type DocsButtonProps = {
  doc: DocPage;
  variant: ButtonProps['variant'];
  label: string;
  queryParams?: Record<string, string>;
  scrollTo?: string;
};

export const DocsButton = ({
  doc,
  variant,
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
    <Button
      onClick={handleClick}
      className="w-auto px-4 py-2 flex flex-row items-center gap-x-2"
      variant={variant}
    >
      <BookOpenText className="size-4" />
      <span>{label}</span>
    </Button>
  );
};
