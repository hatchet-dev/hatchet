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
import { Worker } from '@/next/lib/api';

interface WorkerIdProps {
  worker: {
    metadata: {
      id: string;
    };
    name: string;
    serviceName: string;
  };
}

export function WorkerId({ worker }: WorkerIdProps) {
  const name = useMemo(() => {
    return getFriendlyWorkerId(worker);
  }, [worker]);

  return (
    <TooltipProvider>
      <Tooltip>
        <TooltipTrigger asChild>
          <span>
            <Link
              to={ROUTES.services.workerDetail(
                encodeURIComponent(worker.serviceName),
                encodeURIComponent(worker.name),
              )}
              className="hover:underline text-blue-500"
            >
              {name}
            </Link>
          </span>
        </TooltipTrigger>
        <TooltipContent>
          <Code
            variant="inline"
            className="font-medium"
            language={'plaintext'}
            value={worker.metadata.id}
          >
            {worker.metadata.id}
          </Code>
        </TooltipContent>
      </Tooltip>
    </TooltipProvider>
  );
}

export function getFriendlyWorkerId(worker: Worker) {
  if (!worker) {
    return;
  }

  return worker.name;
}
