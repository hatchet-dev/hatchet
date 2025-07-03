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
  highlightLines?: number[];
  highlightStrings?: string[];
  link?: string;
}

const defaultHighlightLines: number[] = [];
const defaultHighlightStrings: string[] = [];

export function Code({
  variant = 'block',
  language,
  value,
  title,
  noHeader,
  showLineNumbers = false,
  highlightLines = defaultHighlightLines,
  highlightStrings = defaultHighlightStrings,
  className,
  link,
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
          highlightLines={highlightLines}
          highlightStrings={highlightStrings}
          className={cn(className)}
          link={link}
          {...props}
        />
      );
  }
}
