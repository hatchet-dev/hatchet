import {
  Card,
  CardContent,
  CardHeader,
  CardTitle,
  CardDescription,
} from '@/next/components/ui/card';
import { Button } from '@/next/components/ui/button';
import { Separator } from '@/next/components/ui/separator';
import {
  Play,
  Pause,
  Server,
  Box,
  Activity,
  HardDrive,
  Cpu,
  Clock,
} from 'lucide-react';
import { WorkerStatusBadge } from './worker-status-badge';
import { Time } from '@/next/components/ui/time';
import {
  WorkerDetailProvider,
  useWorkerDetail,
} from '@/next/hooks/use-worker-detail';
import { WrongTenant } from '@/next/components/errors/unauthorized';
import { useTenant } from '@/next/hooks/use-tenant';
import { formatDuration } from '@/next/lib/utils/formatDuration';
import { intervalToDuration } from 'date-fns';
// Extending Worker type with additional properties that may exist
interface IWorkerDetails {
  language?: string;
  languageVersion?: string;
  os?: string;
  sdkVersion?: string;
  lastListenerEstablished?: string;
  runtimeExtra?: string;
}

export interface WorkerDetailsProps {
  worker?: any;
  workerId?: string;
  showActions?: boolean;
}

export function WorkerDetails({
  worker,
  workerId,
  showActions = true,
}: WorkerDetailsProps) {
  return (
    <WorkerDetailProvider workerId={workerId}>
      <WorkerDetailsContent
        worker={worker}
        workerId={workerId}
        showActions={showActions}
      />
    </WorkerDetailProvider>
  );
}

