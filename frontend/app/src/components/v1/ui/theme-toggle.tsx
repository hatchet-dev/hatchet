import { useTheme } from '@/components/hooks/use-theme';
import { Button } from '@/components/v1/ui/button';
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from '@/components/v1/ui/dropdown-menu';
import { cn } from '@/lib/utils';
import { Check, Monitor, Moon, Sun } from 'lucide-react';
import * as React from 'react';

export const THEME_OPTIONS = [
  { value: 'light' as const, label: 'Light', icon: Sun },
  { value: 'dark' as const, label: 'Dark', icon: Moon },
  { value: 'system' as const, label: 'System', icon: Monitor },
];

type ThemeToggleProps = {
  className?: string;
  align?: 'start' | 'center' | 'end';
  /** Accessible label for the trigger button. */
  'aria-label'?: string;
};

/**
 * Compact theme switcher (Light / Dark / System).
 * Visible from auth pages and the main top nav so light mode is easy to find.
 */
export function ThemeToggle({
  className,
  align = 'end',
  'aria-label': ariaLabel = 'Change color theme',
}: ThemeToggleProps) {
  const [open, setOpen] = React.useState(false);
  const { theme, setTheme, currentlyVisibleTheme } = useTheme();

  // Prefer showing System icon when following OS; otherwise show resolved light/dark.
  const TriggerIcon =
    theme === 'system'
      ? Monitor
      : currentlyVisibleTheme === 'dark'
        ? Moon
        : Sun;

  return (
    <DropdownMenu open={open} onOpenChange={setOpen}>
      <DropdownMenuTrigger asChild>
        <Button
          type="button"
          variant="outline"
          size="icon"
          aria-label={ariaLabel}
          className={cn(
            'bg-muted/20 shadow-none hover:bg-muted/30',
            open && 'bg-muted/30',
            className,
          )}
        >
          <TriggerIcon className="size-4" />
        </Button>
      </DropdownMenuTrigger>
      <DropdownMenuContent align={align} className="w-40">
        {THEME_OPTIONS.map((option) => (
          <DropdownMenuItem
            key={option.value}
            variant="interactive"
            className="cursor-pointer"
            onClick={() => setTheme(option.value)}
          >
            <option.icon className="mr-2 size-4" />
            {option.label}
            {theme === option.value && <Check className="ml-auto size-4" />}
          </DropdownMenuItem>
        ))}
      </DropdownMenuContent>
    </DropdownMenu>
  );
}
