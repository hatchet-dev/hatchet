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
import useWorkers from '@/next/hooks/use-workers';

// Extending Worker type with additional properties that may exist
interface IWorkerDetails {
  language?: string;
  languageVersion?: string;
  os?: string;
  sdkVersion?: string;
  lastListenerEstablished?: string;
  runtimeExtra?: string;
}

interface WorkerDetailsProps {
  worker?: any;
  workerId?: string;
  showActions?: boolean;
}

export function WorkerDetails({
  worker: providedWorker,
  workerId,
  showActions = true,
}: WorkerDetailsProps) {
  const { data: workers = [], update } = useWorkers({
    initialPagination: { currentPage: 1, pageSize: 100 },
    refetchInterval: 5000,
  });

  // Find the worker if only ID is provided
  const worker =
    providedWorker || workers.find((w) => w.metadata.id === workerId);

  if (!worker) {
    return (
      <div className="text-center text-muted-foreground py-8">
        Worker not found
      </div>
    );
  }

  // Cast worker to include additional properties
  const workerDetails = worker as unknown as typeof worker & IWorkerDetails;

  // Status management
  const handlePauseWorker = async () => {
    if (!worker) {
      return;
    }

    try {
      await update.mutateAsync({
        workerId: worker.metadata.id,
        data: { isPaused: true },
      });
    } catch (error) {
      console.error('Failed to pause worker:', error);
    }
  };

  const handleResumeWorker = async () => {
    if (!worker) {
      return;
    }

    try {
      await update.mutateAsync({
        workerId: worker.metadata.id,
        data: { isPaused: false },
      });
    } catch (error) {
      console.error('Failed to resume worker:', error);
    }
  };

  // Calculate time since last heartbeat
  const getTimeSinceLastHeartbeat = () => {
    if (!worker?.lastHeartbeatAt) {
      return 'Never connected';
    }

    const lastHeartbeat = new Date(worker.lastHeartbeatAt);
    const now = new Date();
    const diffMs = now.getTime() - lastHeartbeat.getTime();

    // Convert to seconds, minutes, hours
    const diffSeconds = Math.floor(diffMs / 1000);
    const diffMinutes = Math.floor(diffSeconds / 60);
    const diffHours = Math.floor(diffMinutes / 60);
    const diffDays = Math.floor(diffHours / 24);

    if (diffDays > 0) {
      return `${diffDays} day${diffDays > 1 ? 's' : ''} ago`;
    }
    if (diffHours > 0) {
      return `${diffHours} hour${diffHours > 1 ? 's' : ''} ago`;
    }
    if (diffMinutes > 0) {
      return `${diffMinutes} minute${diffMinutes > 1 ? 's' : ''} ago`;
    }
    return `${diffSeconds} second${diffSeconds !== 1 ? 's' : ''} ago`;
  };

  return (
    <div className="flex flex-1 flex-col gap-4">
      <div className="grid grid-cols-1 gap-4">
        {/* Main worker info */}
        <Card>
          <CardHeader>
            <div className="flex justify-between items-start">
              <div>
                <CardTitle className="flex items-center">
                  {worker.name}
                </CardTitle>
                <CardDescription>
                  Worker ID: {worker.metadata.id}
                </CardDescription>
              </div>
              {showActions && (
                <div className="flex gap-2">
                  <div className="flex items-center gap-2">
                    <WorkerStatusBadge
                      status={worker.status}
                      variant="outline"
                    />
                    {worker.status === 'ACTIVE' && (
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
                    {worker.status === 'PAUSED' && (
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
                      <p className="font-medium">{worker.type || 'Standard'}</p>
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
                          date={worker.lastHeartbeatAt}
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
                          date={worker.metadata.createdAt}
                          variant="timeSince"
                        />
                      </p>
                    </div>
                  </div>
                  <div className="flex items-center">
                    <Activity className="h-5 w-5 mr-2 text-muted-foreground" />
                    <div>
                      <p className="text-sm text-muted-foreground">Status</p>
                      <p className="font-medium">{worker.status}</p>
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
                        date={worker.metadata.createdAt}
                        variant="timeSince"
                      />
                    </span>
                  </div>
                  <div className="flex justify-between items-center">
                    <span className="text-sm">Last Updated</span>
                    <span className="text-sm">
                      <Time
                        date={worker.metadata.updatedAt}
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
