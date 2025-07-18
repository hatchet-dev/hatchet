import { columns } from './components/webhook-columns';
import { DataTable } from '@/components/v1/molecules/data-table/data-table';
import { useWebhooks } from './hooks/use-webhooks';

export default function Events() {
  return <EventsTable />;
}

function EventsTable() {
  const { data, isLoading, error } = useWebhooks();

  return (
    <DataTable
      error={error}
      isLoading={isLoading}
      columns={columns()}
      data={data}
      filters={[]}
    />
  );
}
