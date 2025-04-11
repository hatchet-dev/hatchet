import {
  Card,
  CardContent,
  CardHeader,
  CardTitle,
  CardDescription,
} from '@/next/components/ui/card';
import { Code } from '@/next/components/ui/code';
import { V1TaskStatus } from '@/next/lib/api';
import { RunsBadge } from './runs-badge';
import { cn } from '@/next/lib/utils';
import {
  Collapsible,
  CollapsibleContent,
  CollapsibleTrigger,
} from '@/next/components/ui/collapsible';
import { ChevronDownIcon } from '@radix-ui/react-icons';
import { useState } from 'react';

export type RunOutputCardVariant = 'input' | 'output' | 'metadata';

export interface RunOutputCardProps {
  title: string;
  description?: string;
  output?: any;
  status: V1TaskStatus;
  variant: RunOutputCardVariant;
  error?: string;
  collapsed?: boolean;
  actions?: React.ReactNode;
}

export function RunDataCard({
  title,
  description,
  output,
  status,
  variant,
  error,
  collapsed = false,
  actions,
}: RunOutputCardProps) {
  const [isOpen, setIsOpen] = useState(!collapsed);

  return (
    <Collapsible open={isOpen} onOpenChange={setIsOpen}>
      <Card>
        <CollapsibleTrigger asChild>
          <CardHeader className="py-3 px-4 cursor-pointer hover:bg-muted/50">
            <div className="flex items-center justify-between">
              <div className="flex items-center gap-2">
                <ChevronDownIcon
                  className={cn(
                    'h-4 w-4 transition-transform duration-200',
                    isOpen ? 'rotate-0' : '-rotate-90',
                  )}
                />
                <CardTitle className="text-sm font-medium">{title}</CardTitle>
                {variant === 'output' && (
                  <RunsBadge status={status} variant="xs" />
                )}
              </div>
              {actions}
            </div>
            {description && (
              <CardDescription className="text-xs">
                {description}
              </CardDescription>
            )}
          </CardHeader>
        </CollapsibleTrigger>
        <CollapsibleContent>
          <CardContent className={cn(error && 'border-destructive')}>
            <Code
              language="json"
              value={
                error
                  ? JSON.stringify(JSON.parse(error), null, 2)
                  : JSON.stringify(output, null, 2)
              }
            />
          </CardContent>
        </CollapsibleContent>
      </Card>
    </Collapsible>
  );
}
