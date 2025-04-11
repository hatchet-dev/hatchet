import {
  Tooltip,
  TooltipContent,
  TooltipProvider,
  TooltipTrigger,
} from '@/next/components/ui/tooltip';
import { Code } from '@/next/components/ui/code';
import { useMemo } from 'react';
import { Link } from 'react-router-dom';
import { ROUTES } from '@/next/lib/routes';
import { Worker } from '@/next/lib/api/generated/data-contracts';

interface WorkerIdProps {
  worker: Worker;
}

export function WorkerId({ worker }: WorkerIdProps) {
  const name = useMemo(() => {
    return getFriendlyWorkerId(worker);
  }, [worker]);

  return (
    <TooltipProvider>
      <Tooltip>
        <TooltipTrigger asChild>
          <Link
            to={ROUTES.services.workerDetail(
              encodeURIComponent(worker.name),
              worker.metadata.id,
            )}
            className="hover:underline"
          >
            <Code language="plaintext" value={name}>
              {name}
            </Code>
          </Link>
        </TooltipTrigger>
        <TooltipContent>
          <p>View worker details</p>
        </TooltipContent>
      </Tooltip>
    </TooltipProvider>
  );
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
