import {
  Tooltip,
  TooltipContent,
  TooltipProvider,
  TooltipTrigger,
} from './tooltip';
import { cn } from '@/lib/utils';
import { Slot, Slottable } from '@radix-ui/react-slot';
import { cva, type VariantProps } from 'class-variance-authority';
import * as React from 'react';

const buttonVariants = cva(
  'inline-flex items-center justify-center whitespace-nowrap rounded-md text-sm font-medium transition-colors focus-visible:outline-none disabled:pointer-events-none disabled:opacity-50',
  {
    variants: {
      size: {
        default: 'h-9 px-4 py-2',
        sm: 'h-8 rounded-md px-3 text-xs',
        lg: 'h-10 rounded-md px-8',
        icon: 'h-9 w-9',
        xs: '',
      },
      variant: {
        default:
          'bg-primary text-primary-foreground shadow hover:bg-primary/90',
        destructive:
          'bg-destructive text-destructive-foreground shadow-sm hover:bg-destructive/90',
        outline:
          'border border-input bg-transparent shadow-sm hover:bg-accent hover:text-accent-foreground',
        secondary:
          'bg-secondary text-secondary-foreground shadow-sm hover:bg-secondary/80',
        ghost: 'hover:bg-accent hover:text-accent-foreground',
        link: 'text-primary underline-offset-4 hover:underline',
        icon: 'hover:bg-accent hover:text-accent-foreground p-1',
        cta: 'bg-primary text-primary-foreground shadow hover:bg-primary/90 h-8 border px-3',
      },
    },
    defaultVariants: {
      variant: 'default',
      size: 'default',
    },
  },
);

export interface ButtonProps
  extends React.ButtonHTMLAttributes<HTMLButtonElement>,
    VariantProps<typeof buttonVariants> {
  asChild?: boolean;
  hoverText?: string;
  hoverTextSide?: 'top' | 'right' | 'bottom' | 'left';
  leftIcon?: React.ReactNode;
  rightIcon?: React.ReactNode;
  fullWidth?: boolean;
}

const Button = React.forwardRef<HTMLButtonElement, ButtonProps>(
  (
    {
      className,
      variant,
      size,
      asChild = false,
      hoverText = undefined,
      hoverTextSide = 'top',
      leftIcon,
      rightIcon,
      fullWidth,
      children,
      ...props
    },
    ref,
  ) => {
    const Comp = asChild ? Slot : 'button';

    const iconPaddingClasses = cn(
      leftIcon && size !== 'icon' && 'pl-3',
      rightIcon && size !== 'icon' && 'pr-3',
    );

    const fullWidthClass = fullWidth ? 'w-full' : '';

    return (
      <TooltipProvider>
        <Tooltip>
          <TooltipTrigger asChild>
            <Comp
              className={cn(
                buttonVariants({ variant, size, className }),
                iconPaddingClasses,
                fullWidthClass,
              )}
              ref={ref}
              {...props}
            >
              {leftIcon && <span className="-ml-1 mr-2">{leftIcon}</span>}
              <Slottable>{children}</Slottable>
              {rightIcon && <span className="-mr-1 ml-2">{rightIcon}</span>}
            </Comp>
          </TooltipTrigger>
          {hoverText && (
            <TooltipContent side={hoverTextSide}>{hoverText}</TooltipContent>
          )}
        </Tooltip>
      </TooltipProvider>
    );
  },
);
Button.displayName = 'Button';

export { Button, buttonVariants };
