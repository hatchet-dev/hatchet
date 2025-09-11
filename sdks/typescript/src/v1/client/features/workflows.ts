import { Workflow } from '@hatchet/workflow';
import { BaseWorkflowDeclaration, WorkflowDefinition } from '@hatchet/v1';
import { isValidUUID } from '@util/uuid';
import { HatchetClient } from '../client';

export const workflowNameString = (
  workflow: string | Workflow | WorkflowDefinition | BaseWorkflowDeclaration<any, any>
) => {
  if (typeof workflow === 'string') {
    return workflow;
  }
  if (typeof workflow === 'object' && 'id' in workflow) {
    return workflow.id;
  }
  return workflow.name;
};

/**
 * WorkflowsClient is used to list and manage workflows
 */
export class WorkflowsClient {
  api: HatchetClient['api'];
  tenantId: string;
  // Add a cache for workflows
  private workflowCache: Map<string, { workflow: any; expiry: number }> = new Map();
  // Default cache TTL: 5 minutes (in ms)
  private cacheTTL: number = 5 * 60 * 1000;

  constructor(client: HatchetClient, cacheTTL?: number) {
    this.api = client.api;
    this.tenantId = client.tenantId;
    // Allow custom cache TTL if provided
    if (cacheTTL !== undefined) {
      this.cacheTTL = cacheTTL;
    }
  }

  /**
   * Gets the workflow ID from a workflow name, ID, or object.
   * If the input is not a valid UUID, it will look up the workflow by name.
   * @param workflow - The workflow name, ID, or object.
   * @returns The workflow ID as a string.
   */
  async getWorkflowIdFromName(
    workflow: string | Workflow | WorkflowDefinition | BaseWorkflowDeclaration<any, any>
  ): Promise<string> {
    const str = (() => {
      if (typeof workflow === 'string') {
        return workflow;
      }

      if (typeof workflow === 'object' && 'name' in workflow) {
        return workflow.name;
      }

      if (typeof workflow === 'object' && 'id' in workflow) {
        if (!workflow.id) {
          throw new Error('Workflow ID is required');
        }
        return workflow.id;
      }

      throw new Error(
        'Invalid workflow: must be a string, Workflow object, or WorkflowDefinition object'
      );
    })();

    if (!isValidUUID(str)) {
      const wf = await this.get(str);
      if (!wf) {
        throw new Error('Invalid workflow ID: must be a valid UUID');
      }
      return wf.metadata.id;
    }

    return str;
  }

  async get(workflow: string | BaseWorkflowDeclaration<any, any> | Workflow) {
    // Get workflow name string
    const name = workflowNameString(workflow);

    // Check cache first
    const cached = this.workflowCache.get(name);
    if (cached && cached.expiry > Date.now()) {
      return cached.workflow as NonNullable<
        Awaited<ReturnType<typeof this.api.workflowList>>['data']['rows']
      >[number];
    }

    // If not in cache or expired, fetch from API
    try {
      // Since the API only supports listing with a name filter,
      // we'll use the list endpoint with a name filter
      const { data } = await this.api.workflowList(this.tenantId, {
        name,
      });

      if (data && data.rows && data.rows.length > 0) {
        const wf = data.rows[0];

        // Cache the result
        this.workflowCache.set(name, {
          workflow: wf,
          expiry: Date.now() + this.cacheTTL,
        });

        return wf;
      }

      throw new Error(`Workflow with name ${name} not found`);
    } catch (error) {
      // Clear cache on error
      this.workflowCache.delete(name);
      throw error;
    }
  }

  async list(opts?: Parameters<typeof this.api.workflowList>[1]) {
    const { data } = await this.api.workflowList(this.tenantId, opts);
    return data;
  }

  async delete(workflow: string | BaseWorkflowDeclaration<any, any> | Workflow) {
    const name = workflowNameString(workflow);

    try {
      // Get the workflow first to find its ID
      const workflowObj = await this.get(name);

      if (!workflowObj || !workflowObj.metadata || !workflowObj.metadata.id) {
        throw new Error(`Could not find workflow with name ${name}`);
      }

      const { data } = await this.api.workflowDelete(workflowObj.metadata.id);

      // Remove from cache after deletion
      this.workflowCache.delete(name);

      return data;
    } catch (error) {
      // Clear cache on error
      this.workflowCache.delete(name);
      throw error;
    }
  }

  // async isPaused(workflow: string | WorkflowDeclaration<any, any> | Workflow) {
  //   const wf = await this.get(workflow);
  //   return wf.isPaused;
  // }

  // async pause(workflow: string | WorkflowDeclaration<any, any> | Workflow) {
  //   const name = workflowNameString(workflow);

  //   try {
  //     // Get the workflow first to find its ID
  //     const workflowObj = await this.get(name);

  //     if (!workflowObj || !workflowObj.metadata || !workflowObj.metadata.id) {
  //       throw new Error(`Could not find workflow with name ${name}`);
  //     }

  //     const { data } = await this.api.workflowUpdate(workflowObj.metadata.id, {
  //       isPaused: true,
  //     });

  //     // Update cache
  //     if (data) {
  //       this.workflowCache.set(name, {
  //         workflow: data,
  //         expiry: Date.now() + this.cacheTTL,
  //       });
  //     }

  //     return data;
  //   } catch (error) {
  //     // Clear cache on error
  //     this.workflowCache.delete(name);
  //     throw error;
  //   }
  // }

  // async unpause(workflow: string | WorkflowDeclaration<any, any> | Workflow) {
  //   const name = workflowNameString(workflow);

  //   try {
  //     // Get the workflow first to find its ID
  //     const workflowObj = await this.get(name);

  //     if (!workflowObj || !workflowObj.metadata || !workflowObj.metadata.id) {
  //       throw new Error(`Could not find workflow with name ${name}`);
  //     }

  //     const { data } = await this.api.workflowUpdate(workflowObj.metadata.id, {
  //       isPaused: false,
  //     });

  //     // Update cache
  //     if (data) {
  //       this.workflowCache.set(name, {
  //         workflow: data,
  //         expiry: Date.now() + this.cacheTTL,
  //       });
  //     }

  //     return data;
  //   } catch (error) {
  //     // Clear cache on error
  //     this.workflowCache.delete(name);
  //     throw error;
  //   }
  // }
}
