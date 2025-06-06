import * as React from 'react';
import { Button, ButtonProps } from './button';
import { BookOpenIcon } from 'lucide-react';
import { cn } from '@/next/lib/utils';
import {
  Tooltip,
  TooltipContent,
  TooltipProvider,
  TooltipTrigger,
} from './tooltip';
import { useSidePanel } from '@/next/hooks/use-side-panel';
import useApiMeta from '@/next/hooks/use-api-meta';

export type DocRef = {
  title: string;
  href: string;
};

interface DocsButtonProps
  extends React.ButtonHTMLAttributes<HTMLButtonElement> {
  prefix?: string;
  doc: DocRef;
  size?: ButtonProps['size'];
  method?: 'sheet' | 'link';
  variant?: ButtonProps['variant'];
  titleOverride?: string;
}

// FIXME: this will need to be dynamic for OSS
export const cloudDocsUrl = 'https://docs.onhatchet.run';
export const baseDocsUrl = 'https://docs.hatchet.run';

export function DocsButton({
  doc,
  prefix = 'Learn more about ',
  size = 'sm',
  variant = 'outline',
  titleOverride,
  ...props
}: DocsButtonProps) {
  const { open } = useSidePanel();
  const { isCloud } = useApiMeta();

  const handleClick = (e: React.MouseEvent) => {
    e.preventDefault();
    open({
      type: 'docs',
      content: {
        href: `${isCloud ? cloudDocsUrl : baseDocsUrl}${doc.href}`,
        title: doc.title,
      },
    });
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
