import * as React from 'react';
import { Button, ButtonProps } from './button';
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from './dropdown-menu';
import { cn } from '@/next/lib/utils';
import { ChevronDown } from 'lucide-react';

export interface SplitButtonProps extends Omit<ButtonProps, 'className'> {
  dropdownItems: {
    label: string;
    onClick: () => void;
    disabled?: boolean;
  }[];
  className?: string;
}

const SplitButton = React.forwardRef<HTMLButtonElement, SplitButtonProps>(
  ({ className, dropdownItems, ...props }, ref) => {
    return (
      <div className={cn('inline-flex', className)}>
        <Button {...props} ref={ref} className="rounded-r-none" />
        <DropdownMenu>
          <DropdownMenuTrigger asChild>
            <Button
              variant={props.variant}
              size={props.size}
              disabled={props.disabled}
              className="rounded-l-none border-l px-0"
            >
              <ChevronDown className="h-3 w-3" />
              <span className="sr-only">More options</span>
            </Button>
          </DropdownMenuTrigger>
          <DropdownMenuContent align="end">
            {dropdownItems.map((item, index) => (
              <DropdownMenuItem
                key={index}
                onClick={item.onClick}
                disabled={item.disabled}
              >
                {item.label}
              </DropdownMenuItem>
            ))}
          </DropdownMenuContent>
        </DropdownMenu>
      </div>
    );
  },
);
SplitButton.displayName = 'SplitButton';

export { SplitButton };
