import GithubButton from './github-button';
import { ManagedWorkerBuild } from './managed-worker-build';
import { ManagedWorkerIaC } from './managed-worker-iac';
import RelativeDate from '@/components/v1/molecules/relative-date';
import { Button } from '@/components/v1/ui/button';
import {
  Card,
  CardContent,
  CardDescription,
  CardFooter,
  CardHeader,
  CardTitle,
} from '@/components/v1/ui/card';
import { Spinner } from '@/components/v1/ui/loading';
import { Separator } from '@/components/v1/ui/separator';
import { useRefetchInterval } from '@/contexts/refetch-interval-context';
import { queries } from '@/lib/api';
import {
  ManagedWorker,
  ManagedWorkerEvent,
  ManagedWorkerEventStatus,
} from '@/lib/api/generated/cloud/data-contracts';
import { cn } from '@/lib/utils';
import { ArrowRightIcon, ChevronLeftIcon } from '@radix-ui/react-icons';
import { useQuery } from '@tanstack/react-query';
import { useState } from 'react';

export function ManagedWorkerActivity({
  managedWorker,
}: {
  managedWorker: ManagedWorker | undefined;
}) {
  const [buildId, setBuildId] = useState<string | undefined>();
  const [iacDeployKey, setIacDeployKey] = useState<string | undefined>();

  if (buildId) {
    return <Build buildId={buildId} back={() => setBuildId(undefined)} />;
  }

  if (iacDeployKey) {
    return (
      <IaCDebug
        managedWorkerId={managedWorker!.metadata.id}
        deployKey={iacDeployKey}
        back={() => setIacDeployKey(undefined)}
      />
    );
  }

  return (
    <EventList
      managedWorker={managedWorker}
      setBuildId={setBuildId}
      setDeployKey={setIacDeployKey}
    />
  );
}

function EventList({
  managedWorker,
  setBuildId,
  setDeployKey,
}: {
  managedWorker: ManagedWorker | undefined;
  setBuildId: (id: string) => void;
  setDeployKey: (key: string) => void;
}) {
  const { refetchInterval } = useRefetchInterval();

  const getLogsQuery = useQuery({
    ...queries.cloud.listManagedWorkerEvents(managedWorker!.metadata.id || ''),
    enabled: !!managedWorker,
    refetchInterval,
  });

  if (!managedWorker || getLogsQuery.isLoading) {
    return <Spinner />;
  }

  const events = getLogsQuery.data?.rows || [];

  return (
    <div className="mt-4 flex flex-col gap-2">
      {getLogsQuery.isLoading && <Spinner />}
      {events.length === 0 && (
        <Card className="h-[400px] bg-muted/30">
          <CardHeader>
            <CardTitle className="text-sm tracking-wide">
              No events found
            </CardTitle>
          </CardHeader>
        </Card>
      )}
      {events.map((item, index) => (
        <ManagedWorkerEventCard
          key={index}
          managedWorker={managedWorker}
          event={item}
          setBuildId={setBuildId}
          setDeployKey={setDeployKey}
        />
      ))}
    </div>
  );
}

function Build({ buildId, back }: { buildId: string; back: () => void }) {
  return (
    <div className="mt-8 flex flex-col items-start justify-start gap-4">
      <div className="flex flex-row items-center justify-start gap-4">
        <Button
          onClick={back}
          variant="link"
          leftIcon={<ChevronLeftIcon className="size-4" />}
        >
          Back
        </Button>
      </div>
      <Separator />
      <ManagedWorkerBuild buildId={buildId} />
    </div>
  );
}

function IaCDebug({
  managedWorkerId,
  deployKey,
  back,
}: {
  managedWorkerId: string;
  deployKey: string;
  back: () => void;
}) {
  return (
    <div className="mt-8 flex flex-col items-start justify-start gap-4">
      <div className="flex flex-row items-center justify-start gap-4">
        <Button
          onClick={back}
          variant="link"
          leftIcon={<ChevronLeftIcon className="size-4" />}
        >
          Back
        </Button>
      </div>
      <Separator />
      <ManagedWorkerIaC
        managedWorkerId={managedWorkerId}
        deployKey={deployKey}
      />
    </div>
  );
}

function ManagedWorkerEventCard({
  managedWorker,
  event,
  setBuildId,
  setDeployKey,
}: {
  managedWorker: ManagedWorker;
  event: ManagedWorkerEvent;
  setBuildId: (id: string) => void;
  setDeployKey: (key: string) => void;
}) {
  return (
    <Card className="bg-muted/30">
      <CardHeader>
        <div className="flex flex-row items-center justify-between text-sm">
          <div className="flex flex-row items-center justify-between gap-3">
            <EventIndicator severity={event.status} />
            <CardTitle className="text-sm tracking-wide">
              {event.message}
            </CardTitle>
          </div>
          <RelativeDate date={event.timeLastSeen} />
        </div>
        <CardDescription className="mt-2">{event.message}</CardDescription>
      </CardHeader>
      <CardContent className="z-10 bg-background p-0"></CardContent>
      {renderCardFooter(managedWorker, event, setBuildId, setDeployKey)}
    </Card>
  );
}

function renderCardFooter(
  managedWorker: ManagedWorker,
  event: ManagedWorkerEvent,
  setBuildId: (id: string) => void,
  setDeployKey: (key: string) => void,
) {
  if (event.data) {
    const data = event.data as any;

    const buttons = [];

    if (data.build_id) {
      buttons.push(
        <Button
          variant="link"
          size="xs"
          onClick={() => {
            setBuildId(data.build_id);
          }}
          leftIcon={<ArrowRightIcon className="size-4" />}
        >
          View build info
        </Button>,
      );
    }

    if (data.deploy_key) {
      buttons.push(
        <Button
          variant="link"
          size="xs"
          onClick={() => {
            setDeployKey(data.deploy_key);
          }}
          leftIcon={<ArrowRightIcon className="size-4" />}
        >
          View IaC debug info
        </Button>,
      );
    }

    if (data.commit_sha && managedWorker.buildConfig) {
      buttons.push(
        <GithubButton
          buildConfig={managedWorker.buildConfig}
          commitSha={data.commit_sha}
        />,
      );
    }

    if (buttons.length) {
      return <CardFooter className="gap-4">{buttons}</CardFooter>;
    }
  }

  return null;
}

const RUN_STATUS_VARIANTS: Record<ManagedWorkerEventStatus, string> = {
  SUCCEEDED: 'border-transparent rounded-full bg-green-500',
  FAILED: 'border-transparent rounded-full bg-red-500',
  CANCELLED: 'border-transparent rounded-full bg-gray-500',
  IN_PROGRESS: 'border-transparent rounded-full bg-yellow-500',
  SCALE_UP: 'border-transparent rounded-full bg-blue-500',
  SCALE_DOWN: 'border-transparent rounded-full bg-purple-500',
};

function EventIndicator({ severity }: { severity: ManagedWorkerEventStatus }) {
  return (
    <div
      className={cn(
        RUN_STATUS_VARIANTS[severity],
        'h-[6px] w-[6px] rounded-full',
      )}
    />
  );
}
