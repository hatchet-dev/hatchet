import { FC } from 'react';

interface WorkersTabProps {
  serviceName: string;
}

export const WorkersTab: FC<WorkersTabProps> = ({ serviceName }) => {
  return (
    <div className="p-4">
      <h2 className="text-lg font-semibold mb-4">Workers</h2>
      <p className="text-muted-foreground">
        This tab will display the list of workers for {serviceName}.
      </p>
    </div>
  );
};
