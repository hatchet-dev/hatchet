import { FC } from 'react';

interface MetricsTabProps {
  serviceName: string;
}

export const MetricsTab: FC<MetricsTabProps> = ({ serviceName }) => {
  return (
    <div className="p-4">
      <h2 className="text-lg font-semibold mb-4">Metrics</h2>
      <p className="text-muted-foreground">
        This tab will display metrics and performance data for {serviceName}.
      </p>
    </div>
  );
};
