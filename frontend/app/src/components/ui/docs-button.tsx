import * as React from 'react';
import { Button, ButtonProps } from './button';
import { BookOpenIcon } from 'lucide-react';
import { DocRef, useDocs } from '@/hooks/use-docs-sheet';

interface DocsButtonProps
  extends React.ButtonHTMLAttributes<HTMLButtonElement> {
  prefix?: string;
  doc: DocRef;
  size?: ButtonProps['size'];
  method?: 'sheet' | 'link';
}

const baseDocsUrl = 'https://docs.hatchet.run';

export function DocsButton({
  doc,
  prefix = 'Learn more about ',
  size = 'sm',
  method = 'sheet',
  ...props
}: DocsButtonProps) {
  const { open } = useDocs();

  const handleClick = (e: React.MouseEvent) => {
    if (method === 'sheet') {
      e.preventDefault();
      open(doc);
    } else {
      window.open(`${baseDocsUrl}${doc.href}`, '_blank');
    }
  };

  return (
    <Button variant="outline" {...props} size={size} onClick={handleClick}>
      <BookOpenIcon className="w-4 h-4 mr-2" />
      {prefix} {doc.title}
    </Button>
  );
}
