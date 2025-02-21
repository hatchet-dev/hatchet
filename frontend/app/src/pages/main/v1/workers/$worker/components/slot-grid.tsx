import RelativeDate from '@/components/v1/molecules/relative-date';
import {
  Tooltip,
  TooltipProvider,
  TooltipTrigger,
  TooltipContent,
} from '@/components/v1/ui/tooltip';
import { SemaphoreSlots } from '@/lib/api';
import { Link } from 'react-router-dom';

const statusColors = {
  PENDING: 'bg-yellow-500',
  PENDING_ASSIGNMENT: 'bg-yellow-400',
  ASSIGNED: 'bg-blue-500',
  RUNNING: 'bg-green-500',
  SUCCEEDED: 'bg-green-700',
  FAILED: 'bg-red-500',
  CANCELLING: 'bg-yellow-500',
  CANCELLED: 'bg-gray-500',
  UNDEFINED: 'bg-gray-300', // Default color for undefined status
};

interface WorkerSlotGridProps {
  slots?: SemaphoreSlots[];
}

const WorkerSlotGrid: React.FC<WorkerSlotGridProps> = ({ slots = [] }) => {
  return (
    <div className="flex flex-wrap gap-0.5">
      {slots?.map((slot, index) => (
        <TooltipProvider key={index}>
          <Tooltip>
            <TooltipTrigger>
              <div
                className={`h-4 w-4 ${
                  slot.status
                    ? statusColors[slot.status]
                    : statusColors.UNDEFINED
                } cursor-pointer`}
              >
                <span className="sr-only">{slot.stepRunId}</span>
              </div>
            </TooltipTrigger>
            <TooltipContent side="top">
              {slot.status ? (
                <>
                  <div>
                    <Link to={'/workflow-runs/' + slot.workflowRunId}>
                      <div className="pl-0 cursor-pointer hover:underline min-w-fit whitespace-nowrap">
                        {slot.actionId}:{slot.workflowRunId?.split('-')[0]}
                      </div>
                    </Link>
                  </div>
                  <div>Status {slot.status || 'UNDEFINED'}</div>
                  {slot.startedAt && (
                    <div>
                      Started <RelativeDate date={slot.startedAt} />
                    </div>
                  )}
                  {slot.timeoutAt && (
                    <div>
                      Timeout <RelativeDate date={slot.timeoutAt} />
                    </div>
                  )}
                </>
              ) : (
                <>Waiting for run</>
              )}
            </TooltipContent>
          </Tooltip>
        </TooltipProvider>
      ))}
    </div>
  );
};

export default WorkerSlotGrid;
