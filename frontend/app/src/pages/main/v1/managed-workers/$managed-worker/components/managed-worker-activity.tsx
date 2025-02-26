import { queries } from '@/lib/api';
import { useQuery } from '@tanstack/react-query';
import {
  Card,
  CardContent,
  CardDescription,
  CardFooter,
  CardHeader,
  CardTitle,
} from '@/components/v1/ui/card';
import { Spinner } from '@/components/v1/ui/loading';
import RelativeDate from '@/components/v1/molecules/relative-date';
import { Button } from '@/components/v1/ui/button';
import { ArrowRightIcon, ChevronLeftIcon } from '@radix-ui/react-icons';
import { cn } from '@/lib/utils';
import { useState } from 'react';
import {
  ManagedWorker,
  ManagedWorkerEvent,
  ManagedWorkerEventStatus,
} from '@/lib/api/generated/cloud/data-contracts';
import { Separator } from '@/components/v1/ui/separator';
import { ManagedWorkerBuild } from './managed-worker-build';
import GithubButton from './github-button';
import { ManagedWorkerIaC } from './managed-worker-iac';

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
  const getLogsQuery = useQuery({
    ...queries.cloud.listManagedWorkerEvents(managedWorker!.metadata.id || ''),
    enabled: !!managedWorker,
    refetchInterval: () => {
      return 5000;
    },
  });

  if (!managedWorker || getLogsQuery.isLoading) {
    return <Spinner />;
  }

  const events = getLogsQuery.data?.rows || [];

  return (
    <div className="flex flex-col gap-2 mt-4">
      {getLogsQuery.isLoading && <Spinner />}
      {events.length === 0 && (
        <Card className="bg-muted/30 h-[400px]">
          <CardHeader>
            <CardTitle className="tracking-wide text-sm">
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
    <div className="flex flex-col justify-start items-start gap-4 mt-8">
      <div className="flex flex-row justify-start gap-4 items-center">
        <Button
          onClick={back}
          variant="link"
          className="flex items-center gap-1"
        >
          <ChevronLeftIcon className="w-4 h-4" />
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
    <div className="flex flex-col justify-start items-start gap-4 mt-8">
      <div className="flex flex-row justify-start gap-4 items-center">
        <Button
          onClick={back}
          variant="link"
          className="flex items-center gap-1"
        >
          <ChevronLeftIcon className="w-4 h-4" />
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
    <Card className=" bg-muted/30">
      <CardHeader>
        <div className="flex flex-row justify-between items-center text-sm">
          <div className="flex flex-row justify-between gap-3 items-center">
            <EventIndicator severity={event.status} />
            <CardTitle className="tracking-wide text-sm">
              {event.message}
            </CardTitle>
          </div>
          <RelativeDate date={event.timeLastSeen} />
        </div>
        <CardDescription className="mt-2">{event.message}</CardDescription>
      </CardHeader>
      <CardContent className="p-0 z-10 bg-background"></CardContent>
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
        >
          <ArrowRightIcon className="w-4 h-4 mr-1" />
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
        >
          <ArrowRightIcon className="w-4 h-4 mr-1" />
          View IaC debug info
        </Button>,
      );
    }

    if (data.commit_sha) {
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
};

function EventIndicator({ severity }: { severity: ManagedWorkerEventStatus }) {
  return (
    <div
      className={cn(
        RUN_STATUS_VARIANTS[severity],
        'rounded-full h-[6px] w-[6px]',
      )}
    />
  );
}
