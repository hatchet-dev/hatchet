import { cn } from '@/lib/utils';
import { Light as SyntaxHighlighter } from 'react-syntax-highlighter';

import typescript from 'react-syntax-highlighter/dist/esm/languages/hljs/typescript';
import yaml from 'react-syntax-highlighter/dist/esm/languages/hljs/yaml';
import json from 'react-syntax-highlighter/dist/esm/languages/hljs/json';

import { anOldHope } from 'react-syntax-highlighter/dist/esm/styles/hljs';
import CopyToClipboard from './copy-to-clipboard';

SyntaxHighlighter.registerLanguage('typescript', typescript);
SyntaxHighlighter.registerLanguage('yaml', yaml);
SyntaxHighlighter.registerLanguage('json', json);

export function Code({
  children,
  language,
  className,
  maxHeight,
  maxWidth,
  copy,
  wrapLines = true,
}: {
  children: string;
  language: string;
  className?: string;
  maxHeight?: string;
  maxWidth?: string;
  copy?: boolean;
  wrapLines?: boolean;
}) {
  return (
    <div className={cn('text-xs flex flex-col gap-4 justify-end', className)}>
      <SyntaxHighlighter
        language={language}
        style={anOldHope}
        wrapLines={wrapLines}
        lineProps={{
          style: { wordBreak: 'break-all', whiteSpace: 'pre-wrap' },
        }}
        customStyle={{
          background: 'hsl(var(--muted) / 0.5)',
          borderRadius: '0.5rem',
          maxHeight: maxHeight,
          maxWidth: maxWidth,
          fontFamily:
            "ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, 'Liberation Mono', 'Courier New', monospace",
        }}
      >
        {children.trim()}
      </SyntaxHighlighter>
      {copy && <CopyToClipboard text={children.trim()} />}
    </div>
  );
}
