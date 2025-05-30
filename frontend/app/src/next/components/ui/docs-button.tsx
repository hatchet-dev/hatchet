import * as React from 'react';
import { Button, ButtonProps } from './button';
import { BookOpenIcon } from 'lucide-react';
import { DocRef } from '@/next/hooks/use-docs-sheet';
import { cn } from '@/next/lib/utils';
import {
  Tooltip,
  TooltipContent,
  TooltipProvider,
  TooltipTrigger,
} from './tooltip';
import { useSidePanel } from '@/next/hooks/use-side-panel';

interface DocsButtonProps
  extends React.ButtonHTMLAttributes<HTMLButtonElement> {
  prefix?: string;
  doc: DocRef;
  size?: ButtonProps['size'];
  method?: 'sheet' | 'link';
  variant?: ButtonProps['variant'];
  titleOverride?: string;
}

const baseDocsUrl = 'https://docs.hatchet.run';

export function DocsButton({
  doc,
  prefix = 'Learn more about ',
  size = 'sm',
  method = 'sheet',
  variant = 'outline',
  titleOverride,
  ...props
}: DocsButtonProps) {
  const { close: closeSideSheet, open } = useSidePanel();

  const handleClick = (e: React.MouseEvent) => {
    if (method === 'sheet') {
      e.preventDefault();
      closeSideSheet();
      open({
        type: 'docs',
        content: {
          href: `${baseDocsUrl}${doc.href}`,
          title: doc.title,
        },
      });
    } else {
      window.open(`${baseDocsUrl}${doc.href}`, '_blank');
    }
  };

  const buttonContent = (
    <Button variant={variant} {...props} size={size} onClick={handleClick}>
      <BookOpenIcon className={cn('w-4 h-4', size === 'icon' && 'w-6 h-6')} />
      {size !== 'icon' && (
        <span>
          {prefix} {titleOverride || doc.title}
        </span>
      )}
    </Button>
  );

  if (size === 'icon') {
    return (
      <TooltipProvider>
        <Tooltip>
          <TooltipTrigger asChild>{buttonContent}</TooltipTrigger>
          <TooltipContent>
            <p>
              {prefix} {doc.title}
            </p>
          </TooltipContent>
        </Tooltip>
      </TooltipProvider>
    );
  }

  return buttonContent;
}
