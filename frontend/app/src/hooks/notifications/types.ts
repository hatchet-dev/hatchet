export type NotificationColor = 'blue' | 'green' | 'yellow' | 'red';

export type Notification = {
  color: NotificationColor;
  shortTitle: string;
  title: string;
  message: string;
  timestamp: string;
  url: string;
};
