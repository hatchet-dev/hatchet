import CopyToClipboard from './copy-to-clipboard';
import { useTheme } from '@/components/hooks/use-theme';
import { cn } from '@/lib/utils';
import { Light as SyntaxHighlighter } from 'react-syntax-highlighter';
import json from 'react-syntax-highlighter/dist/esm/languages/hljs/json';
import typescript from 'react-syntax-highlighter/dist/esm/languages/hljs/typescript';
import yaml from 'react-syntax-highlighter/dist/esm/languages/hljs/yaml';
import {
  anOldHope,
  atomOneLight,
} from 'react-syntax-highlighter/dist/esm/styles/hljs';

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
  minWidth,
  copy = true,
  wrapLines = true,
  onCopy,
}: {
  code: string;
  copyCode?: string;
  language: string;
  className?: string;
  maxHeight?: string;
  minHeight?: string;
  maxWidth?: string;
  minWidth?: string;
  copy?: boolean;
  wrapLines?: boolean;
  onCopy?: () => void;
}) {
  const { theme } = useTheme();

  return (
    <div className={cn('relative h-fit w-full rounded-lg bg-muted', className)}>
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
          maxHeight,
          minHeight,
          maxWidth,
          minWidth,
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
          className="absolute right-2 top-2"
          text={(copyCode || code).trim()}
          onCopy={onCopy}
        />
      )}
    </div>
  );
}
