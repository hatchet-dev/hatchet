import {
  DropdownMenu,
  DropdownMenuTrigger,
  DropdownMenuContent,
  DropdownMenuItem,
} from '@/next/components/ui/dropdown-menu';
import { Button } from '@/next/components/ui/button';
import { Avatar, AvatarFallback } from '@/next/components/ui/avatar';
import { Bell, Loader2 } from 'lucide-react';
import useNotifications from '@/next/hooks/use-notifications';
import { useEffect, useState } from 'react';
import {
  Tooltip,
  TooltipContent,
  TooltipProvider,
  TooltipTrigger,
} from '@/next/components/ui/tooltip';
import { cn } from '@/next/lib/utils';
import { TenantInvite } from '@/lib/api';
import useUser from '@/next/hooks/use-user';
import { useTenantDetails } from '@/next/hooks/use-tenant';
import { Card, CardHeader, CardTitle } from '@/next/components/ui/card';
import { TenantBlock } from './user-dropdown';
import { formatDistance } from 'date-fns';

export function InviteCard({ invite }: { invite: TenantInvite }) {
  const { invites } = useUser();
  const { setTenant } = useTenantDetails();

  return (
    <Card>
      <CardHeader>
        <CardTitle className="flex flex-row gap-2 items-center justify-between">
          <div className="flex flex-row gap-4 items-center">
            <TenantBlock
              variant="default"
              tenant={{
                name: invite.tenantName || 'Team',
              }}
            />
          </div>
          <div className="flex flex-row gap-2 items-center">
            <span className="text-sm text-muted-foreground">
              Expires{' '}
              {formatDistance(new Date(invite.expires), new Date(), {
                addSuffix: true,
              })}
            </span>
            <Button
              variant="outline"
              onClick={() => invites.reject.mutate(invite.metadata.id)}
              disabled={invites.reject.isPending}
            >
              {invites.reject.isPending ? (
                <Loader2 className="h-4 w-4 mr-2 animate-spin" />
              ) : null}
              Decline
            </Button>
            <Button
              onClick={async () => {
                await invites.accept.mutateAsync(invite.metadata.id);
                setTenant(invite.tenantId);
              }}
              disabled={invites.accept.isPending}
            >
              {invites.accept.isPending ? (
                <Loader2 className="h-4 w-4 mr-2 animate-spin" />
              ) : null}
              Accept
            </Button>
          </div>
        </CardTitle>
      </CardHeader>
    </Card>
  );
}

export function Alerter() {
  const { notifications: alerts } = useNotifications();

  const [open, setOpen] = useState(false);

  useEffect(() => {
    if (alerts.length === 0) {
      setOpen(false);
    }
    if (alerts.length > 0) {
      setOpen(true);
    }
  }, [alerts.length]);

  // Empty state with tooltip
  if (alerts.length === 0) {
    return (
      <TooltipProvider>
        <Tooltip>
          <TooltipTrigger asChild>
            <Button
              variant="ghost"
              className="flex items-center gap-2 p-0 opacity-30"
            >
              <Avatar className="h-8 w-8 rounded-lg">
                <AvatarFallback className="rounded-lg bg-transparent hover:bg-transparent">
                  <Bell className="h-4 w-4 text-muted-foreground" />
                </AvatarFallback>
              </Avatar>
            </Button>
          </TooltipTrigger>
          <TooltipContent>
            <p>No notifications</p>
          </TooltipContent>
        </Tooltip>
      </TooltipProvider>
    );
  }

  // Notification state with dropdown
  return (
    <DropdownMenu open={open} onOpenChange={setOpen}>
      <DropdownMenuTrigger asChild>
        <Button variant="ghost" className="flex items-center gap-2 p-0">
          <Avatar className="h-8 w-8 rounded-lg">
            <AvatarFallback
              className={cn(
                'rounded-lg',
                'bg-primary text-primary-foreground animate-pulse',
              )}
            >
              {alerts.length}
            </AvatarFallback>
          </Avatar>
        </Button>
      </DropdownMenuTrigger>
      <DropdownMenuContent
        className="min-w-56 rounded-lg"
        align="end"
        sideOffset={4}
      >
        {alerts.map((alert) => (
          <DropdownMenuItem key={alert.id}>
            <div className="flex items-center gap-2 px-1 py-1.5 text-left text-sm">
              {alert.invite ? <InviteCard invite={alert.invite} /> : null}
            </div>
          </DropdownMenuItem>
        ))}
      </DropdownMenuContent>
    </DropdownMenu>
  );
}
