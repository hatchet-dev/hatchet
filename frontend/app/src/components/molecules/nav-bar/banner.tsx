import React from 'react';
import { Button } from '@/components/ui/button';

export interface BannerProps {
  message: React.ReactNode;
  type?: 'info' | 'warning' | 'success' | 'error';
  actionText?: string;
  onAction?: () => void;
  showBanner?: boolean;
}

export const Banner: React.FC<BannerProps> = ({
  message,
  type = 'info',
  actionText,
  onAction,
  showBanner = true,
}) => {
  if (!showBanner) {
    return null;
  }

  const getBgColor = () => {
    switch (type) {
      case 'warning':
        return 'bg-amber-50 dark:bg-amber-900/20 text-amber-800 dark:text-amber-200';
      case 'success':
        return 'bg-green-50 dark:bg-green-900/20 text-green-800 dark:text-green-200';
      case 'error':
        return 'bg-red-50 dark:bg-red-900/20 text-red-800 dark:text-red-200';
      default:
        return 'bg-blue-50 dark:bg-blue-900/20 text-blue-800 dark:text-blue-200';
    }
  };

  return (
    <div className={`w-full py-2 px-4 h-12 ${getBgColor()}`}>
      <div className="flex items-center justify-between">
        <div className="flex flex-1 items-center">
          <p className="text-sm font-medium">{message}</p>
        </div>
        <div className="flex items-center gap-2">
          {actionText && onAction && (
            <Button
              variant="outline"
              size="sm"
              onClick={onAction}
              className="text-xs font-medium hover:bg-opacity-20"
            >
              {actionText}
            </Button>
          )}
        </div>
      </div>
    </div>
  );
};
