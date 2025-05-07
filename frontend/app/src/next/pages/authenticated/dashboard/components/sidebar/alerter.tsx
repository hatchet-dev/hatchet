import {
  DropdownMenu,
  DropdownMenuTrigger,
  DropdownMenuContent,
  DropdownMenuItem,
} from '@/next/components/ui/dropdown-menu';
import { Button } from '@/next/components/ui/button';
import { Avatar, AvatarFallback } from '@/next/components/ui/avatar';
import { Bell } from 'lucide-react';
import useNotifications from '@/next/hooks/use-notifications';
import { InviteCard } from '@/next/pages/authenticated/onboarding/invites/invites.page';
import { useEffect, useState } from 'react';
import {
  Tooltip,
  TooltipContent,
  TooltipProvider,
  TooltipTrigger,
} from '@/next/components/ui/tooltip';
import { cn } from '@/next/lib/utils';

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
              {alert.invite && <InviteCard invite={alert.invite} />}
            </div>
          </DropdownMenuItem>
        ))}
      </DropdownMenuContent>
    </DropdownMenu>
  );
}
