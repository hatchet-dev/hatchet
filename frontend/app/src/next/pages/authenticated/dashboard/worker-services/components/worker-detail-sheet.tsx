import { InfoSheet } from '@/next/components/ui/info-sheet';
import { WorkerId } from './worker-id';
import { CpuIcon } from 'lucide-react';
import { WorkerDetails } from './worker-details';

interface WorkerDetailSheetProps {
  isOpen: boolean;
  onClose: () => void;
  serviceName: string;
  workerId: string;
}

export function WorkerDetailSheet({
  isOpen,
  onClose,
  workerId,
}: WorkerDetailSheetProps) {
  return (
    <InfoSheet
      isOpen={isOpen}
      onClose={onClose}
      title={
        <div className="flex items-center gap-2">
          <CpuIcon className="h-4 w-4" />
          <WorkerId workerId={workerId} />
        </div>
      }
    >
      <WorkerDetails workerId={workerId} />
    </InfoSheet>
  );
}
