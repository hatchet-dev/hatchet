import { Button } from '@/components/v1/ui/button';
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from '@/components/v1/ui/dropdown-menu';
import {
  Notification,
  NotificationColor,
  useNotifications,
} from '@/hooks/notifications';
import { useCurrentUser } from '@/hooks/use-current-user';
import { cn } from '@/lib/utils';
import { useNavigate } from '@tanstack/react-router';
import { RiNotification3Line } from 'react-icons/ri';

const colorToTailwind: Record<NotificationColor, string> = {
  blue: 'bg-blue-500',
  green: 'bg-green-500',
  yellow: 'bg-yellow-500',
  red: 'bg-red-500',
};

const colorPriority: Record<NotificationColor, number> = {
  blue: 0,
  green: 1,
  yellow: 2,
  red: 3,
};

const getMostSevereColor = (notifications: Notification[]): NotificationColor =>
  notifications.reduce<NotificationColor>(
    (worst, n) =>
      colorPriority[n.color] > colorPriority[worst] ? n.color : worst,
    'green',
  );

export function Notifications() {
  const { notifications } = useNotifications();
  const { currentUser } = useCurrentUser();
  const navigate = useNavigate();
  const count = notifications.length;
  const mostSevere = count > 0 ? getMostSevereColor(notifications) : null;
  const notificationLabel = `${count} notification${count !== 1 ? 's' : ''}`;
  const ariaLabel = currentUser?.email
    ? `${notificationLabel} for ${currentUser.email}`
    : notificationLabel;
  const displayTitle =
    count > 0 ? notifications[0].shortTitle : 'Notifications';
  const hasNotifications = count > 0;

  if (!hasNotifications) {
    return null;
  }

  return (
    <DropdownMenu>
      <DropdownMenuTrigger asChild>
        <Button
          variant="icon"
          title={ariaLabel}
          aria-label={ariaLabel}
          data-cy="notifications-button"
          className="relative gap-1.5 rounded-md border border-border px-2.5"
          size="sm"
        >
          <RiNotification3Line className="size-4" />
          <span className="text-xs font-medium leading-none">
            {displayTitle}
          </span>
          {count > 1 && (
            <span className="text-xs font-medium leading-none text-muted-foreground">
              {count}
            </span>
          )}
          {mostSevere && (
            <span
              className={cn(
                'absolute -top-0.5 -right-0.5 size-2.5 rounded-full border-2 border-background',
                colorToTailwind[mostSevere],
              )}
            />
          )}
        </Button>
      </DropdownMenuTrigger>
      <DropdownMenuContent className="w-80" align="end">
        {notifications.map((notification, i) => (
          <DropdownMenuItem
            key={`${notification.url}-${i}`}
            variant="interactive"
            className="flex cursor-pointer items-start gap-2 px-3 py-2"
            onClick={() => navigate({ to: notification.url })}
          >
            <span
              className={cn(
                'mt-1.5 size-2 shrink-0 rounded-full',
                colorToTailwind[notification.color],
              )}
            />
            <div className="min-w-0 flex-1" title={notification.message}>
              <p className="truncate text-sm font-medium">
                {notification.title}
              </p>
              <p className="truncate text-xs text-muted-foreground">
                {notification.message}
              </p>
            </div>
          </DropdownMenuItem>
        ))}
      </DropdownMenuContent>
    </DropdownMenu>
  );
}
