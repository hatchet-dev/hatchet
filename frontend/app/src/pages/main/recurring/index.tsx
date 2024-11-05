import { Separator } from '@/components/ui/separator';
import { CronsTable } from './components/recurring-table';

export default function Crons() {
  return (
    <div className="flex-grow h-full w-full">
      <div className="mx-auto max-w-7xl py-8 px-4 sm:px-6 lg:px-8">
        <h2 className="text-2xl font-bold leading-tight text-foreground">
          Cron Jobs
        </h2>
        <Separator className="my-4" />
        <CronsTable />
      </div>
    </div>
  );
}
