import { Card, CardContent } from '@/components/v1/ui/card';
import { Step } from '@/lib/api';
import { cn } from '@/lib/utils';
import { memo } from 'react';
import { Handle, Position } from 'reactflow';

export interface StepNodeProps {
  step: Step;
  variant: 'default' | 'input_only' | 'output_only';
}

// eslint-disable-next-line react/display-name
export default memo(({ data }: { data: StepNodeProps }) => {
  const variant = data.variant;

  return (
    <Card
      className={cn(
        'step-run-card p-3 cursor-pointer bg-[#020817] shadow-[0_1000px_0_0_hsl(0_0%_20%)_inset] transition-colors duration-200',
      )}
    >
      <span className="step-run-backdrop absolute inset-[1px] rounded-full bg-background transition-colors duration-200" />
      {(variant == 'default' || variant == 'input_only') && (
        <Handle
          type="target"
          position={Position.Left}
          style={{ visibility: 'hidden' }}
          isConnectable={false}
        />
      )}
      <CardContent className="p-0 z-10 bg-background">
        <div className="flex flex-row justify-between gap-2 items-center">
          <div className="font-bold text-sm">
            {data.step?.readableId || data.step.metadata.id}
          </div>
        </div>
        <div className="flex flex-col mt-1"></div>
      </CardContent>
      {(variant == 'default' || variant == 'output_only') && (
        <Handle
          type="source"
          position={Position.Right}
          style={{ visibility: 'hidden' }}
          isConnectable={false}
        />
      )}
    </Card>
  );
});
