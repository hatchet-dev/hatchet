import {
  Card,
  CardHeader,
  CardTitle,
  CardDescription,
} from '@/next/components/ui/card';

interface ContainerConfigProps {
  value: any;
  onChange: (value: any) => void;
  type: 'create' | 'update';
  actions?: React.ReactNode;
}

export function ContainerConfig({
  value,
  onChange,
  type,
  actions,
}: ContainerConfigProps) {
  return (
    <Card variant={type === 'update' ? 'borderless' : 'default'}>
      <CardHeader>
        <CardTitle>Container Configuration</CardTitle>
        <CardDescription>
          Configure the container settings for your worker service.
        </CardDescription>
      </CardHeader>
      {actions}
    </Card>
  );
}
