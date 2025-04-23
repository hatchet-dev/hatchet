import { FC } from 'react';
import { WorkerTable } from '../../components/worker-table';
interface WorkersTabProps {
  serviceName: string;
}

export const WorkersTab: FC<WorkersTabProps> = ({ serviceName }) => {
  return (
    <>
      <WorkerTable serviceName={serviceName} />
    </>
  );
};
