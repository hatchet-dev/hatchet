import HatchetError from '@util/errors/hatchet-error';
import { retrier, type RetrierConfig } from '@hatchet/util/retrier';
import type { Logger } from '@hatchet/util/logger/logger';
import type { V1AdminRpc } from '@hatchet/clients/admin/rpc';
import type {
  CreateWorkflowVersionRequest,
  CreateWorkflowVersionResponse,
} from '@hatchet/protoc/v1/workflows';
import type { BaseWorkflowDeclaration, WorkflowDefinition } from '@hatchet/v1/declaration';
import type { InputType, OutputType } from '@hatchet/v1/types';
import { workflowToProto } from '@hatchet/v1/client/worker/workflow-proto';

export interface WorkflowsClientDeps {
  rpc: V1AdminRpc;
  logger: Logger;
  namespace?: string;
  retrier?: RetrierConfig;
}

/** Registers workflow definitions with the engine. */
export class WorkflowsClient {
  constructor(private readonly deps: WorkflowsClientDeps) {}

  /**
   * Creates or updates a workflow. A declaration from `/edge` (or its definition) is
   * converted with the client's namespace the way a worker registers it; a
   * `CreateWorkflowVersionRequest` is sent as given.
   */
  async put<I extends InputType, O extends OutputType>(
    workflow: CreateWorkflowVersionRequest | WorkflowDefinition | BaseWorkflowDeclaration<I, O>
  ): Promise<CreateWorkflowVersionResponse> {
    const request = isWireRequest(workflow)
      ? workflow
      : workflowToProto(workflow, { namespace: this.deps.namespace });

    try {
      return await retrier(
        () => this.deps.rpc.putWorkflow(request),
        this.deps.logger,
        this.deps.retrier
      );
    } catch (e: unknown) {
      throw new HatchetError(e instanceof Error ? e.message : String(e));
    }
  }
}

function isWireRequest<I extends InputType, O extends OutputType>(
  workflow: CreateWorkflowVersionRequest | WorkflowDefinition | BaseWorkflowDeclaration<I, O>
): workflow is CreateWorkflowVersionRequest {
  return 'tasks' in workflow && Array.isArray(workflow.tasks);
}
