import { Button, type ButtonProps } from '@/components/v1/ui/button';
import {
  Collapsible,
  CollapsibleContent,
} from '@/components/v1/ui/collapsible';
import { cn } from '@/lib/utils';
import { Link, useMatchRoute } from '@tanstack/react-router';
import React from 'react';

type SidebarButtonPrimaryLinkProps = {
  onNavLinkClick: () => void;
  to: string;
  params?: Record<string, string>;
  name: string;
  icon: React.ReactNode;
  prefix?: string;
  collapsibleChildren?: React.ReactNode[];
};

export type SidebarButtonPrimaryActionProps = Omit<ButtonProps, 'children'> & {
  name: string;
  icon: React.ReactNode;
  selected?: boolean;
};

// Use this when a primary sidebar button needs to behave like a true `<button>`
// trigger (e.g. Radix `DropdownMenuTrigger asChild`), so event handlers + refs
// passed by the parent flow through to the underlying DOM element.
export const SidebarButtonPrimaryAction = React.forwardRef<
  HTMLButtonElement,
  SidebarButtonPrimaryActionProps
>(({ name, icon, selected, className, ...props }, ref) => {
  return (
    <Button
      {...props}
      ref={ref}
      type="button"
      variant="ghost"
      className={cn(
        'w-full justify-start pl-2 min-w-0 overflow-hidden',
        selected && 'bg-slate-200 dark:bg-slate-800',
        className,
      )}
    >
      {icon}
      <span className="truncate">{name}</span>
    </Button>
  );
});
SidebarButtonPrimaryAction.displayName = 'SidebarButtonPrimaryAction';

export function SidebarButtonPrimary(
  props: SidebarButtonPrimaryLinkProps | SidebarButtonPrimaryActionProps,
) {
  const matchRoute = useMatchRoute();

  // Action-style (no routing) button, used for things like dropdown triggers.
  if (!('to' in props)) {
    // Keep backwards compatibility: this component is still used in a few places.
    // Prefer `SidebarButtonPrimaryAction` when the caller needs `asChild` behavior.
    const { name, icon, selected, ...rest } = props;
    return (
      <SidebarButtonPrimaryAction
        {...rest}
        name={name}
        icon={icon}
        selected={selected}
      />
    );
  }

  const {
    onNavLinkClick,
    to,
    params,
    name,
    icon,
    prefix,
    collapsibleChildren = [],
  } = props;

  // `to` (and `prefix`) are TanStack route templates (e.g. `/tenants/$tenant/...`).
  // Use the router matcher instead of raw string comparisons against `location.pathname`.
  const open =
    collapsibleChildren.length > 0
      ? prefix
        ? Boolean(matchRoute({ to: prefix, params, fuzzy: true }))
        : Boolean(matchRoute({ to, params, fuzzy: true }))
      : false;

  const selected =
    collapsibleChildren.length > 0 ? open : Boolean(matchRoute({ to, params }));

  const primaryLink = (
    <Link to={to} params={params} onClick={onNavLinkClick}>
      <Button
        variant="ghost"
        className={cn(
          'w-full justify-start pl-2 min-w-0 overflow-hidden [&_svg]:text-muted-foreground/50',
          selected && 'bg-slate-200 dark:bg-slate-800 [&_svg]:text-primary',
        )}
      >
        {icon}
        <span className="truncate">{name}</span>
      </Button>
    </Link>
  );

  return collapsibleChildren.length === 0 ? (
    primaryLink
  ) : (
    <Collapsible open={open} className="w-full">
      {primaryLink}
      <CollapsibleContent className={'ml-4 space-y-2 border-l border-muted'}>
        {collapsibleChildren}
      </CollapsibleContent>
    </Collapsible>
  );
}

export function SidebarButtonSecondary({
  onNavLinkClick,
  to,
  params,
  name,
  prefix,
}: {
  onNavLinkClick: () => void;
  to: string;
  params?: Record<string, string>;
  name: string;
  prefix?: string;
}) {
  const matchRoute = useMatchRoute();
  const hasPrefix = prefix
    ? Boolean(matchRoute({ to: prefix, params, fuzzy: true }))
    : false;
  const selected = Boolean(matchRoute({ to, params })) || hasPrefix;

  return (
    <Link to={to} params={params} onClick={onNavLinkClick}>
      <Button
        variant="ghost"
        size="sm"
        className={cn(
          'my-[1px] ml-1 mr-3 w-[calc(100%-3px)] justify-start pl-3 pr-0 min-w-0 overflow-hidden',
          selected && 'bg-slate-200 dark:bg-slate-800',
        )}
      >
        <span className="truncate">{name}</span>
      </Button>
    </Link>
  );
}
