import { Button } from '@/components/ui/button';
import { useTenant } from '@/lib/atoms';
import { Squares2X2Icon } from '@heroicons/react/24/outline';
import { Link } from 'react-router-dom';

export const WorkflowDefinitionLink = ({
  workflowId,
}: {
  workflowId: string;
}) => {
  const { tenant } = useTenant();

  return (
    <Link
      to={`/tenants/${tenant?.metadata.id}/workflows/${workflowId}`}
      target="_blank"
      rel="noreferrer"
    >
      <Button size={'sm'} className="px-2 py-2 gap-2" variant="outline">
        <Squares2X2Icon className="w-4 h-4" />
        Workflow
      </Button>
    </Link>
  );
};
