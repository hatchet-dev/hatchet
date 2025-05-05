import { ReactNode } from 'react';
import { Card, CardContent } from '@/next/components/ui/card';
interface TwoColumnLayoutProps {
  left: ReactNode;
  right: ReactNode;
  leftClassName?: string;
  rightClassName?: string;
}

export function TwoColumnLayout({
  left,
  right,
  leftClassName = '',
  rightClassName = '',
}: TwoColumnLayoutProps) {
  return (
    <div className="flex h-[calc(100vh-4rem)]">
      {/* Left panel */}
      <div className={`flex-1 overflow-y-auto p-4 ${leftClassName}`}>
        {left}
      </div>

      {/* Right panel */}
      <div className={`w-1/2 flex-1 p-4 ${rightClassName}`}>
        <Card className="h-full">
          <CardContent className="space-y-6 py-4 h-full overflow-y-auto">
            {right}
          </CardContent>
        </Card>
      </div>
    </div>
  );
}
