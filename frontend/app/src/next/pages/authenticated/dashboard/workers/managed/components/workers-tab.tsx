import { FC } from 'react';
import { WorkerTable } from '../../components/worker-table';
interface WorkersTabProps {
  poolName: string;
}

export const WorkersTab: FC<WorkersTabProps> = ({ poolName }) => {
  return <WorkerTable poolName={poolName} />;
};
