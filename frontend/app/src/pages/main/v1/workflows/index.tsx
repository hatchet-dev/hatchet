import { Separator } from '@/components/v1/ui/separator';
import { WorkflowTable } from './components/workflow-table';

export default function Workflows() {
  return (
    <div className="flex-grow h-full w-full">
      <div className="mx-auto py-8 px-4 sm:px-6 lg:px-8">
        <h2 className="text-2xl font-bold leading-tight text-foreground">
          Workflows
        </h2>
        <Separator className="my-4" />
        <WorkflowTable />
      </div>
    </div>
  );
}
