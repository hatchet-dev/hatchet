import { useMemo } from 'react';
import { Worker } from '@/lib/api/generated/data-contracts';

interface WorkerIdProps {
  worker?: Worker;
}

export function WorkerId({ worker: providedWorker }: WorkerIdProps) {
  const worker = providedWorker;

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
