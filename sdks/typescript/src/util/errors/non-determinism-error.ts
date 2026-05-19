import HatchetError from './hatchet-error';

export class NonDeterminismError extends HatchetError {
  taskExternalId: string;
  invocationCount: number;
  nodeId: number;

  constructor(taskExternalId: string, invocationCount: number, nodeId: number, message: string) {
    const detail = message
      ? message
      : `Non-determinism detected in task ${taskExternalId} on invocation ${invocationCount} at node ${nodeId}`;
    super(
      `${detail}\n` +
        `Check out our documentation for more details on expectations of durable tasks: https://docs.hatchet.run/v1/patterns`
    );
    this.name = 'NonDeterminismError';
    this.taskExternalId = taskExternalId;
    this.invocationCount = invocationCount;
    this.nodeId = nodeId;
  }
}
