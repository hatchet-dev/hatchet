import { useMemo } from 'react';
import { Worker } from '@/lib/api/generated/data-contracts';
import {
  Tooltip,
  TooltipContent,
  TooltipProvider,
  TooltipTrigger,
} from '@/next/components/ui/tooltip';
import { Link } from 'react-router-dom';
import { ROUTES } from '@/next/lib/routes';
import { useCurrentTenantId } from '@/next/hooks/use-tenant';

interface WorkerIdProps {
  worker?: Worker;
  poolName: string;
  onClick?: () => void;
}

export function WorkerId({
  worker: providedWorker,
  poolName,
  onClick,
}: WorkerIdProps) {
  const { tenantId } = useCurrentTenantId();
  const worker = providedWorker;

  const name = useMemo(() => {
    if (!worker) {
      return 'Worker not found';
    }
    return getFriendlyWorkerId(worker);
  }, [worker]);

  const url = worker
    ? ROUTES.workers.workerDetail(
        tenantId,
        poolName,
        worker.metadata.id,
        worker.type,
      )
    : undefined;

  return (
    <TooltipProvider>
      <Tooltip>
        <TooltipTrigger asChild>
          <span>
            {url && !onClick ? (
              <Link to={url} className="hover:underline text-foreground">
                {name}-{worker?.metadata.id.split('-')[0]}
              </Link>
            ) : (
              <span
                className={onClick ? 'cursor-pointer' : ''}
                onClick={onClick}
              >
                {name}-{worker?.metadata.id.split('-')[0]}
              </span>
            )}
          </span>
        </TooltipTrigger>
        <TooltipContent className="bg-muted">
          <div className="font-mono text-foreground">
            {worker?.metadata.id || ''}
          </div>
        </TooltipContent>
      </Tooltip>
    </TooltipProvider>
  );
}

function getFriendlyWorkerId(worker: Worker) {
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
