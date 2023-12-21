import { Icons } from '@/components/ui/icons.tsx';

export function Spinner() {
  return <Icons.spinner className="mr-2 h-4 w-4 animate-spin" />;
}

export function Loading() {
  return (
    <div className="flex flex-row flex-1 w-full h-full">
      <Spinner />
    </div>
  );
}
