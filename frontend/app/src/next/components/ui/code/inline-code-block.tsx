import React, { useState } from 'react';
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
  const [isHovered, setIsHovered] = useState(false);

  const copyToClipboard = () => {
    navigator.clipboard.writeText(value);
    setCopied(true);
    setTimeout(() => setCopied(false), 2000);
  };

  return (
    <TooltipProvider>
      <div
        className={cn(
          'relative inline-flex items-center rounded-md bg-muted/30 font-mono text-sm group',
          className,
        )}
        onMouseEnter={() => setIsHovered(true)}
        onMouseLeave={() => setIsHovered(false)}
      >
        <div className="px-2 py-1">
          <CodeStyleRender
            parsed={value}
            language={language}
            className="inline"
          />
        </div>
        {isHovered && (
          <Tooltip>
            <TooltipTrigger asChild>
              <Button
                variant="ghost"
                size="icon"
                onClick={copyToClipboard}
                className="h-6 w-6 p-0 absolute right-0 top-0 opacity-80 hover:opacity-100 hover:bg-background/50"
              >
                {copied ? (
                  <CheckIcon className="h-3 w-3" />
                ) : (
                  <Copy className="h-3 w-3" />
                )}
              </Button>
            </TooltipTrigger>
            <TooltipContent>
              <p>Copy code</p>
            </TooltipContent>
          </Tooltip>
        )}
      </div>
    </TooltipProvider>
  );
}
