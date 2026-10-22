import { Button } from '@/components/v1/ui/button';
import { type ReactNode } from 'react';

export type PaginationFooterProps = {
  summary: ReactNode;
  // zero-based
  page: number;
  numPages: number;
  onPageChange: (page: number) => void;
};

export function PaginationFooter({
  summary,
  page,
  numPages,
  onPageChange,
}: PaginationFooterProps) {
  return (
    <div className="flex items-center gap-2 border-t border-border/50 px-4 py-2 text-xs text-muted-foreground">
      <span className="font-mono">{summary}</span>
      <div className="ml-auto flex gap-1.5">
        <Button
          variant="outline"
          size="sm"
          className="h-7 text-xs"
          disabled={page === 0}
          onClick={() => onPageChange(page - 1)}
        >
          ← Previous
        </Button>
        <Button
          variant="outline"
          size="sm"
          className="h-7 text-xs"
          disabled={page + 1 >= numPages}
          onClick={() => onPageChange(page + 1)}
        >
          Next →
        </Button>
      </div>
    </div>
  );
}
