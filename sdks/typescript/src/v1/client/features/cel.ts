import { V1CELDebugResponseStatus } from '@hatchet/clients/rest/generated/data-contracts';
import type { RestApi, RestClientDeps } from '@hatchet/clients/rest/rest-api';
import { rethrowRestError } from '@hatchet/clients/rest/errors';

export type DebugCELInput = {
  expression: string;
  input: Record<string, any>;
  additionalMetadata?: Record<string, string>;
  filterPayload?: Record<string, any>;
};

export type CELEvaluationResult =
  | {
      status: V1CELDebugResponseStatus.SUCCESS;
      output: boolean;
    }
  | {
      status: V1CELDebugResponseStatus.ERROR;
      error: string;
    };

/**
 * Client for debugging CEL expressions.
 */
export class CELClient {
  api: RestApi;
  tenantId: string;

  constructor(client: RestClientDeps) {
    this.api = client.api;
    this.tenantId = client.tenantId;
  }

  /**
   * Debug a CEL expression with the provided input, filter payload, and optional metadata. Useful for testing and validating CEL expressions and debugging issues in production.
   *
   * @param input - The input data to evaluate the CEL expression against.
   * @returns A promise that resolves to the evaluation result, which can either be a success with an output or an error with a message.
   * @throws Will throw an error if the input is invalid or the API call fails.
   */
  async debug(input: DebugCELInput): Promise<CELEvaluationResult> {
    try {
      const response = await this.api.v1CelDebug(this.tenantId, input);

      if (response.data.status === V1CELDebugResponseStatus.ERROR) {
        if (!response.data.error) {
          throw new Error('No error message received from CEL debug API.');
        }

        return {
          status: V1CELDebugResponseStatus.ERROR,
          error: response.data.error,
        };
      }

      if (response.data.output === undefined) {
        throw new Error('No output received from CEL debug API.');
      }

      return {
        status: V1CELDebugResponseStatus.SUCCESS,
        output: response.data.output,
      };
    } catch (err) {
      rethrowRestError(err);
    }
  }
}
