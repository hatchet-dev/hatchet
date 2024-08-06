import {
  Tooltip,
  TooltipProvider,
  TooltipTrigger,
} from '@/components/ui/tooltip';
import { SemaphoreSlots } from '@/lib/api';
import { TooltipContent } from '@radix-ui/react-tooltip';

const statusColors = {
  PENDING: 'bg-yellow-500',
  PENDING_ASSIGNMENT: 'bg-yellow-400',
  ASSIGNED: 'bg-blue-500',
  RUNNING: 'bg-green-500',
  SUCCEEDED: 'bg-green-700',
  FAILED: 'bg-red-500',
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
            <TooltipContent side="top" className="tooltip-content">
              <div className="bg-white text-black p-2 rounded shadow-md">
                <div>
                  <strong>ID:</strong> {slot.stepRunId}
                </div>
                <div>
                  <strong>Status:</strong> {slot.status || 'UNDEFINED'}
                </div>
              </div>
            </TooltipContent>
          </Tooltip>
        </TooltipProvider>
      ))}
    </div>
  );
};

export default WorkerSlotGrid;
