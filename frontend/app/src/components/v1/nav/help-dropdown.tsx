import { usePylon } from '@/components/support-chat';
import { SidebarButtonPrimaryAction } from '@/components/v1/nav/sidebar-buttons';
import { Button } from '@/components/v1/ui/button';
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from '@/components/v1/ui/dropdown-menu';
import { cn } from '@/lib/utils';
import { VersionInfo } from '@/pages/main/info/components/version-info';
import { ChevronDownIcon } from '@radix-ui/react-icons';
import React, { useState } from 'react';
import {
  BiHelpCircle,
  BiBook,
  BiCalendar,
  BiChat,
  BiLogoDiscordAlt,
} from 'react-icons/bi';

function HelpDropdownItems({ includeChat = true }: { includeChat?: boolean }) {
  const pylon = usePylon();

  return (
    <>
      {includeChat && pylon.enabled && (
        <DropdownMenuItem variant="interactive" onClick={pylon.show}>
          <BiChat className="mr-2" />
          Chat with Support
        </DropdownMenuItem>
      )}
      <DropdownMenuItem variant="interactive" asChild>
        <a
          href="https://docs.hatchet.run"
          target="_blank"
          rel="noopener noreferrer"
        >
          <BiBook className="mr-2" />
          Documentation
        </a>
      </DropdownMenuItem>
      <DropdownMenuItem variant="interactive" asChild>
        <a
          href="https://discord.com/invite/ZMeUafwH89"
          target="_blank"
          rel="noopener noreferrer"
        >
          <BiLogoDiscordAlt className="mr-2" />
          Join Discord
        </a>
      </DropdownMenuItem>
      <DropdownMenuItem variant="interactive" asChild>
        <a
          href="https://hatchet.run/office-hours"
          target="_blank"
          rel="noopener noreferrer"
        >
          <BiCalendar className="mr-2" />
          Schedule Office Hours
        </a>
      </DropdownMenuItem>
      <DropdownMenuSeparator />
      <DropdownMenuItem variant="static">
        <VersionInfo />
      </DropdownMenuItem>
    </>
  );
}

export function HelpDropdown({
  variant = 'default',
  triggerVariant = 'icon',
  align = 'end',
  side,
  className,
}: {
  variant?: 'default' | 'sidebar';
  triggerVariant?: 'icon' | 'button' | 'split';
  align?: React.ComponentProps<typeof DropdownMenuContent>['align'];
  side?: React.ComponentProps<typeof DropdownMenuContent>['side'];
  className?: string;
}) {
  const pylon = usePylon();
  const [open, setOpen] = useState(false);

  const isSplit = triggerVariant === 'split' && pylon.enabled;
  const includeChat = !isSplit;
  const title = pylon.enabled ? 'Support Chat' : 'Help';

  const trigger = (() => {
    // Only show the split button when Pylon is enabled. Otherwise, fall back to a simple Help menu trigger.
    if (triggerVariant === 'split' && !pylon.enabled) {
      return (
        <SidebarButtonPrimaryAction
          name="Help"
          icon={<BiHelpCircle className="size-4 mr-2" />}
          selected={open}
          className={cn(className)}
        />
      );
    }

    if (isSplit) {
      return (
        <div className={cn('flex w-full', className)}>
          <SidebarButtonPrimaryAction
            name={title}
            icon={<BiChat className="size-4 mr-2" />}
            className="w-auto flex-1 rounded-r-none"
            onClick={() => {
              if (pylon.enabled) {
                pylon.show();
                return;
              }

              setOpen(true);
            }}
            aria-label="Open Support Chat"
            selected={open}
          />

          <DropdownMenuTrigger asChild>
            <Button
              type="button"
              variant="ghost"
              size="icon"
              aria-label="Open Support Menu"
              className="rounded-l-none border-l border-slate-200 dark:border-slate-800"
            >
              <ChevronDownIcon className="size-4 opacity-70" />
            </Button>
          </DropdownMenuTrigger>
        </div>
      );
    }

    if (triggerVariant === 'button') {
      return (
        <SidebarButtonPrimaryAction
          name="Help & Support"
          icon={<BiHelpCircle className="size-4 mr-2" />}
          selected={open}
          className={cn(className)}
        />
      );
    }

    return (
      <Button
        variant="icon"
        size="icon"
        aria-label="Help Menu"
        hoverText="Help & Support"
        hoverTextSide="right"
        className={className}
      >
        <BiHelpCircle className="h-6 w-6 cursor-pointer text-foreground" />
      </Button>
    );
  })();

  return (
    <DropdownMenu open={open} onOpenChange={setOpen}>
      {isSplit ? (
        trigger
      ) : (
        <DropdownMenuTrigger asChild>{trigger}</DropdownMenuTrigger>
      )}
      <DropdownMenuContent
        className="w-56"
        variant={variant === 'sidebar' ? 'sidebar' : 'default'}
        align={align}
        side={side}
        forceMount
      >
        <HelpDropdownItems includeChat={includeChat} />
      </DropdownMenuContent>
    </DropdownMenu>
  );
}
