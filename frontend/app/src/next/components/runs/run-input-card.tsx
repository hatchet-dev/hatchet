import {
  Card,
  CardContent,
  CardHeader,
  CardTitle,
} from '@/next/components/ui/card';
import { Code } from '@/next/components/ui/code';

export interface RunInputCardProps {
  input: any;
}

export function RunInputCard({ input }: RunInputCardProps) {
  return (
    <Card>
      <CardHeader className="py-3 px-4">
        <CardTitle className="text-sm font-medium">Input</CardTitle>
      </CardHeader>
      <CardContent className="px-4 py-2">
        <Code language="json" value={JSON.stringify(input, null, 2)} />
      </CardContent>
    </Card>
  );
}
