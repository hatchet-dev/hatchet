export type NotificationColor = 'blue' | 'green' | 'yellow' | 'red';

type NotificationBase = {
  color: NotificationColor;
  shortTitle: string;
  title: string;
  message: string;
  timestamp: string;
};

export type Notification =
  | (NotificationBase & { url: string; onClick?: never })
  | (NotificationBase & { onClick: () => void; url?: never });
