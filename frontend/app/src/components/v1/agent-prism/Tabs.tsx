import * as RadixTabs from '@radix-ui/react-tabs';
import cn from 'classnames';
import type { ComponentPropsWithRef } from 'react';
import * as React from 'react';

export interface TabItem<T extends string = string> {
  value: T;
  label: string;
  icon?: React.ReactNode;
  disabled?: boolean;
}

export type TabTheme = 'underline' | 'pill';

const BASE_TRIGGER =
  'text-sm font-medium transition-colors duration-200 disabled:opacity-50 disabled:cursor-not-allowed';

const THEMES = {
  underline: {
    list: 'h-9 flex border-b border-agentprism-border',
    trigger: `w-full justify-center px-3 ${BASE_TRIGGER}
      text-agentprism-secondary-foreground data-[state=active]:text-agentprism-foreground
      border-b-2 border-transparent data-[state=active]:border-agentprism-border-inverse
      -mb-[2px]
      data-[state=inactive]:[&:not(:disabled)]:hover:border-agentprism-border-inverse/20
      data-[state=inactive]:[&:not(:disabled)]:hover:text-agentprism-muted-foreground`,
  },
  pill: {
    list: 'h-9 inline-flex gap-1 p-1 bg-agentprism-secondary rounded-lg',
    trigger: `px-3 ${BASE_TRIGGER} rounded-md
      text-agentprism-muted-foreground data-[state=active]:text-agentprism-foreground
      data-[state=inactive]:[&:not(:disabled)]:hover:bg-agentprism-background/50 data-[state=active]:bg-agentprism-background data-[state=active]:shadow-sm
      dark:data-[state=active]:shadow-none`,
  },
} as const;

export type TabsProps<T extends string = string> = Omit<
  ComponentPropsWithRef<'div'>,
  'dir'
> & {
  /**
   * Array of tab items to display
   */
  items: TabItem<T>[];

  /**
   * The initially selected tab value (uncontrolled)
   */
  defaultValue?: T;

  /**
   * The currently selected tab value (controlled)
   */
  value?: T;

  /**
   * Callback fired when the selected tab changes
   */
  onValueChange?: (value: T) => void;

  /**
   * Visual theme variant for the tabs
   * @default "underline"
   */
  theme?: TabTheme;

  /**
   * Optional className for the root container
   */
  className?: string;

  /**
   * Optional className for the tabs list container
   */
  tabsListClassName?: string;

  /**
   * Optional className for individual tab triggers
   */
  triggerClassName?: string;

  /**
   * The direction of the content of the tabs
   */
  dir?: 'ltr' | 'rtl';
};

export const Tabs = <T extends string = string>({
  items,
  defaultValue,
  value,
  onValueChange,
  theme = 'underline',
  className = '',
  tabsListClassName = '',
  triggerClassName = '',
  dir,
  ...rest
}: TabsProps<T>) => {
  const defaultTab = defaultValue || items[0]?.value;

  const currentTheme = THEMES[theme];

  return (
    <RadixTabs.Root
      className={className}
      defaultValue={!value ? defaultTab : undefined}
      value={value}
      onValueChange={onValueChange as (value: string) => void}
      dir={dir}
      {...rest}
    >
      <RadixTabs.List
        className={cn(currentTheme.list, tabsListClassName)}
        aria-label="Navigation tabs"
      >
        {items.map((item: TabItem) => (
          <RadixTabs.Trigger
            key={item.value}
            value={item.value}
            disabled={item.disabled}
            className={cn(
              'group flex items-center overflow-hidden',
              currentTheme.trigger,
              triggerClassName,
            )}
          >
            {item.icon && (
              <span className="text-agentprism-secondary-foreground mr-2 group-data-[state=active]:text-current">
                {item.icon}
              </span>
            )}
            <span className="truncate text-sm font-medium">{item.label}</span>
          </RadixTabs.Trigger>
        ))}
      </RadixTabs.List>
    </RadixTabs.Root>
  );
};
