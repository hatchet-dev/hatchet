import { RunDetailSheetSerializableProps } from '@/next/pages/authenticated/dashboard/runs/detail-sheet/run-detail-sheet';
import { WorkerDetailsProps } from '../pages/authenticated/dashboard/workers/components/worker-details';


export interface SideSheet {
  isExpanded: boolean;
  openProps?: OpenSheetProps;
}

export type OpenSheetProps =
  | {
      type: 'task-detail';
      props: RunDetailSheetSerializableProps;
    }
  | {
      type: 'worker-detail';
      props: WorkerDetailsProps;
    };




