import { Button } from '@/components/v1/ui/button';
import { WorkflowRunShape } from '@/lib/api';
import { preferredWorkflowRunViewAtom } from '@/lib/atoms';
import { type ViewOptions } from '@/lib/atoms';
import { useAtom } from 'jotai';
import { BiExitFullscreen, BiExpand } from 'react-icons/bi';
import { useWorkflowDetails } from '../../hooks/workflow-details';

const ToggleIcon = ({ view }: { view: ViewOptions | undefined }) => {
  switch (view) {
    case 'graph':
      return <BiExpand />;
    case 'minimap':
      return <BiExitFullscreen />;
    case undefined:
      return null;
    default:
      // eslint-disable-next-line no-case-declarations
      const exhaustiveCheck: never = view;
      throw new Error(`Unhandled case: ${exhaustiveCheck}`);
  }
};

export const ViewToggle = () => {
  const [view, setView] = useAtom(preferredWorkflowRunViewAtom);
  const otherView = view === 'graph' ? 'minimap' : 'graph';

  const { shape, isLoading, isError } = useWorkflowDetails();

  if (isLoading || isError) {
    return null;
  }

  // only render if there are at least two dependent steps, otherwise the view toggle is not needed
  if (!shape.some((t) => t.childrenStepIds.length > 0)) {
    return null;
  }

  return (
    <div className="sticky ml-auto mt-auto bottom-2 right-2 z-20">
      <Button variant="outline" size="icon" onClick={() => setView(otherView)}>
        <ToggleIcon view={view} />
      </Button>
    </div>
  );
};

export function hasChildSteps(shape: WorkflowRunShape) {
  return shape.jobRuns?.some((jobRun) => {
    return jobRun.job?.steps.some((step) => {
      return step?.parents?.length;
    });
  });
}
