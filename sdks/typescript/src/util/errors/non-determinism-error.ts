import HatchetError from './hatchet-error';

export class NonDeterminismError extends HatchetError {
  taskExternalId: string;
  invocationCount: number;
  nodeId: number;

  constructor(taskExternalId: string, invocationCount: number, nodeId: number, message: string) {
    super(
      `Non-determinism detected in task ${taskExternalId} on invocation ${invocationCount} at node ${nodeId}: ${message}\n` +
        `Check out our documentation for more details on expectations of durable tasks: https://docs.hatchet.run/v1/patterns/mixing-patterns`
    );
    this.name = 'NonDeterminismError';
    this.taskExternalId = taskExternalId;
    this.invocationCount = invocationCount;
    this.nodeId = nodeId;
  }
}
