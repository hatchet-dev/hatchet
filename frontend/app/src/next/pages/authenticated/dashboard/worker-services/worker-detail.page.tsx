import { useParams, Link } from 'react-router-dom';
import useWorkers from '@/next/hooks/use-workers';
import {
  Card,
  CardContent,
  CardHeader,
  CardTitle,
  CardDescription,
} from '@/next/components/ui/card';
import { Button } from '@/next/components/ui/button';
import { Badge } from '@/next/components/ui/badge';
import { Separator } from '@/next/components/ui/separator';
import {
  ChevronLeft,
  Play,
  Pause,
  StopCircle,
  RefreshCw,
  Clock,
  Server,
  Box,
  Activity,
  HardDrive,
  Cpu,
} from 'lucide-react';
import { WorkerStatusBadge } from './components/worker-status-badge';

// Extending Worker type with additional properties that may exist
interface WorkerDetails {
  language?: string;
  languageVersion?: string;
  os?: string;
  sdkVersion?: string;
  lastListenerEstablished?: string;
  runtimeExtra?: string;
}

export default function WorkerDetailPage() {
  const { serviceName = '', workerId = '' } = useParams<{
    serviceName: string;
    workerId: string;
  }>();
  const decodedServiceName = decodeURIComponent(serviceName);

  const {
    data: workers = [],
    isLoading,
    update,
  } = useWorkers({
    initialPagination: { currentPage: 1, pageSize: 100 },
    refetchInterval: 5000, // Ensure real-time updates
  });

  // Find the specific worker
  const worker = workers.find((w) => w.metadata.id === workerId);
  // Cast worker to include additional properties
  const workerDetails = worker as unknown as typeof worker & WorkerDetails;

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

  const handleStopWorker = async () => {
    if (!worker) {
      return;
    }

    try {
      await update.mutateAsync({
        workerId: worker.metadata.id,
        // In a real implementation, we might use a different API call to terminate the worker
        // For now, we're just using isPaused as that's what's supported by the API
        data: { isPaused: true },
      });
    } catch (error) {
      console.error('Failed to stop worker:', error);
    }
  };

  // Helper function to format dates
  const formatDate = (dateString?: string) => {
    if (!dateString) {
      return 'Never';
    }
    return new Date(dateString).toLocaleString();
  };

  // Helper function to get the correct status badge
  const getStatusBadge = (status?: string) => {
    if (!status) {
      return <Badge variant="outline">Unknown</Badge>;
    }

    switch (status) {
      case 'ACTIVE':
        return (
          <Badge
            variant="outline"
            className="bg-green-50 text-green-700 border-green-200"
          >
            Active
          </Badge>
        );
      case 'INACTIVE':
        return (
          <Badge
            variant="outline"
            className="bg-red-50 text-red-700 border-red-200"
          >
            Inactive
          </Badge>
        );
      case 'PAUSED':
        return (
          <Badge
            variant="outline"
            className="bg-yellow-50 text-yellow-700 border-yellow-200"
          >
            Paused
          </Badge>
        );
      default:
        return <Badge variant="outline">{status}</Badge>;
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

  if (isLoading) {
    return (
      <div className="flex flex-col items-center justify-center h-96">
        <RefreshCw className="h-10 w-10 animate-spin mb-4 text-primary" />
        <p className="text-lg">Loading worker details...</p>
      </div>
    );
  }

  if (!worker) {
    return (
      <div className="flex flex-1 flex-col gap-4 p-4 pt-0">
        <div className="flex items-center mb-4">
          <Link
            to={`/services/${encodeURIComponent(decodedServiceName)}`}
            className="mr-2"
          >
            <Button variant="ghost" size="sm">
              <ChevronLeft className="h-4 w-4 mr-1" />
              Back to Service
            </Button>
          </Link>
        </div>
        <div className="p-6 bg-background rounded-lg border shadow-sm">
          <h1 className="text-2xl font-bold mb-2">Worker Not Found</h1>
          <p className="text-muted-foreground mb-4">
            The worker you are looking for could not be found.
          </p>
          <div>
            <p>Worker ID: {workerId}</p>
            <p>Service: {decodedServiceName}</p>
          </div>
          <div className="mt-6">
            <Button asChild>
              <Link to={`/services/${encodeURIComponent(decodedServiceName)}`}>
                Return to Service
              </Link>
            </Button>
          </div>
        </div>
      </div>
    );
  }

  return (
    <div className="flex flex-1 flex-col gap-4 p-4 pt-0">
      <div className="flex items-center mb-4">
        <Link
          to={`/services/${encodeURIComponent(decodedServiceName)}`}
          className="mr-2"
        >
          <Button variant="ghost" size="sm">
            <ChevronLeft className="h-4 w-4 mr-1" />
            Back to Service
          </Button>
        </Link>
      </div>

      <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
        {/* Main worker info */}
        <Card className="md:col-span-2">
          <CardHeader>
            <div className="flex justify-between items-start">
              <div>
                <CardTitle className="flex items-center">
                  {worker.name}
                  <span className="ml-2">
                    <WorkerStatusBadge
                      status={worker.status}
                      variant="outline"
                    />
                  </span>
                </CardTitle>
                <CardDescription>
                  Worker ID: {worker.metadata.id}
                </CardDescription>
              </div>
              <div className="flex gap-2">
                <div className="flex items-center gap-2">
                  <WorkerStatusBadge status={worker.status} variant="outline" />
                  <Button
                    size="sm"
                    variant="outline"
                    onClick={() => handlePauseWorker()}
                    disabled={worker.status === 'PAUSED'}
                  >
                    <Pause className="h-4 w-4 mr-1" />
                    Pause
                  </Button>
                  <Button
                    size="sm"
                    variant="outline"
                    onClick={() => handleResumeWorker()}
                    disabled={worker.status === 'ACTIVE'}
                  >
                    <Play className="h-4 w-4 mr-1" />
                    Resume
                  </Button>
                  <Button
                    size="sm"
                    variant="outline"
                    className="text-red-600"
                    onClick={() => handleStopWorker()}
                    disabled={worker.status === 'INACTIVE'}
                  >
                    <StopCircle className="h-4 w-4 mr-1" />
                    Stop
                  </Button>
                </div>
              </div>
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
                        {formatDate(worker.lastHeartbeatAt)}
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
                        {formatDate(worker.metadata.createdAt)}
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
                      {formatDate(worker.metadata.createdAt)}
                    </span>
                  </div>
                  <div className="flex justify-between items-center">
                    <span className="text-sm">Last Updated</span>
                    <span className="text-sm">
                      {formatDate(worker.metadata.updatedAt)}
                    </span>
                  </div>
                  <div className="flex justify-between items-center">
                    <span className="text-sm">Last Connected</span>
                    <span className="text-sm">
                      {formatDate(workerDetails.lastListenerEstablished)}
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
        <Card className="md:col-span-3">
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