function WorkerDetailsContent({
  worker: providedWorker,
  showActions = true,
}: WorkerDetailsProps) {
  const { tenant } = useTenant();
  const { data: workerDetail, update } = useWorkerDetail();

  // Use provided worker if available, otherwise use the one from the hook
  const currentWorker = providedWorker || workerDetail;

  if (!currentWorker) {
    return (
      <div className="text-center text-muted-foreground py-8">
        Worker not found
      </div>
    );
  }

  // Cast worker to include additional properties
  const workerDetails = currentWorker as unknown as typeof currentWorker &
    IWorkerDetails;

  // Status management
  const handlePauseWorker = async () => {
    if (!currentWorker) {
      return;
    }

    try {
      await update.mutateAsync({
        workerId: currentWorker.metadata.id,
        data: { isPaused: true },
      });
    } catch (error) {
      console.error('Failed to pause worker:', error);
    }
  };

  const handleResumeWorker = async () => {
    if (!currentWorker) {
      return;
    }

    try {
      await update.mutateAsync({
        workerId: currentWorker.metadata.id,
        data: { isPaused: false },
      });
    } catch (error) {
      console.error('Failed to resume worker:', error);
    }
  };

  // Calculate time since last heartbeat
  const getTimeSinceLastHeartbeat = () => {
    if (!currentWorker?.lastHeartbeatAt) {
      return 'Never connected';
    }

    return formatDuration(
      intervalToDuration({
        start: currentWorker.lastHeartbeatAt,
        end: new Date(),
      }),
      new Date().getTime() - new Date(currentWorker.lastHeartbeatAt).getTime(),
    );
  };

  // wrong tenant selected error
  if (tenant?.metadata.id !== currentWorker.tenantId) {
    return (
      <div className="flex flex-1 flex-col gap-4 p-4">
        {currentWorker?.tenantId && (
          <WrongTenant desiredTenantId={currentWorker.tenantId} />
        )}
      </div>
    );
  }

  return (
    <div className="flex flex-1 flex-col gap-4">
      <div className="grid grid-cols-1 gap-4">
        {/* Main worker info */}
        <Card>
          <CardHeader>
            <div className="flex justify-between items-start">
              <div>
                <CardTitle className="flex items-center">
                  {currentWorker.name}
                </CardTitle>
                <CardDescription>
                  Worker ID: {currentWorker.metadata.id}
                </CardDescription>
              </div>
              {showActions && (
                <div className="flex gap-2">
                  <div className="flex items-center gap-2">
                    <WorkerStatusBadge
                      status={currentWorker.status}
                      variant="outline"
                    />
                    {currentWorker.status === 'ACTIVE' && (
                      <Button
                        size="sm"
                        variant="outline"
                        title="Pause assigning new tasks"
                        onClick={() => handlePauseWorker()}
                      >
                        <Pause className="h-4 w-4 mr-1" />
                        <span className="sr-only">Pause</span>
                      </Button>
                    )}
                    {currentWorker.status === 'PAUSED' && (
                      <Button
                        size="sm"
                        variant="outline"
                        title="Resume assigning new tasks"
                        onClick={() => handleResumeWorker()}
                      >
                        <Play className="h-4 w-4 mr-1" />
                        <span className="sr-only">Resume</span>
                      </Button>
                    )}
                  </div>
                </div>
              )}
            </div>
          </CardHeader>
          <CardContent>
            <div className="space-y-4">
              <div>
                <h3 className="text-lg font-medium mb-2">Worker Details</h3>
                <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
                  <div className="flex items-center">
                    <Server className="h-5 w-5 mr-2 text-muted-foreground" />
                    <div>
                      <p className="text-sm text-muted-foreground">Type</p>
                      <p className="font-medium">
                        {currentWorker.type || 'Standard'}
                      </p>
                    </div>
                  </div>
                  <div className="flex items-center">
                    <Clock className="h-5 w-5 mr-2 text-muted-foreground" />
                    <div>
                      <p className="text-sm text-muted-foreground">
                        Last Heartbeat
                      </p>
                      <p className="font-medium">
                        <Time
                          date={currentWorker.lastHeartbeatAt}
                          variant="timeSince"
                        />
                      </p>
                    </div>
                  </div>
                  <div className="flex items-center">
                    <Box className="h-5 w-5 mr-2 text-muted-foreground" />
                    <div>
                      <p className="text-sm text-muted-foreground">
                        Created At
                      </p>
                      <p className="font-medium">
                        <Time
                          date={currentWorker.metadata.createdAt}
                          variant="timeSince"
                        />
                      </p>
                    </div>
                  </div>
                  <div className="flex items-center">
                    <Activity className="h-5 w-5 mr-2 text-muted-foreground" />
                    <div>
                      <p className="text-sm text-muted-foreground">Status</p>
                      <p className="font-medium">{currentWorker.status}</p>
                    </div>
                  </div>
                </div>
              </div>

              <Separator />

              <div>
                <h3 className="text-lg font-medium mb-2">
                  Runtime Information
                </h3>
                <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
                  <div className="flex items-center">
                    <Cpu className="h-5 w-5 mr-2 text-muted-foreground" />
                    <div>
                      <p className="text-sm text-muted-foreground">Language</p>
                      <p className="font-medium">
                        {workerDetails.language || 'Not specified'}
                      </p>
                    </div>
                  </div>
                  <div className="flex items-center">
                    <HardDrive className="h-5 w-5 mr-2 text-muted-foreground" />
                    <div>
                      <p className="text-sm text-muted-foreground">
                        Operating System
                      </p>
                      <p className="font-medium">
                        {workerDetails.os || 'Not specified'}
                      </p>
                    </div>
                  </div>
                  <div className="flex items-center">
                    <Box className="h-5 w-5 mr-2 text-muted-foreground" />
                    <div>
                      <p className="text-sm text-muted-foreground">
                        SDK Version
                      </p>
                      <p className="font-medium">
                        {workerDetails.sdkVersion || 'Not specified'}
                      </p>
                    </div>
                  </div>
                  <div className="flex items-center">
                    <Activity className="h-5 w-5 mr-2 text-muted-foreground" />
                    <div>
                      <p className="text-sm text-muted-foreground">
                        Language Version
                      </p>
                      <p className="font-medium">
                        {workerDetails.languageVersion || 'Not specified'}
                      </p>
                    </div>
                  </div>
                </div>
              </div>
            </div>
          </CardContent>
        </Card>

        {/* Status card */}
        <Card>
          <CardHeader>
            <CardTitle>Health Status</CardTitle>
          </CardHeader>
          <CardContent>
            <div className="space-y-4">
              <div className="flex flex-col">
                <span className="text-sm text-muted-foreground">
                  Last Heartbeat
                </span>
                <span className="text-xl font-bold">
                  {getTimeSinceLastHeartbeat()}
                </span>
              </div>

              <Separator />

              <div>
                <h3 className="font-medium mb-2">Status History</h3>
                <div className="space-y-2">
                  <div className="flex justify-between items-center">
                    <span className="text-sm">Created</span>
                    <span className="text-sm">
                      <Time
                        date={currentWorker.metadata.createdAt}
                        variant="timeSince"
                      />
                    </span>
                  </div>
                  <div className="flex justify-between items-center">
                    <span className="text-sm">Last Updated</span>
                    <span className="text-sm">
                      <Time
                        date={currentWorker.metadata.updatedAt}
                        variant="timeSince"
                      />
                    </span>
                  </div>
                  <div className="flex justify-between items-center">
                    <span className="text-sm">Last Connected</span>
                    <span className="text-sm">
                      <Time
                        date={workerDetails.lastListenerEstablished}
                        variant="timeSince"
                      />
                    </span>
                  </div>
                </div>
              </div>

              <Separator />

              <div>
                <Button variant="outline" size="sm" className="w-full">
                  View Full Logs
                </Button>
              </div>
            </div>
          </CardContent>
        </Card>

        {/* Recent tasks card */}
        <Card>
          <CardHeader>
            <CardTitle>Recent Tasks</CardTitle>
            <CardDescription>Tasks executed by this worker</CardDescription>
          </CardHeader>
          <CardContent>
            <div className="text-center text-muted-foreground py-8">
              Task history is not available in this view
            </div>
          </CardContent>
        </Card>
      </div>
    </div>
  );
}
