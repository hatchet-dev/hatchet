import React from 'react';
import { CodeBlock } from './code-block';
import { InlineCodeBlock } from './inline-code-block';
import { cn } from '@/next/lib/utils';

type CodeVariant = 'block' | 'inline';

export interface CodeProps extends React.HTMLAttributes<HTMLDivElement> {
  variant?: CodeVariant;
  language: string;
  value: string;
  title?: string;
  noHeader?: boolean;
  showLineNumbers?: boolean;
}

export function Code({
  variant = 'block',
  language,
  value,
  title,
  noHeader,
  showLineNumbers = false,
  className,
  ...props
}: CodeProps) {
  switch (variant) {
    case 'inline':
      return (
        <InlineCodeBlock
          language={language}
          value={value}
          className={className}
          {...props}
        />
      );
    case 'block':
    default:
      return (
        <CodeBlock
          language={language}
          value={value}
          title={title}
          noHeader={noHeader}
          showLineNumbers={showLineNumbers}
          className={cn(className)}
          {...props}
        />
      );
  }
}
