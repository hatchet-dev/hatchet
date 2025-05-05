import { useWorkers, WorkersProvider } from '@/next/hooks/use-workers';
import { Loader2, CheckCircle2 } from 'lucide-react';

interface WorkerListenerProps {
  name: string;
}

export function WorkerListener(props: WorkerListenerProps) {
  return (
    <WorkersProvider refetchInterval={1000}>
      <WorkerConnection name={props.name} />
    </WorkersProvider>
  );
}

function WorkerConnection({ name }: { name: string }) {
  const { data: workers } = useWorkers();

  return (
    <div className="flex items-center gap-2 p-4 bg-muted rounded-lg">
      {workers
        ?.filter((w) => w.name === name)
        .filter((w) => w.status === 'ACTIVE').length === 0 ? (
        <>
          <Loader2 className="h-4 w-4 animate-spin" />
          <span>
            Waiting for <pre className="inline">{name}</pre> to connect...
          </span>
        </>
      ) : (
        <>
          <CheckCircle2 className="h-4 w-4 text-green-500" />
          <span>Worker connected successfully!</span>
        </>
      )}
    </div>
  );
}
