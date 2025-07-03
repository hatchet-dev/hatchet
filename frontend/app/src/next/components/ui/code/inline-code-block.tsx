import { useState } from 'react';
import { cn } from '@/next/lib/utils';
import { Button } from '@/next/components/ui/button';
import { CheckIcon, Copy } from 'lucide-react';
import CodeStyleRender from './code-render';
import {
  Tooltip,
  TooltipContent,
  TooltipProvider,
  TooltipTrigger,
} from '@/next/components/ui/tooltip';

interface InlineCodeBlockProps {
  language: string;
  value: string;
  className?: string;
}

export function InlineCodeBlock({
  language,
  value,
  className,
}: InlineCodeBlockProps) {
  const [copied, setCopied] = useState(false);

  const copyToClipboard = async () => {
    await navigator.clipboard.writeText(value);
    setCopied(true);
    setTimeout(() => setCopied(false), 2000);
  };

  return (
    <TooltipProvider>
      <div
        className={cn(
          'relative inline-flex items-center rounded-md bg-muted/30 font-mono text-sm group mx-1',
          className,
        )}
      >
        <div className="px-0.5 py-0.5">
          <CodeStyleRender
            showLineNumbers={false}
            parsed={value}
            language={language}
            className="inline"
          />
        </div>
        <Tooltip>
          <TooltipTrigger asChild>
            <Button
              variant="ghost"
              size="icon"
              onClick={copyToClipboard}
              className="h-6 w-6 p-0 absolute right-0 top-1/2 -translate-y-1/2 opacity-0 group-hover:opacity-80 hover:opacity-100 hover:bg-background/50"
            >
              {copied ? (
                <CheckIcon className="h-3 w-3" />
              ) : (
                <Copy className="h-3 w-3" />
              )}
            </Button>
          </TooltipTrigger>
          <TooltipContent>
            <p>Copy</p>
          </TooltipContent>
        </Tooltip>
      </div>
    </TooltipProvider>
  );
}
