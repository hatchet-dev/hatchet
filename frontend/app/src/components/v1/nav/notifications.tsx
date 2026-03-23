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
  green: 'bg-green-500',
  yellow: 'bg-yellow-500',
  red: 'bg-red-500',
};

const colorPriority: Record<NotificationColor, number> = {
  green: 0,
  yellow: 1,
  red: 2,
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
  const title = `${count} notification${count !== 1 ? 's' : ''} for ${currentUser?.email ?? ''}`;

  return (
    <DropdownMenu>
      <DropdownMenuTrigger asChild>
        <button
          type="button"
          title={title}
          aria-label={title}
          className="relative flex size-8 aspect-square items-center justify-center rounded-full border border-border bg-muted/20 text-foreground hover:bg-muted/40"
        >
          {count > 0 ? (
            <span className="text-xs font-medium leading-none">{count}</span>
          ) : (
            <RiNotification3Line className="size-4" />
          )}
          {mostSevere && (
            <span
              className={cn(
                'absolute -bottom-0.5 -right-0.5 size-2.5 rounded-full border-2 border-background',
                colorToTailwind[mostSevere],
              )}
            />
          )}
        </button>
      </DropdownMenuTrigger>
      <DropdownMenuContent className="w-80" align="end">
        {count === 0 ? (
          <div className="px-3 py-4 text-center text-sm text-muted-foreground">
            No notifications
          </div>
        ) : (
          notifications.map((notification, i) => (
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
          ))
        )}
      </DropdownMenuContent>
    </DropdownMenu>
  );
}
