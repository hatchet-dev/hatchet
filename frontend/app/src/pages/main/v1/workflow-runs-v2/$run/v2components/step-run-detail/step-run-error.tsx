import LoggingComponent from '@/components/v1/cloud/logging/logs';

export default function StepRunError({ text }: { text: string }) {
  return (
    <div className="p-0 w-full min-w-[500px] rounded-md h-[400px] overflow-y-auto">
      <LoggingComponent
        logs={[
          {
            line: text,
          },
        ]}
        onTopReached={() => {}}
        onBottomReached={() => {}}
        autoScroll={false}
      />
    </div>
  );
}
