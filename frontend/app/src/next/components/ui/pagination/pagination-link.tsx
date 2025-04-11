import * as React from 'react';
import { ChevronLeft, ChevronRight } from 'lucide-react';
import { cn } from '@/next/lib/utils';
import { ButtonProps, buttonVariants } from '@/next/components/ui/button';
import { cva } from 'class-variance-authority';

const paginationVariants = cva(
  'cursor-pointer flex items-center justify-center',
  {
    variants: {
      variant: {
        default: '',
        active: 'border border-input bg-background shadow-sm',
        previous: '',
        next: '',
        content: 'w-auto min-w-[2.5rem]',
      },
    },
    defaultVariants: {
      variant: 'default',
    },
  },
);

type PaginationLinkProps = {
  isActive?: boolean;
  disabled?: boolean;
  variant?: 'default' | 'active' | 'previous' | 'next' | 'content';
} & Pick<ButtonProps, 'size'> &
  React.ComponentProps<'a'>;

const PaginationLink = React.forwardRef<HTMLAnchorElement, PaginationLinkProps>(
  (
    {
      className,
      isActive,
      disabled,
      variant = 'default',
      size = 'icon',
      children,
      onClick,
      ...props
    },
    ref,
  ) => {
    const computedVariant = isActive ? 'active' : variant;
    const buttonVariant = isActive ? 'outline' : 'ghost';

    const handleClick = (e: React.MouseEvent<HTMLAnchorElement>) => {
      if (disabled) {
        e.preventDefault();
        return;
      }
      if (onClick) {
        e.preventDefault();
        onClick(e);
      }
    };

    return (
      <a
        ref={ref}
        aria-current={isActive ? 'page' : undefined}
        aria-disabled={disabled}
        className={cn(
          buttonVariants({ variant: buttonVariant, size }),
          paginationVariants({ variant: computedVariant }),
          disabled && 'pointer-events-none opacity-50',
          'select-none',
          className,
        )}
        onClick={handleClick}
        {...props}
      >
        {children}
      </a>
    );
  },
);
PaginationLink.displayName = 'PaginationLink';

// Previous component
const PaginationPrevious = React.forwardRef<
  HTMLAnchorElement,
  PaginationLinkProps
>((props, ref) => (
  <PaginationLink
    ref={ref}
    variant="previous"
    size="icon"
    aria-label="Go to previous page"
    {...props}
  >
    <ChevronLeft className="h-4 w-4" />
    <span className="sr-only">Previous</span>
  </PaginationLink>
));
PaginationPrevious.displayName = 'PaginationPrevious';

// Next component
const PaginationNext = React.forwardRef<HTMLAnchorElement, PaginationLinkProps>(
  (props, ref) => (
    <PaginationLink
      ref={ref}
      variant="next"
      size="icon"
      aria-label="Go to next page"
      {...props}
    >
      <span className="sr-only">Next</span>
      <ChevronRight className="h-4 w-4" />
    </PaginationLink>
  ),
);
PaginationNext.displayName = 'PaginationNext';

// Item component
const PaginationItem = React.forwardRef<
  HTMLLIElement,
  React.ComponentProps<'li'>
>(({ className, ...props }, ref) => (
  <li ref={ref} className={cn('', className)} {...props} />
));
PaginationItem.displayName = 'PaginationItem';

export {
  PaginationLink,
  PaginationPrevious,
  PaginationNext,
  PaginationItem,
  paginationVariants,
};
