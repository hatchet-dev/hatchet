import { Light as SyntaxHighlighter } from 'react-syntax-highlighter';

import typescript from 'react-syntax-highlighter/dist/esm/languages/hljs/typescript';
import yaml from 'react-syntax-highlighter/dist/esm/languages/hljs/yaml';
import json from 'react-syntax-highlighter/dist/esm/languages/hljs/json';

import {
  anOldHope,
  atomOneLight,
} from 'react-syntax-highlighter/dist/esm/styles/hljs';
import CopyToClipboard from './copy-to-clipboard';
import { cn } from '@/lib/utils';
import { useTheme } from '../theme-provider';

SyntaxHighlighter.registerLanguage('typescript', typescript);
SyntaxHighlighter.registerLanguage('yaml', yaml);
SyntaxHighlighter.registerLanguage('json', json);

export function CodeHighlighter({
  code,
  copyCode,
  language,
  className,
  maxHeight,
  minHeight,
  maxWidth,
  copy = true,
  wrapLines = true,
}: {
  code: string;
  copyCode?: string;
  language: string;
  className?: string;
  maxHeight?: string;
  minHeight?: string;
  maxWidth?: string;
  copy?: boolean;
  wrapLines?: boolean;
}) {
  const { theme } = useTheme();

  return (
    <div className={cn('w-full h-fit relative bg-muted rounded-lg', className)}>
      <SyntaxHighlighter
        language={language}
        style={theme == 'dark' ? anOldHope : atomOneLight}
        wrapLines={wrapLines}
        lineProps={{
          style: { wordBreak: 'break-all', whiteSpace: 'pre-wrap' },
        }}
        customStyle={{
          cursor: 'default',
          borderRadius: '0.5rem',
          maxHeight: maxHeight,
          minHeight: minHeight,
          maxWidth: maxWidth,
          fontFamily:
            "ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, 'Liberation Mono', 'Courier New', monospace",
          fontSize: '0.75rem',
          lineHeight: '1rem',
          padding: '0.5rem',
          paddingRight: '2rem',
          flex: '1',
          background: 'transparent',
        }}
      >
        {code.trim()}
      </SyntaxHighlighter>
      {copy && (
        <CopyToClipboard
          className="absolute top-2 right-2"
          text={(copyCode || code).trim()}
        />
      )}
    </div>
  );
}
