import * as React from 'react';
import { PaginationContent } from './pagination-content';
import { PageSizeSelector } from './page-size-selector';
import { PageSelector } from './page-selector';
import { cn } from '@/next/lib/utils';

interface PaginationProps extends React.ComponentProps<'nav'> {
  children?: React.ReactNode;
}

const Pagination = ({ className, children, ...props }: PaginationProps) => {
  return (
    <nav
      role="navigation"
      aria-label="pagination"
      className={'mx-auto flex w-full justify-center'}
      {...props}
    >
      <PaginationContent className={cn('w-full justify-between', className)}>
        {children || (
          <>
            <PageSizeSelector />
            <PageSelector />
          </>
        )}
      </PaginationContent>
    </nav>
  );
};
Pagination.displayName = 'Pagination';

export { Pagination };
