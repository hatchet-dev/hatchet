import { SidebarButtonPrimaryAction } from '@/components/v1/nav/sidebar-buttons';
import { Button } from '@/components/v1/ui/button';
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from '@/components/v1/ui/dropdown-menu';
import { useTenantDetails } from '@/hooks/use-tenant';
import { cn } from '@/lib/utils';
import useApiMeta from '@/pages/auth/hooks/use-api-meta';
import { VersionInfo } from '@/pages/main/info/components/version-info';
import { appRoutes } from '@/router';
import { useNavigate } from '@tanstack/react-router';
import React, { useMemo, useState } from 'react';
import {
  BiBook,
  BiCalendar,
  BiChat,
  BiHelpCircle,
  BiLogoDiscordAlt,
  BiSolidGraduation,
} from 'react-icons/bi';

export function HelpDropdown({
  variant = 'default',
  triggerVariant = 'icon',
  align = 'end',
  side,
  className,
}: {
  variant?: 'default' | 'sidebar';
  triggerVariant?: 'icon' | 'button';
  align?: React.ComponentProps<typeof DropdownMenuContent>['align'];
  side?: React.ComponentProps<typeof DropdownMenuContent>['side'];
  className?: string;
}) {
  const { meta } = useApiMeta();
  const navigate = useNavigate();
  const { tenant } = useTenantDetails();
  const [open, setOpen] = useState(false);

  const hasPylon = useMemo(() => {
    return !!meta?.pylonAppId;
  }, [meta?.pylonAppId]);

  const trigger =
    triggerVariant === 'button' ? (
      <SidebarButtonPrimaryAction
        name="Help"
        icon={<BiHelpCircle className="size-4 mr-2" />}
        selected={open}
        className={cn(className)}
      />
    ) : (
      <Button
        variant="icon"
        size="icon"
        aria-label="Help Menu"
        hoverText="Help"
        hoverTextSide="right"
        className={className}
      >
        <BiHelpCircle className="h-6 w-6 cursor-pointer text-foreground" />
      </Button>
    );

  return (
    <DropdownMenu open={open} onOpenChange={setOpen}>
      <DropdownMenuTrigger asChild>{trigger}</DropdownMenuTrigger>
      <DropdownMenuContent
        className="w-56"
        variant={variant === 'sidebar' ? 'sidebar' : 'default'}
        align={align}
        side={side}
        forceMount
      >
        {hasPylon && (
          <DropdownMenuItem
            variant="interactive"
            onClick={() => (window as any).Pylon('show')}
          >
            <BiChat className="mr-2" />
            Chat with Support
          </DropdownMenuItem>
        )}
        <DropdownMenuItem
          variant="interactive"
          onClick={() => window.open('https://docs.hatchet.run', '_blank')}
        >
          <BiBook className="mr-2" />
          Documentation
        </DropdownMenuItem>
        <DropdownMenuItem
          variant="interactive"
          onClick={() =>
            window.open('https://discord.com/invite/ZMeUafwH89', '_blank')
          }
        >
          <BiLogoDiscordAlt className="mr-2" />
          Join Discord
        </DropdownMenuItem>
        <DropdownMenuItem
          variant="interactive"
          onClick={() =>
            window.open('https://hatchet.run/office-hours', '_blank')
          }
        >
          <BiCalendar className="mr-2" />
          Schedule Office Hours
        </DropdownMenuItem>
        <DropdownMenuItem
          variant="interactive"
          onClick={() => {
            if (!tenant) {
              return;
            }

            navigate({
              to: appRoutes.tenantOnboardingGetStartedRoute.to,
              params: { tenant: tenant.metadata.id },
            });
          }}
        >
          <BiSolidGraduation className="mr-2" />
          Restart Tutorial
        </DropdownMenuItem>
        <DropdownMenuSeparator />
        <DropdownMenuItem variant="static">
          <VersionInfo />
        </DropdownMenuItem>
      </DropdownMenuContent>
    </DropdownMenu>
  );
}
