import { Link } from 'react-router-dom';
import {
  Tooltip,
  TooltipContent,
  TooltipProvider,
  TooltipTrigger,
} from '@/next/components/ui/tooltip';
import { Code } from '@/next/components/ui/code';

interface WorkerIdProps {
  worker: {
    metadata: {
      id: string;
    };
    name: string;
  };
}

function getFriendlyWorkerId(worker: WorkerIdProps['worker']) {
  return `${worker.name}/${worker.metadata.id.substring(0, 8)}...`;
}

export function WorkerId({ worker }: WorkerIdProps) {
  return (
    <TooltipProvider>
      <Tooltip>
        <TooltipTrigger asChild>
          <span>
            <Link
              to={`/services/${encodeURIComponent(worker.name)}/${encodeURIComponent(
                worker.metadata.id,
              )}`}
              className="hover:underline text-primary"
            >
              {getFriendlyWorkerId(worker)}
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
