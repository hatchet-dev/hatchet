import { Button, ReviewedButtonTemp } from '@/components/v1/ui/button';
import { useCurrentTenantId } from '@/hooks/use-tenant';
import { Squares2X2Icon } from '@heroicons/react/24/outline';
import { Link } from 'react-router-dom';

export const WorkflowDefinitionLink = ({
  workflowId,
}: {
  workflowId: string;
}) => {
  const { tenantId } = useCurrentTenantId();

  return (
    <Link
      to={`/tenants/${tenantId}/workflows/${workflowId}`}
      target="_blank"
      rel="noreferrer"
    >
      <ReviewedButtonTemp
        size="sm"
        variant="outline"
        leftIcon={<Squares2X2Icon className="w-4 h-4" />}
      >
        Workflow
      </ReviewedButtonTemp>
    </Link>
  );
};
