import { ConfirmDialog } from '@/components/v1/molecules/confirm-dialog';
import { Loading } from '@/components/v1/ui/loading';
import { relativeDate } from '@/lib/utils';
import { TriggerRunModal } from '@/next/components/runs/trigger-run-modal';
import { Badge } from '@/next/components/ui/badge';
import { Button } from '@/next/components/ui/button';
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from '@/next/components/ui/dropdown-menu';
import { Separator } from '@/next/components/ui/separator';
import {
  Tabs,
  TabsContent,
  TabsList,
  TabsTrigger,
} from '@/next/components/ui/tabs';
import {
  useWorkflowDetails,
  WorkflowDetailsProvider,
} from '@/next/hooks/use-workflow-details';
import { Square3Stack3DIcon } from '@heroicons/react/24/outline';
import { useState } from 'react';
import { useParams } from 'react-router-dom';
import WorkflowGeneralSettings from './settings';
import { RunsProvider } from '@/next/hooks/use-runs';
import { RunsTable } from '@/next/components/runs/runs-table/runs-table';
import { V1TaskStatus } from '@/lib/api';

export default function WorkflowDetailPage() {
  const { workflowId: workflowIdRaw } = useParams<{
    workflowId: string;
  }>();
  return (
    <WorkflowDetailsProvider workflowId={workflowIdRaw || ''}>
      <WorkflowDetailPageContent workflowId={workflowIdRaw || ''} />
    </WorkflowDetailsProvider>
  );
}

function WorkflowDetailPageContent({ workflowId }: { workflowId: string }) {
  const [triggerWorkflow, setTriggerWorkflow] = useState(false);
  const [showDeleteModal, setShowDeleteModal] = useState(false);

  const {
    workflow,
    workflowVersion,
    pauseWorkflow,
    unpauseWorkflow,
    deleteWorkflow,
    workflowIsLoading,
    workflowVersionIsLoading,
    hasGithubIntegration,
    currentVersion,
    isDeleting,
  } = useWorkflowDetails();

  if (workflowIsLoading || !workflow) {
    return <Loading />;
  }

  return (
    <div className="flex-grow h-full w-full">
      <div className="mx-auto py-8 px-4 sm:px-6 lg:px-8">
        <div className="flex flex-row justify-between items-center">
          <div className="flex flex-row gap-4 items-center">
            <Square3Stack3DIcon className="h-6 w-6 text-foreground mt-1" />
            <h2 className="text-2xl font-bold leading-tight text-foreground">
              {workflow.name}
            </h2>
            {currentVersion && (
              <Badge className="text-sm mt-1" variant="outline">
                {currentVersion}
              </Badge>
            )}
            <DropdownMenu>
              <DropdownMenuTrigger>
                {workflow.isPaused ? (
                  <Badge
                    variant="secondary" // TODO: This should be `inProgress`
                    className="px-2"
                    onClick={() => {
                      unpauseWorkflow();
                    }}
                  >
                    Paused
                  </Badge>
                ) : (
                  <Badge
                    variant="default" // TODO: This should be `successful`
                    className="px-2"
                    onClick={() => {
                      pauseWorkflow();
                    }}
                  >
                    Active
                  </Badge>
                )}
              </DropdownMenuTrigger>
              <DropdownMenuContent>
                <DropdownMenuItem>
                  {workflow.isPaused ? (
                    <div
                      onClick={() => {
                        unpauseWorkflow();
                      }}
                    >
                      Unpause runs
                    </div>
                  ) : (
                    <div
                      onClick={() => {
                        pauseWorkflow();
                      }}
                    >
                      Pause runs
                    </div>
                  )}
                </DropdownMenuItem>
              </DropdownMenuContent>
            </DropdownMenu>
          </div>
          {/* <WorkflowTags tags={workflow.tags || []} /> */}
          <div className="flex flex-row gap-2">
            <Button
              className="text-sm"
              onClick={() => setTriggerWorkflow(true)}
            >
              Trigger Workflow
            </Button>
          </div>
          <TriggerRunModal
            show={triggerWorkflow}
            defaultWorkflowId={workflow.metadata.id}
            onClose={() => setTriggerWorkflow(false)}
          />
        </div>
        <div className="flex flex-row justify-start items-center mt-4">
          <div className="text-sm text-gray-700 dark:text-gray-300">
            Updated{' '}
            {relativeDate(
              workflow.versions && workflow.versions[0].metadata.updatedAt,
            )}
          </div>
        </div>
        {workflow.description && (
          <div className="text-sm text-gray-700 dark:text-gray-300 mt-4">
            {workflow.description}
          </div>
        )}
        <div className="flex flex-row justify-start items-center mt-4"></div>
        <Tabs defaultValue="runs">
          {/* TODO: Add layout */}
          <TabsList>
            {/* TODO: Add variant to tabs */}
            <TabsTrigger value="runs">Runs</TabsTrigger>
            {/* TODO: Add variant to tabs */}
            <TabsTrigger value="settings">Settings</TabsTrigger>
          </TabsList>
          <TabsContent value="runs">
            <h3 className="text-xl font-bold leading-tight text-foreground mt-4">
              Recent Runs
            </h3>
            <Separator className="my-4" />
            <RunsProvider
              initialFilters={{
                workflow_ids: [workflowId],
                statuses: [
                  V1TaskStatus.RUNNING,
                  V1TaskStatus.COMPLETED,
                  V1TaskStatus.FAILED,
                  V1TaskStatus.CANCELLED,
                ],
              }}
            >
              <RunsTable />
            </RunsProvider>
          </TabsContent>
          <TabsContent value="settings">
            <h3 className="text-xl font-bold leading-tight text-foreground mt-4">
              Settings
            </h3>
            <Separator className="my-4" />
            {workflowVersionIsLoading || !workflowVersion ? (
              <Loading />
            ) : (
              <WorkflowGeneralSettings />
            )}
            <Separator className="my-4" />
            {hasGithubIntegration && (
              <div className="hidden">
                <h3 className="hidden text-xl font-bold leading-tight text-foreground mt-8">
                  Deployment Settings
                </h3>
                <Separator className="hidden my-4" />
              </div>
            )}
            <h4 className="text-lg font-bold leading-tight text-foreground mt-8">
              Danger Zone
            </h4>
            <Separator className="my-4" />
            <Button
              variant="destructive"
              className="mt-2"
              onClick={() => {
                setShowDeleteModal(true);
              }}
            >
              Delete Workflow
            </Button>

            <ConfirmDialog
              title={`Delete workflow`}
              description={`Are you sure you want to delete the workflow ${workflow.name}? This action cannot be undone, and will immediately prevent any services running with this workflow from executing steps.`}
              submitLabel={'Delete'}
              onSubmit={function (): void {
                deleteWorkflow();
              }}
              onCancel={function (): void {
                setShowDeleteModal(false);
              }}
              isLoading={isDeleting}
              isOpen={showDeleteModal}
            />
          </TabsContent>
        </Tabs>
      </div>
    </div>
  );
}
