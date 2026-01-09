import { cn } from '@/lib/utils';
import * as SeparatorPrimitive from '@radix-ui/react-separator';
import * as React from 'react';

const Separator = React.forwardRef<
  React.ElementRef<typeof SeparatorPrimitive.Root>,
  React.ComponentPropsWithoutRef<typeof SeparatorPrimitive.Root> & {
    /**
     * When `true`, the separator will extend to the full size of its container,
     * breaking out of any parent padding or margins. It uses container query units
     * (cqw) so must be used within an inline-size container.
     *
     * @default false
     * @example
     * // Use flush when the separator should span the full container size
     * <div className="px-4">
     *   <Separator flush />
     * </div>
     */
    flush?: boolean;
  }
>(
  (
    {
      className,
      orientation = 'horizontal',
      decorative = true,
      flush = false,
      ...props
    },
    ref,
  ) => (
    <SeparatorPrimitive.Root
      ref={ref}
      decorative={decorative}
      orientation={orientation}
      className={cn(
        'shrink-0 bg-border',
        orientation === 'horizontal' ? 'h-[1px] w-full' : 'h-full w-[1px]',
        flush &&
          orientation === 'horizontal' &&
          'relative left-1/2 right-1/2 -ml-[50cqw] -mr-[50cqw] w-[100cqw] max-w-[100cqw]',
        flush &&
          orientation === 'vertical' &&
          'relative top-1/2 bottom-1/2 -mt-[50cqh] -mb-[50cqh] h-[100cqh] max-h-[100cqh]',
        className,
      )}
      {...props}
    />
  ),
);
Separator.displayName = SeparatorPrimitive.Root.displayName;

export { Separator };
