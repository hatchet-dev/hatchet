import * as React from 'react';
import { cn } from '@/next/lib/utils';

interface HeadlineProps extends React.HTMLAttributes<HTMLDivElement> {
  children: React.ReactNode;
}

const Headline = React.forwardRef<HTMLDivElement, HeadlineProps>(
  ({ className, children, ...props }, ref) => {
    return (
      <div className={cn('mb-6', className)} ref={ref} {...props}>
        <div className="flex flex-wrap items-center justify-between gap-4 mb-4">
          {children}
        </div>
      </div>
    );
  },
);
Headline.displayName = 'Headline';

interface PageTitleProps extends React.HTMLAttributes<HTMLDivElement> {
  children: React.ReactNode;
  description?: React.ReactNode;
}

const PageTitle = React.forwardRef<HTMLDivElement, PageTitleProps>(
  ({ className, children, description, ...props }, ref) => {
    return (
      <div ref={ref} {...props}>
        <h1 className="text-2xl font-bold">{children}</h1>
        {description && (
          <p className="text-muted-foreground mt-2">{description}</p>
        )}
      </div>
    );
  },
);
PageTitle.displayName = 'PageTitle';

interface PageDescriptionProps
  extends React.HTMLAttributes<HTMLParagraphElement> {
  children: React.ReactNode;
}

const PageDescription = React.forwardRef<
  HTMLParagraphElement,
  PageDescriptionProps
>(({ className, children, ...props }, ref) => {
  return (
    <p className={cn('text-muted-foreground', className)} ref={ref} {...props}>
      {children}
    </p>
  );
});
PageDescription.displayName = 'PageDescription';

interface HeadlineActionsProps extends React.HTMLAttributes<HTMLDivElement> {
  children: React.ReactNode;
}

const HeadlineActions = React.forwardRef<HTMLDivElement, HeadlineActionsProps>(
  ({ className, children, ...props }, ref) => {
    return (
      <div
        className={cn('flex flex-row items-center gap-2', className)}
        ref={ref}
        {...props}
      >
        {children}
      </div>
    );
  },
);
HeadlineActions.displayName = 'HeadlineActions';

interface HeadlineActionItemProps extends React.HTMLAttributes<HTMLDivElement> {
  children: React.ReactNode;
}

const HeadlineActionItem = React.forwardRef<
  HTMLDivElement,
  HeadlineActionItemProps
>(({ className, children, ...props }, ref) => {
  return (
    <div className={cn('flex items-center', className)} ref={ref} {...props}>
      {children}
    </div>
  );
});
HeadlineActionItem.displayName = 'HeadlineActionItem';

export {
  Headline,
  PageTitle,
  PageDescription,
  HeadlineActions,
  HeadlineActionItem,
};
