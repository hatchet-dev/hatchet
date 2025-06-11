import { useManagedComputeDetail } from '@/next/hooks/use-managed-compute-detail';
import { FC, useState } from 'react';
import { Button } from '@/next/components/ui/button';
import { ArrowPathIcon, ChevronLeftIcon } from '@heroicons/react/24/outline';
import {
  Card,
  CardContent,
  CardDescription,
  CardFooter,
  CardHeader,
  CardTitle,
} from '@/next/components/ui/card';
import { Spinner } from '@/next/components/ui/spinner';
import { Separator } from '@/next/components/ui/separator';
import {
  ManagedWorker,
  ManagedWorkerEvent,
  ManagedWorkerEventStatus,
} from '@/lib/api/generated/cloud/data-contracts';
import RelativeDate from '@/next/components/ui/relative-date';
import { cn } from '@/next/lib/utils';
import { ArrowRightIcon } from '@radix-ui/react-icons';
import { ManagedWorkerBuild } from './managed-worker-build';

export const BuildsTab: FC = () => {
  const { data: managedWorker } = useManagedComputeDetail();
  const [buildId, setBuildId] = useState<string | undefined>();
  const [iacDeployKey, setIacDeployKey] = useState<string | undefined>();

  if (buildId) {
    return <Build buildId={buildId} back={() => setBuildId(undefined)} />;
  }

  if (iacDeployKey) {
    return (
      <IaCDebug
        managedWorkerId={managedWorker?.metadata?.id || ''}
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
};

function EventList({
  managedWorker,
  setBuildId,
  setDeployKey,
}: {
  managedWorker: ManagedWorker | undefined;
  setBuildId: (id: string) => void;
  setDeployKey: (key: string) => void;
}) {
  const { events } = useManagedComputeDetail();
  const [rotate, setRotate] = useState(false);

  if (!managedWorker || events?.isLoading) {
    return <Spinner />;
  }

  const eventsList: ManagedWorkerEvent[] = events?.data?.rows || [];

  return (
    <div className="flex flex-col gap-4">
      <div className="flex justify-end">
        <Button
          className="h-8 px-2 lg:px-3"
          size="sm"
          onClick={() => {
            void events?.refetch();
            setRotate(!rotate);
          }}
          variant="outline"
          aria-label="Refresh events list"
        >
          <ArrowPathIcon
            className={`h-4 w-4 transition-transform ${rotate ? 'rotate-180' : ''}`}
          />
        </Button>
      </div>

      <div className="flex flex-col gap-2">
        {events?.isLoading ? <Spinner /> : null}
        {eventsList.length === 0 && (
          <Card className="bg-muted/30 h-[400px]">
            <CardHeader>
              <CardTitle className="tracking-wide text-sm">
                No events found
              </CardTitle>
            </CardHeader>
          </Card>
        )}
        {eventsList.map((item, index) => (
          <ManagedWorkerEventCard
            key={index}
            managedWorker={managedWorker}
            event={item}
            setBuildId={setBuildId}
            setDeployKey={setDeployKey}
          />
        ))}
      </div>
    </div>
  );
}

function Build({ buildId, back }: { buildId: string; back: () => void }) {
  return (
    <div className="flex flex-col justify-start items-start gap-4">
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
  deployKey,
  back,
}: {
  managedWorkerId: string;
  deployKey: string;
  back: () => void;
}) {
  return (
    <div className="flex flex-col justify-start items-start gap-4">
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
      {/* TODO: Import and use ManagedWorkerIaC component */}
      <div>IaC debug info for {deployKey}</div>
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
          key="build"
          variant="link"
          size="sm"
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
          key="deploy"
          variant="link"
          size="sm"
          onClick={() => {
            setDeployKey(data.deploy_key);
          }}
        >
          <ArrowRightIcon className="w-4 h-4 mr-1" />
          View IaC debug info
        </Button>,
      );
    }

    if (data.commit_sha && managedWorker.buildConfig?.githubRepository) {
      buttons.push(
        <Button
          key="github"
          variant="link"
          size="sm"
          onClick={() => {
            const githubRepo = managedWorker.buildConfig?.githubRepository;
            if (githubRepo) {
              const { repo_owner: repoOwner, repo_name: repoName } = githubRepo;
              const repoUrl = `https://github.com/${repoOwner}/${repoName}`;
              window.open(`${repoUrl}/commit/${data.commit_sha}`, '_blank');
            }
          }}
        >
          <ArrowRightIcon className="w-4 h-4 mr-1" />
          View commit
        </Button>,
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
        'rounded-full h-[6px] w-[6px]',
      )}
    />
  );
}
