import { escapeControls } from '../../endpoints/lib/escape';
import { LogLine } from '@/components/v1/cloud/logging/log-search/use-logs';
import { LogViewer } from '@/components/v1/cloud/logging/log-viewer';
import { V1LogLineLevel } from '@/lib/api';
import {
  SandboxProviderEvent,
  SandboxProviderEventKind,
} from '@/lib/api/generated/cloud/data-contracts';
import { useMemo } from 'react';

const LEVEL: Record<SandboxProviderEventKind, V1LogLineLevel> = {
  [SandboxProviderEventKind.HEALTHCHECK_FAILED]: V1LogLineLevel.WARN,
  [SandboxProviderEventKind.UNHEALTHY]: V1LogLineLevel.ERROR,
  [SandboxProviderEventKind.HEALTHY]: V1LogLineLevel.INFO,
  [SandboxProviderEventKind.SANDBOX_CREATED]: V1LogLineLevel.INFO,
  [SandboxProviderEventKind.SANDBOX_CREATE_FAILED]: V1LogLineLevel.ERROR,
  [SandboxProviderEventKind.TURN_FAILED]: V1LogLineLevel.ERROR,
  [SandboxProviderEventKind.SNAPSHOT_FAILED]: V1LogLineLevel.ERROR,
  [SandboxProviderEventKind.SANDBOX_DESTROYED]: V1LogLineLevel.INFO,
};

const LABEL: Record<SandboxProviderEventKind, string> = {
  [SandboxProviderEventKind.HEALTHCHECK_FAILED]: 'Credential check failed',
  [SandboxProviderEventKind.UNHEALTHY]: 'Provider unhealthy',
  [SandboxProviderEventKind.HEALTHY]: 'Provider healthy',
  [SandboxProviderEventKind.SANDBOX_CREATED]: 'Sandbox created',
  [SandboxProviderEventKind.SANDBOX_CREATE_FAILED]: 'Sandbox create failed',
  [SandboxProviderEventKind.TURN_FAILED]: 'Turn failed',
  [SandboxProviderEventKind.SNAPSHOT_FAILED]: 'Snapshot failed',
  [SandboxProviderEventKind.SANDBOX_DESTROYED]: 'Sandbox destroyed',
};

const toLogLine = (event: SandboxProviderEvent): LogLine => {
  const parts = [LABEL[event.kind]];

  if (event.sandboxId) {
    parts.push(event.sandboxId);
  }

  if (event.message) {
    parts.push(escapeControls(event.message));
  }

  return {
    timestamp: event.createdAt,
    level: LEVEL[event.kind],
    line: parts.join(' · '),
    metadata: event.details,
  };
};

export const EventsLog = ({
  events,
  isLoading,
}: {
  events: SandboxProviderEvent[];
  isLoading?: boolean;
}) => {
  const logs = useMemo(() => events.map(toLogLine), [events]);

  return (
    <div className="flex h-[480px] flex-col">
      <LogViewer
        logs={logs}
        isLoading={isLoading}
        showAttempt={false}
        emptyMessage="No events yet. The operator records credential checks, sandbox creations and failures here."
      />
    </div>
  );
};
