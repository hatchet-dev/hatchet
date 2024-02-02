import { Card, CardContent } from '@/components/ui/card';
import { Label } from '@/components/ui/label';
import { Step } from '@/lib/api';
import { memo } from 'react';
import { Handle, Position } from 'reactflow';

// eslint-disable-next-line react/display-name
export default memo(
  ({
    data,
  }: {
    data: {
      step: Step;
      variant: 'default' | 'input_only' | 'output_only';
    };
  }) => {
    const variant = data.variant;

    return (
      <Card className="p-3 cursor-pointer bg-[#020817]">
        {(variant == 'default' || variant == 'input_only') && (
          <Handle
            type="target"
            position={Position.Left}
            style={{ visibility: 'hidden' }}
            isConnectable={false}
          />
        )}
        <CardContent className="p-0">
          <div className="font-bold text-sm">{data.step.readableId}</div>
          <Label className="cursor-pointer">
            <span className="font-bold mr-2 text-xs">Timeout</span>
            <span className="text-gray-500 font-medium text-xs">
              {data.step.timeout || '300s'}
            </span>
          </Label>
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
  },
);
