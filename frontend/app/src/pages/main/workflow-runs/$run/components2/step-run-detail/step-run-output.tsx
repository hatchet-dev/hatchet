import RelativeDate from '@/components/molecules/relative-date';
import { Alert, AlertTitle } from '@/components/ui/alert';
import { StepRun, StepRunStatus, WorkflowRunShape } from '@/lib/api';
import React from 'react';

const readableReason = (reason?: string): string => {
  return reason
    ? reason
        .toLowerCase()
        .split('_')
        .map((word) => word.charAt(0).toUpperCase() + word.slice(1))
        .join(' ')
    : '';
};

type StepRunOutputProps = {
  stepRun: StepRun;
  workflowRun: WorkflowRunShape;
};

const StepRunOutputCancelled = ({ stepRun }: StepRunOutputProps) => {
  return (
    <Alert variant="warn">
      <AlertTitle>
        {readableReason(stepRun.cancelledReason) || 'Cancelled'}
      </AlertTitle>
      <pre>{stepRun.cancelledAt}</pre>
      <RelativeDate date={stepRun.cancelledAt} />
      <pre>{stepRun.cancelledError}</pre>
    </Alert>
  );
};

const StepRunOutputPending = ({ stepRun }: StepRunOutputProps) => {
  return (
    <Alert variant="info">
      <AlertTitle>Pending</AlertTitle>
      <pre>Waiting to start...</pre>
      {stepRun.parents}
    </Alert>
  );
};

const StepRunOutputPendingAssignment = ({ stepRun }: StepRunOutputProps) => {
  return (
    <Alert variant="info">
      <AlertTitle>Pending Assignment</AlertTitle>
      <pre>Waiting for assignment...</pre>
    </Alert>
  );
};

const StepRunOutputAssigned = ({ stepRun }: StepRunOutputProps) => {
  return (
    <Alert variant="info">
      <AlertTitle>Assigned</AlertTitle>
      <pre>Step has been assigned and is ready to run</pre>
    </Alert>
  );
};

const StepRunOutputRunning = ({ stepRun }: StepRunOutputProps) => {
  return (
    <Alert variant="info">
      <AlertTitle>Running</AlertTitle>
      <pre>Step is currently executing...</pre>
      <pre>Started at: {stepRun.startedAt}</pre>
    </Alert>
  );
};

const StepRunOutputSucceeded = ({ stepRun }: StepRunOutputProps) => {
  return (
    <Alert variant="success">
      <AlertTitle>Succeeded</AlertTitle>
      <pre>Step completed successfully</pre>
      <pre>Finished at: {stepRun.finishedAt}</pre>
    </Alert>
  );
};

const StepRunOutputFailed = ({ stepRun }: StepRunOutputProps) => {
  return (
    <Alert variant="error">
      <AlertTitle>Failed</AlertTitle>
      <pre>Error: {stepRun.error}</pre>
      <pre>Finished at: {stepRun.finishedAt}</pre>
    </Alert>
  );
};

const StepRunOutputCancelling = ({ stepRun }: StepRunOutputProps) => {
  return (
    <Alert variant="warn">
      <AlertTitle>Cancelling</AlertTitle>
      <pre>Step is being cancelled...</pre>
    </Alert>
  );
};

const OUTPUT_STATE_MAP: Record<StepRunStatus, React.FC<StepRunOutputProps>> = {
  [StepRunStatus.CANCELLED]: StepRunOutputCancelled,
  [StepRunStatus.PENDING]: StepRunOutputPending,
  [StepRunStatus.PENDING_ASSIGNMENT]: StepRunOutputPendingAssignment,
  [StepRunStatus.ASSIGNED]: StepRunOutputAssigned,
  [StepRunStatus.RUNNING]: StepRunOutputRunning,
  [StepRunStatus.SUCCEEDED]: StepRunOutputSucceeded,
  [StepRunStatus.FAILED]: StepRunOutputFailed,
  [StepRunStatus.CANCELLING]: StepRunOutputCancelling,
};

const StepRunOutput: React.FC<StepRunOutputProps> = (props) => {
  const Component = OUTPUT_STATE_MAP[props.stepRun.status];
  return <Component {...props} />;
};

export default StepRunOutput;
