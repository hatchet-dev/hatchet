import { useState } from 'react';
import { cn } from '@/next/lib/utils';
import { Button } from '@/next/components/ui/button';
import { CheckIcon, Copy } from 'lucide-react';
import CodeStyleRender from './code-render';

interface CodeBlockProps {
  title?: string;
  language: string;
  value: string;
  className?: string;
  noHeader?: boolean;
  showLineNumbers?: boolean;
}

export function CodeBlock({
  noHeader = false,
  title,
  language,
  value,
  showLineNumbers = true,
  className,
}: CodeBlockProps) {
  const [copied, setCopied] = useState(false);

  const copyToClipboard = () => {
    navigator.clipboard.writeText(value);
    setCopied(true);
    setTimeout(() => setCopied(false), 2000);
  };

  return (
    <div
      className={cn(
        'relative rounded-md overflow-hidden border border-muted',
        className,
      )}
    >
      {!noHeader && (
        <div className="flex items-center justify-between px-2 bg-muted/50 border-b rounded-t-md">
          <div className="text-xs text-muted-foreground font-mono">
            {title || language}
          </div>
          <Button
            variant="ghost"
            size="sm"
            onClick={copyToClipboard}
            className="h-8 px-2"
          >
            {copied ? (
              <CheckIcon className="h-4 w-4" />
            ) : (
              <Copy className="h-4 w-4" />
            )}
          </Button>
        </div>
      )}
      <pre
        className={cn(
          // 'p-4 overflow-x-auto text-sm font-mono bg-muted/30 rounded-b-md',
          showLineNumbers && 'pl-8 relative counter-reset-line overflow-hidden',
        )}
      >
        <CodeStyleRender parsed={value} language={language} />
      </pre>
    </div>
  );
}
