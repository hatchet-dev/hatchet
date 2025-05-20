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
import useTenant from '@/next/hooks/use-tenant';

interface WorkerIdProps {
  worker?: Worker;
  serviceName: string;
  onClick?: () => void;
}

export function WorkerId({
  worker: providedWorker,
  serviceName,
  onClick,
}: WorkerIdProps) {
  const { tenant } = useTenant();
  const worker = providedWorker;

  const name = useMemo(() => {
    if (!worker) {
      return 'Worker not found';
    }
    return getFriendlyWorkerId(worker);
  }, [worker]);

  const url = worker
    ? ROUTES.services.workerDetail(
        tenant?.metadata.id || '',
        serviceName,
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
