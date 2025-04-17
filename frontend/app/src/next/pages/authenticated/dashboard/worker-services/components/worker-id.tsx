import { useMemo } from 'react';
import { Worker } from '@/next/lib/api/generated/data-contracts';
import useWorkers from '@/next/hooks/use-workers';

interface WorkerIdProps {
  worker?: Worker;
  workerId?: string;
}

export function WorkerId({ worker: providedWorker, workerId }: WorkerIdProps) {
  const { data: workers = [] } = useWorkers({
    initialPagination: { currentPage: 1, pageSize: 100 },
    refetchInterval: 5000,
  });

  const worker =
    providedWorker || workers.find((w: Worker) => w.metadata.id === workerId);

  const name = useMemo(() => {
    if (!worker) {
      return 'Worker not found';
    }
    return worker.metadata.id;
  }, [worker]);

  return <span>{name}</span>;
}

export function getFriendlyWorkerId(worker: Worker) {
  if (!worker) {
    return '';
  }

  // If the worker has a name, use that
  if (worker.name) {
    return worker.name;
  }

  // Otherwise, use the ID
  return worker.metadata.id;
}
