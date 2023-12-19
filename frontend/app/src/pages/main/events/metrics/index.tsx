import BrushChart from '@/components/molecules/brush-chart/brush-chart';
import { Separator } from '@/components/ui/separator';
import ParentSize from '@visx/responsive/lib/components/ParentSize';

export default function EventMetrics() {
  return (
    <div className="flex-grow h-full w-full">
      <div className="mx-auto max-w-7xl py-8 px-4 sm:px-6 lg:px-8">
        <h2 className="text-2xl font-bold leading-tight text-foreground">
          Event Metrics
        </h2>
        <div className="text-sm text-muted-foreground my-4">
          Whoa! You shouldn't have ended up here, but if you're curious, this is
          an event metrics view we've been working on using the fantastic visx
          library. Here's a mockup, now please leave.
        </div>
        <Separator className="my-4" />
        <div className="flex flex-col justify-between items-center">
          <div className="w-full flex-grow h-[450px]">
            <ParentSize>
              {({ width, height }) => (
                <BrushChart width={width} height={height} />
              )}
            </ParentSize>
          </div>
        </div>
      </div>
    </div>
  );
}
