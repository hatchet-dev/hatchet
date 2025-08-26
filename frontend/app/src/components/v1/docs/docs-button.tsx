import { Button } from '../ui/button';
import { useSidePanel } from '@/hooks/use-side-panel';
import { BookOpen } from 'lucide-react';

export type DocPage = {
  title: string;
  href: string;
};

type DocsButtonProps = {
  doc: DocPage;
  variant: 'mini' | 'full';
  label: string;
};

export const DocsButton = ({ doc, variant, label }: DocsButtonProps) => {
  const { open } = useSidePanel();

  const handleClick = () => {
    open({
      type: 'docs',
      content: doc,
    });
  };

  switch (variant) {
    case 'full':
      return (
        <Button onClick={handleClick} className="w-auto px-4 py-2">
          {label}
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
            <BookOpen className="size-4" />
          </Button>
        </div>
      );
  }
};
