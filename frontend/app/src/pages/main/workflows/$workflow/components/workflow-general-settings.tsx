import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import { WorkflowVersion } from '@/lib/api';

export default function WorkflowGeneralSettings({
  workflow,
}: {
  workflow: WorkflowVersion;
}) {
  return (
    <>
      <h3 className="text-lg font-semibold mb-4">Concurrency</h3>
      <ConcurrencySettings workflow={workflow} />
    </>
  );
}

function ConcurrencySettings({ workflow }: { workflow: WorkflowVersion }) {
  if (!workflow.concurrency) {
    return (
      <div className="text-[0.8rem] text-muted-foreground">
        There are no concurrency settings for this workflow.
      </div>
    );
  }

  return (
    <div className="flex flex-col gap-2">
      <Label>Max runs</Label>
      <Input
        disabled
        placeholder="shadcn"
        value={workflow.concurrency?.maxRuns}
      />
      <div className="text-[0.8rem] text-muted-foreground">
        The maximum number of concurrency runs.
      </div>
      <Label className="mt-4">Concurrency strategy</Label>
      <Input
        disabled
        placeholder="shadcn"
        value={workflow.concurrency?.limitStrategy}
      />
      <div className="text-[0.8rem] text-muted-foreground">
        The strategy to use when the maximum number of concurrency runs is
        reached.
      </div>
    </div>
  );
}
