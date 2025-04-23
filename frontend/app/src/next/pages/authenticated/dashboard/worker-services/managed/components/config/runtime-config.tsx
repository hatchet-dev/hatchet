import React from 'react';
import {
  Card,
  CardHeader,
  CardTitle,
  CardDescription,
  CardContent,
} from '@/components/ui/card';

interface RuntimeConfigProps {
  actions?: React.ReactNode;
}

export function RuntimeConfig({ actions }: RuntimeConfigProps) {
  return (
    <Card>
      <CardHeader>
        <CardTitle>Runtime Configuration</CardTitle>
        <CardDescription>
          Configure runtime settings for your worker service.
        </CardDescription>
      </CardHeader>
      <CardContent>
        <div>TODO: RuntimeConfig</div>
      </CardContent>
      {actions}
    </Card>
  );
}
