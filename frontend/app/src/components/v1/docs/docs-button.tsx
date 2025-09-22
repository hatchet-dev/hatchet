import { Button, ButtonProps } from '../ui/button';
import { useSidePanel } from '@/hooks/use-side-panel';
import { BookOpenText } from 'lucide-react';

export type DocPage = {
  title: string;
  href: string;
};

type DocsButtonProps = {
  doc: DocPage;
  size: 'mini' | 'full';
  variant: ButtonProps['variant'];
  label: string;
  queryParams?: Record<string, string>;
  scrollTo?: string;
};

export const DocsButton = ({
  doc,
  size,
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

  switch (size) {
    case 'full':
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
    case 'mini':
      return (
        <div className="flex flex-row items-center gap-x-4 w-full justify-center">
          <span className="text-mono font-semibold">{label}</span>
          <Button
            variant="ghost"
            size="icon"
            onClick={handleClick}
            className="border"
          >
            <BookOpenText className="size-4" />
          </Button>
        </div>
      );
  }
};
