import { cn } from '@/lib/utils';
import * as TooltipPrimitive from '@radix-ui/react-tooltip';
import React from 'react';

const PortalTooltipProvider = TooltipPrimitive.Provider;

const PortalTooltip = TooltipPrimitive.Root;

const PortalTooltipTrigger = TooltipPrimitive.Trigger;

const PortalTooltipContent = React.forwardRef<
  React.ElementRef<typeof TooltipPrimitive.Content>,
  React.ComponentPropsWithoutRef<typeof TooltipPrimitive.Content>
>(({ className, sideOffset = 4, ...props }, ref) => (
  <TooltipPrimitive.Portal>
    <TooltipPrimitive.Content
      ref={ref}
      sideOffset={sideOffset}
      className={cn(
        'z-[300] overflow-hidden rounded-md bg-primary px-3 py-1.5 text-xs text-primary-foreground animate-in fade-in-0 zoom-in-95 data-[state=closed]:animate-out data-[state=closed]:fade-out-0 data-[state=closed]:zoom-out-95 data-[side=bottom]:slide-in-from-top-2 data-[side=left]:slide-in-from-right-2 data-[side=right]:slide-in-from-left-2 data-[side=top]:slide-in-from-bottom-2',
        className,
      )}
      avoidCollisions={true}
      collisionPadding={8}
      {...props}
    />
  </TooltipPrimitive.Portal>
));
PortalTooltipContent.displayName = TooltipPrimitive.Content.displayName;

export {
  PortalTooltip,
  PortalTooltipTrigger,
  PortalTooltipContent,
  PortalTooltipProvider,
};
