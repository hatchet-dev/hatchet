import * as React from 'react';
import * as TabsPrimitive from '@radix-ui/react-tabs';
import { cva } from 'class-variance-authority';

import { cn } from '@/next/lib/utils';

interface TabsProps
  extends React.ComponentPropsWithoutRef<typeof TabsPrimitive.Root> {}

const tabsListVariants = cva('', {
  variants: {
    layout: {
      default:
        'inline-flex h-10 items-center justify-center rounded-md bg-muted p-1 text-muted-foreground',
      underlined:
        'w-full justify-start rounded-none border-b bg-transparent p-0',
    },
  },
  defaultVariants: {
    layout: 'default',
  },
});

const tabsTriggerVariants = cva('', {
  variants: {
    variant: {
      underlined:
        'relative rounded-none border-b-2 border-b-transparent bg-transparent px-3 py-1.5 text-sm font-medium text-muted-foreground shadow-none transition-none focus-visible:ring-0 data-[state=active]:border-b-primary data-[state=active]:text-foreground data-[state=active]:shadow-none',
      default:
        'inline-flex items-center justify-center whitespace-nowrap rounded-sm px-3 py-1.5 text-sm font-medium ring-offset-background transition-all focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2 disabled:pointer-events-none disabled:opacity-50 data-[state=active]:bg-background data-[state=active]:text-foreground data-[state=active]:shadow-sm',
    },
  },
  defaultVariants: {
    variant: 'default',
  },
});

const Tabs = React.forwardRef<
  React.ElementRef<typeof TabsPrimitive.Root>,
  TabsProps
>(({ defaultValue, value, onValueChange, ...props }, ref) => {
  const handleValueChange = React.useCallback(
    (newValue: string) => {
      onValueChange?.(newValue);
    },
    [onValueChange],
  );

  return (
    <TabsPrimitive.Root
      ref={ref}
      value={value}
      onValueChange={handleValueChange}
      defaultValue={defaultValue}
      {...props}
    />
  );
});
Tabs.displayName = TabsPrimitive.Root.displayName;

const TabsList = React.forwardRef<
  React.ElementRef<typeof TabsPrimitive.List>,
  React.ComponentPropsWithoutRef<typeof TabsPrimitive.List> & {
    layout?: 'default' | 'underlined';
  }
>(({ className, layout = 'default', ...props }, ref) => (
  <TabsPrimitive.List
    ref={ref}
    className={cn(tabsListVariants({ layout }), className)}
    {...props}
  />
));
TabsList.displayName = TabsPrimitive.List.displayName;

const TabsTrigger = React.forwardRef<
  React.ElementRef<typeof TabsPrimitive.Trigger>,
  React.ComponentPropsWithoutRef<typeof TabsPrimitive.Trigger> & {
    variant?: 'default' | 'underlined';
  }
>(({ className, variant = 'default', ...props }, ref) => (
  <TabsPrimitive.Trigger
    ref={ref}
    className={cn(tabsTriggerVariants({ variant }), className)}
    {...props}
  />
));
TabsTrigger.displayName = TabsPrimitive.Trigger.displayName;

const TabsContent = React.forwardRef<
  React.ElementRef<typeof TabsPrimitive.Content>,
  React.ComponentPropsWithoutRef<typeof TabsPrimitive.Content>
>(({ className, ...props }, ref) => (
  <TabsPrimitive.Content
    ref={ref}
    className={cn(
      'mt-2 ring-offset-background focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2',
      className,
    )}
    {...props}
  />
));
TabsContent.displayName = TabsPrimitive.Content.displayName;

export { Tabs, TabsList, TabsTrigger, TabsContent };
