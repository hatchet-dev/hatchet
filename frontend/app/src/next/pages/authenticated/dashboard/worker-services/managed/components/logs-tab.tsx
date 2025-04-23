import { FC } from 'react';

interface LogsTabProps {
  serviceName: string;
}

export const LogsTab: FC<LogsTabProps> = ({ serviceName }) => {
  return (
    <div className="p-4">
      <h2 className="text-lg font-semibold mb-4">Logs & Activity</h2>
      <p className="text-muted-foreground">
        This tab will display logs and activity for {serviceName}.
      </p>
    </div>
  );
};
