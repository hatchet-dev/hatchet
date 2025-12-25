import {
  DEFAULT_TENANT_COLOR,
  TenantColorDot,
} from '@/components/v1/molecules/nav-bar/tenant-color-dot';
import { Button } from '@/components/v1/ui/button';
import { cn } from '@/lib/utils';
import { CaretSortIcon } from '@radix-ui/react-icons';
import * as React from 'react';

type TenantSwitcherTriggerButtonProps = React.ComponentPropsWithoutRef<
  typeof Button
> & {
  fullWidth?: boolean;
  open: boolean;
  tenantName: string;
  tenantColor?: string;
  shouldGroupByOrganization: boolean;
  disabled: boolean;
  showSpinner: boolean;
};

export const TenantSwitcherTriggerButton = React.forwardRef<
  HTMLButtonElement,
  TenantSwitcherTriggerButtonProps
>(
  (
    {
      className,
      fullWidth,
      open,
      tenantName,
      tenantColor,
      shouldGroupByOrganization,
      disabled,
      showSpinner,
      ...props
    },
    ref,
  ) => {
    return (
      <Button
        ref={ref}
        variant="outline"
        size="sm"
        role="combobox"
        aria-expanded={open}
        aria-label="Select a tenant"
        className={cn(
          shouldGroupByOrganization
            ? 'w-full min-w-0 justify-between gap-2 bg-muted/20 shadow-none hover:bg-muted/30'
            : 'min-w-0 justify-between gap-2 bg-muted/20 shadow-none hover:bg-muted/30',
          fullWidth && 'w-full',
          open && 'bg-muted/30',
          className,
        )}
        style={{ borderColor: tenantColor || DEFAULT_TENANT_COLOR }}
        disabled={disabled || props.disabled}
        {...props}
      >
        <div className="flex min-w-0 flex-1 items-center gap-2 text-left">
          <TenantColorDot color={tenantColor} />
          <span className="min-w-0 flex-1 truncate">{tenantName}</span>
        </div>
        {showSpinner ? (
          <div className="h-4 w-4 animate-spin rounded-full border-2 border-muted-foreground/30 border-t-muted-foreground/70" />
        ) : (
          <CaretSortIcon className="size-4 shrink-0 opacity-50" />
        )}
      </Button>
    );
  },
);

TenantSwitcherTriggerButton.displayName = 'TenantSwitcherTriggerButton';
